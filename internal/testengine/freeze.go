package testengine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var frozenSchemaPaths = []string{
	"config/schemas/cli-report.schema.json",
	"config/schemas/dependency-governance.schema.json",
	"config/schemas/kit-manifest.schema.json",
	"config/schemas/kit-registry.schema.json",
	"config/schemas/mcp-component.schema.json",
	"config/schemas/mcp-registry.schema.json",
}

func checkFrozenSchemas(repo string) error {
	return requirePaths(repo, frozenSchemaPaths...)
}

func checkUniqueProductionType(repo, root, typeName string) error {
	base := filepath.Join(repo, filepath.FromSlash(root))
	pattern := regexp.MustCompile(`(?m)^\s*type\s+` + regexp.QuoteMeta(typeName) + `\s+struct\s*\{`)
	matches := []string{}
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for range pattern.FindAll(raw, -1) {
			rel, relErr := filepath.Rel(repo, path)
			if relErr != nil {
				return relErr
			}
			matches = append(matches, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(matches)
	if len(matches) != 1 {
		return fmt.Errorf("%s must contain exactly one production type %s struct; found %d in %v", root, typeName, len(matches), matches)
	}
	return nil
}
