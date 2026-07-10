package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

const repositoryLayoutPath = "config/repository-layout.json"

type layoutConfig struct {
	SchemaVersion    int                 `json:"schemaVersion"`
	Root             layoutRoot          `json:"root"`
	DirectoryClasses map[string][]string `json:"directoryClasses"`
	Documentation    layoutDocumentation `json:"documentation"`
	Prompts          layoutPrompts       `json:"prompts"`
	TestFixtures     layoutTestFixtures  `json:"testFixtures"`
	Generated        layoutGenerated     `json:"generatedArtifacts"`
	Skills           layoutSkills        `json:"skills"`
}

type layoutRoot struct {
	AllowDirectories     []string `json:"allowDirectories"`
	ForbiddenDirectories []string `json:"forbiddenDirectories"`
	TransientDirectories []string `json:"transientDirectories"`
}

type layoutDocumentation struct {
	Root                string   `json:"root"`
	AllowedRootFiles    []string `json:"allowedRootFiles"`
	AllowedOutsideRoots []string `json:"allowedOutsideRoots"`
}

type layoutPrompts struct {
	AllowedRoots  []string `json:"allowedRoots"`
	ForbiddenRoot string   `json:"forbiddenRoot"`
}

type layoutTestFixtures struct {
	Root          string `json:"root"`
	ForbiddenRoot string `json:"forbiddenRoot"`
}

type layoutGenerated struct {
	Directories []string `json:"directories"`
	Extensions  []string `json:"extensions"`
}

type layoutSkills struct {
	AuthoritativeRoots []string `json:"authoritativeRoots"`
	ExcludedRoots      []string `json:"excludedRoots"`
	RuntimeMirrors     []string `json:"runtimeMirrors"`
}

type LayoutReport struct {
	SchemaVersion    int                 `json:"schemaVersion"`
	Config           string              `json:"config"`
	DirectoryClasses map[string][]string `json:"directoryClasses"`
	Checks           []LayoutCheck       `json:"checks"`
	Errors           []string            `json:"errors"`
}

type LayoutCheck struct {
	Name   string   `json:"name"`
	OK     bool     `json:"ok"`
	Errors []string `json:"errors,omitempty"`
}

// CheckLayout validates the repository ownership and placement rules in the
// single machine-readable layout configuration.
func CheckLayout(repo string) LayoutReport {
	report := LayoutReport{SchemaVersion: 1, Config: repositoryLayoutPath, Checks: []LayoutCheck{}, Errors: []string{}}
	config, err := loadLayoutConfig(repo)
	if err != nil {
		report.addCheck("load layout configuration", []string{err.Error()})
		return report
	}
	report.DirectoryClasses = config.DirectoryClasses
	if config.SchemaVersion != 1 {
		report.addCheck("load layout configuration", []string{"unsupported schemaVersion"})
		return report
	}

	report.addCheck("root directory allowlist", checkRootDirectories(repo, config))
	staged, stagedErr := gitx.StagedFiles(repo)
	if stagedErr != nil {
		report.addCheck("generated artifacts staged", []string{stagedErr.Error()})
	} else {
		report.addCheck("generated artifacts staged", checkGeneratedArtifacts(staged, config.Generated))
	}

	tracked, trackedErr := trackedFiles(repo)
	if trackedErr != nil {
		report.addCheck("documentation placement", []string{trackedErr.Error()})
		report.addCheck("prompt ownership", []string{trackedErr.Error()})
	} else {
		report.addCheck("documentation placement", checkDocumentationPlacement(tracked, config))
		report.addCheck("prompt ownership", checkPromptOwnership(repo, tracked, config))
	}
	report.addCheck("tests and testdata overlap", checkTestFixtureOverlap(repo, config))
	report.addCheck("skill source-of-truth", checkSkillSources(repo, config))
	return report
}

func (r *LayoutReport) addCheck(name string, errs []string) {
	sort.Strings(errs)
	check := LayoutCheck{Name: name, OK: len(errs) == 0, Errors: errs}
	r.Checks = append(r.Checks, check)
	for _, err := range errs {
		r.Errors = append(r.Errors, name+": "+err)
	}
}

func loadLayoutConfig(repo string) (layoutConfig, error) {
	var config layoutConfig
	b, err := os.ReadFile(platform.RepoPath(repo, repositoryLayoutPath))
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(b, &config); err != nil {
		return config, err
	}
	return config, nil
}

func checkRootDirectories(repo string, config layoutConfig) []string {
	errs := []string{}
	entries, err := os.ReadDir(repo)
	if err != nil {
		return []string{err.Error()}
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".git" {
			continue
		}
		name := filepath.ToSlash(entry.Name())
		if containsLayoutPath(config.Root.ForbiddenDirectories, name) {
			errs = append(errs, "forbidden root directory exists: "+name)
			continue
		}
		if !containsLayoutPath(config.Root.AllowDirectories, name) && !containsLayoutPath(config.Root.TransientDirectories, name) {
			errs = append(errs, "root directory is outside allowlist: "+name)
		}
	}
	return errs
}

func checkGeneratedArtifacts(staged []string, generated layoutGenerated) []string {
	errs := []string{}
	for _, file := range staged {
		file = filepath.ToSlash(file)
		for _, dir := range generated.Directories {
			if strings.HasPrefix(file, dir) {
				errs = append(errs, "generated artifact is staged: "+file)
				break
			}
		}
		for _, ext := range generated.Extensions {
			if strings.HasSuffix(strings.ToLower(file), strings.ToLower(ext)) {
				errs = append(errs, "generated artifact is staged: "+file)
				break
			}
		}
	}
	return uniqueLayoutErrors(errs)
}

func trackedFiles(repo string) ([]string, error) {
	out, err := gitx.Run(repo, "ls-files")
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(filepath.ToSlash(line))
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func checkDocumentationPlacement(tracked []string, config layoutConfig) []string {
	errs := []string{}
	docsRoot := strings.TrimSuffix(filepath.ToSlash(config.Documentation.Root), "/")
	for _, file := range tracked {
		if !strings.EqualFold(filepath.Ext(file), ".md") {
			continue
		}
		if isLayoutWithin(file, docsRoot) || containsLayoutPath(config.Documentation.AllowedRootFiles, file) || isWithinAnyLayoutRoot(file, config.Documentation.AllowedOutsideRoots) {
			continue
		}
		errs = append(errs, "documentation file is outside docs: "+file)
	}
	return errs
}

func checkPromptOwnership(repo string, tracked []string, config layoutConfig) []string {
	errs := []string{}
	if config.Prompts.ForbiddenRoot != "" && platform.IsDir(platform.RepoPath(repo, config.Prompts.ForbiddenRoot)) {
		errs = append(errs, "forbidden prompt root exists: "+config.Prompts.ForbiddenRoot)
	}
	for _, file := range tracked {
		if !strings.HasSuffix(strings.ToLower(file), ".prompt.md") {
			continue
		}
		if !strings.Contains("/"+file, "/prompts/") || !isWithinAnyLayoutRoot(file, config.Prompts.AllowedRoots) {
			errs = append(errs, "prompt is separated from its owning tool or skill: "+file)
		}
	}
	return errs
}

func checkTestFixtureOverlap(repo string, config layoutConfig) []string {
	if config.TestFixtures.ForbiddenRoot == "" || !platform.IsDir(platform.RepoPath(repo, config.TestFixtures.ForbiddenRoot)) {
		return nil
	}
	errs := []string{"forbidden test fixture root exists: " + config.TestFixtures.ForbiddenRoot}
	if !platform.IsDir(platform.RepoPath(repo, config.TestFixtures.Root)) {
		return errs
	}
	testDirs, _ := os.ReadDir(platform.RepoPath(repo, config.TestFixtures.ForbiddenRoot))
	fixtureDirs, _ := os.ReadDir(platform.RepoPath(repo, config.TestFixtures.Root))
	fixtures := map[string]bool{}
	for _, entry := range fixtureDirs {
		if entry.IsDir() {
			fixtures[entry.Name()] = true
		}
	}
	for _, entry := range testDirs {
		if entry.IsDir() && fixtures[entry.Name()] {
			errs = append(errs, "tests/testdata duplicate fixture directory: "+entry.Name())
		}
	}
	return errs
}

func checkSkillSources(repo string, config layoutConfig) []string {
	errs := []string{}
	seen := map[string]string{}
	excluded := map[string]bool{}
	for _, root := range config.Skills.ExcludedRoots {
		excluded[filepath.Clean(platform.RepoPath(repo, root))] = true
	}
	for _, root := range config.Skills.AuthoritativeRoots {
		rootPath := platform.RepoPath(repo, root)
		if !platform.IsDir(rootPath) {
			errs = append(errs, "authoritative skill root is missing: "+root)
			continue
		}
		walkErr := filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == ".git" || excluded[filepath.Clean(path)] {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(entry.Name(), "SKILL.md") {
				return nil
			}
			id := layoutSkillID(path)
			if previous, ok := seen[id]; ok && previous != path {
				previousRel, _ := filepath.Rel(repo, previous)
				currentRel, _ := filepath.Rel(repo, path)
				errs = append(errs, "skill has multiple source-of-truth paths: "+id+" ("+filepath.ToSlash(previousRel)+", "+filepath.ToSlash(currentRel)+")")
				return nil
			}
			seen[id] = path
			return nil
		})
		if walkErr != nil {
			errs = append(errs, "cannot inspect authoritative skill root "+root+": "+walkErr.Error())
		}
	}
	return uniqueLayoutErrors(errs)
}

func layoutSkillID(path string) string {
	b, err := os.ReadFile(path)
	if err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "name:") {
				name := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "name:")), "\"'")
				if name != "" {
					return name
				}
			}
		}
	}
	return filepath.Base(filepath.Dir(path))
}

func containsLayoutPath(paths []string, target string) bool {
	for _, path := range paths {
		if filepath.ToSlash(path) == filepath.ToSlash(target) {
			return true
		}
	}
	return false
}

func isWithinAnyLayoutRoot(path string, roots []string) bool {
	for _, root := range roots {
		if isLayoutWithin(path, root) {
			return true
		}
	}
	return false
}

func isLayoutWithin(path, root string) bool {
	path = strings.Trim(filepath.ToSlash(path), "/")
	root = strings.Trim(filepath.ToSlash(root), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func uniqueLayoutErrors(errs []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, err := range errs {
		if err != "" && !seen[err] {
			seen[err] = true
			unique = append(unique, err)
		}
	}
	return unique
}
