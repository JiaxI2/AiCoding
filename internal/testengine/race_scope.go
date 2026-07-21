package testengine

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const impactPolicyPath = "config/impact-policy.json"

type impactPolicy struct {
	SchemaVersion int `json:"schemaVersion"`
	RaceScope     struct {
		Packages []string `json:"packages"`
		Reason   string   `json:"reason"`
	} `json:"raceScope"`
}

func raceTestCommand(cfg Config) []string {
	targets := []string{"./..."}
	if cfg.Profile == ProfileFull {
		if policy, err := loadImpactPolicy(cfg.Repo); err == nil {
			targets = make([]string, 0, len(policy.RaceScope.Packages))
			for _, packageDir := range policy.RaceScope.Packages {
				if packageDir == "." {
					targets = append(targets, ".")
				} else {
					targets = append(targets, "./"+packageDir)
				}
			}
		}
	}
	return append([]string{"go", "test", "-race"}, targets...)
}

func loadImpactPolicy(repo string) (impactPolicy, error) {
	var policy impactPolicy
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(impactPolicyPath)))
	if err != nil {
		return policy, fmt.Errorf("read %s: %w", impactPolicyPath, err)
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		return policy, fmt.Errorf("decode %s: %w", impactPolicyPath, err)
	}
	if policy.SchemaVersion != 1 {
		return policy, fmt.Errorf("%s schemaVersion must be 1", impactPolicyPath)
	}
	if len(policy.RaceScope.Packages) == 0 || strings.TrimSpace(policy.RaceScope.Reason) == "" {
		return policy, fmt.Errorf("%s raceScope requires packages and reason", impactPolicyPath)
	}
	seen := map[string]bool{}
	for index, packageDir := range policy.RaceScope.Packages {
		clean := filepath.ToSlash(filepath.Clean(packageDir))
		if clean != packageDir || filepath.IsAbs(packageDir) || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "./") {
			return policy, fmt.Errorf("%s raceScope package is not a normalized repository-relative directory: %q", impactPolicyPath, packageDir)
		}
		if seen[clean] {
			return policy, fmt.Errorf("%s raceScope package is duplicated: %q", impactPolicyPath, packageDir)
		}
		if index > 0 && policy.RaceScope.Packages[index-1] > packageDir {
			return policy, fmt.Errorf("%s raceScope packages must be sorted", impactPolicyPath)
		}
		seen[clean] = true
	}
	return policy, nil
}

func checkRaceScope(repo string) error {
	policy, err := loadImpactPolicy(repo)
	if err != nil {
		return err
	}
	registered := map[string]bool{}
	for _, packageDir := range policy.RaceScope.Packages {
		registered[packageDir] = true
		info, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(packageDir)))
		if statErr != nil || !info.IsDir() {
			return fmt.Errorf("raceScope package directory is missing: %s", packageDir)
		}
	}

	detected, err := concurrentGoPackages(repo)
	if err != nil {
		return err
	}
	missing := make([]string, 0)
	for packageDir, marker := range detected {
		if !registered[packageDir] {
			missing = append(missing, packageDir+" ("+marker+")")
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		return fmt.Errorf("GO-007 raceScope is missing concurrent package(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func concurrentGoPackages(repo string) (map[string]string, error) {
	packages := map[string]string{}
	err := filepath.WalkDir(repo, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == repo {
				return nil
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" || name == "testdata" || name == "node_modules" || name == "bin" || name == "dist" || name == "test-results" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			} else if !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse Go source %s: %w", path, err)
		}
		marker := ""
		for _, imported := range parsed.Imports {
			if imported.Path != nil && imported.Path.Value == `"sync"` {
				marker = "sync import"
				break
			}
		}
		if marker == "" {
			ast.Inspect(parsed, func(node ast.Node) bool {
				if marker != "" {
					return false
				}
				switch node.(type) {
				case *ast.GoStmt:
					marker = "go statement"
				case *ast.ChanType:
					marker = "channel type"
				}
				return true
			})
		}
		if marker == "" {
			return nil
		}
		relative, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		packageDir := filepath.ToSlash(filepath.Dir(relative))
		if _, exists := packages[packageDir]; !exists {
			packages[packageDir] = filepath.ToSlash(relative) + ": " + marker
		}
		return nil
	})
	return packages, err
}
