package cstyle

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Scope string

const (
	ScopeChanged Scope = "changed"
	ScopeStaged  Scope = "staged"
	ScopeAll     Scope = "all"
	ScopePaths   Scope = "paths"
)

type Options struct {
	RepoRoot string
	Scope    Scope
	Paths    []string
	Check    bool
	Preview  bool
}

type ToolStatus struct {
	Found   bool   `json:"found"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

type Result struct {
	SkillID         string     `json:"skillId"`
	Scope           Scope      `json:"scope"`
	RepoRoot        string     `json:"repoRoot"`
	FormatterConfig string     `json:"formatterConfig"`
	Files           []string   `json:"files"`
	Changed         []string   `json:"changed,omitempty"`
	Errors          []string   `json:"errors,omitempty"`
	ClangFormat     ToolStatus `json:"clangFormat"`
	ElapsedMS       int64      `json:"elapsedMs"`
}

var defaultExcludedDirs = map[string]bool{
	".git":        true,
	"vendor":      true,
	"third_party": true,
	"generated":   true,
	"Drivers":     true,
	"device":      true,
	"build":       true,
	"out":         true,
	"dist":        true,
}

type exclusionSet struct {
	directoryNames map[string]bool
	pathPrefixes   []string
}

func Status() ToolStatus {
	path, err := exec.LookPath("clang-format")
	if err != nil {
		return ToolStatus{Found: false}
	}
	out, _ := exec.Command(path, "--version").CombinedOutput()
	return ToolStatus{
		Found:   true,
		Path:    path,
		Version: strings.TrimSpace(string(out)),
	}
}

func Run(opts Options) (Result, error) {
	return RunBySkill(DefaultSkillID, opts)
}

func FormatBySkill(skillID string, opts Options) (Result, error) {
	opts.Check = false
	return RunBySkill(skillID, opts)
}

func CheckBySkill(skillID string, opts Options) (Result, error) {
	opts.Check = true
	return RunBySkill(skillID, opts)
}

func RunBySkill(skillID string, opts Options) (Result, error) {
	start := time.Now()
	if opts.Scope == "" {
		opts.Scope = ScopeChanged
	}

	id := normalizeSkillID(skillID)
	res := Result{
		SkillID: id,
		Scope:   opts.Scope,
	}

	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		res.ElapsedMS = time.Since(start).Milliseconds()
		res.Errors = []string{err.Error()}
		return res, err
	}
	res.RepoRoot = repoRoot

	cfg, err := LoadSkillConfig(repoRoot, id)
	if err != nil {
		res.ElapsedMS = time.Since(start).Milliseconds()
		res.Errors = []string{err.Error()}
		return res, err
	}

	formatterConfig, err := ResolveFormatterConfig(repoRoot, cfg)
	if err != nil {
		res.ElapsedMS = time.Since(start).Milliseconds()
		res.Errors = []string{err.Error()}
		return res, err
	}
	res.FormatterConfig = relativeRepoPath(repoRoot, formatterConfig)
	if !fileExists(formatterConfig) {
		err := fmt.Errorf("formatter config not found: %s", res.FormatterConfig)
		res.ElapsedMS = time.Since(start).Milliseconds()
		res.Errors = []string{err.Error()}
		return res, err
	}

	files, err := CollectFilesWithExclusions(repoRoot, opts.Scope, opts.Paths, cfg.ExcludedDirectories)
	if err != nil {
		res.ElapsedMS = time.Since(start).Milliseconds()
		res.Errors = []string{err.Error()}
		return res, err
	}
	res.Files = files

	status := Status()
	res.ClangFormat = status
	if len(files) == 0 {
		res.ElapsedMS = time.Since(start).Milliseconds()
		return res, nil
	}
	if !status.Found {
		res.ElapsedMS = time.Since(start).Milliseconds()
		res.Errors = []string{"clang-format not found on PATH"}
		return res, errors.New("clang-format not found on PATH")
	}

	for _, file := range files {
		changed, runErr := runOne(status.Path, formatterConfig, repoRoot, file, opts)
		if runErr != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", file, runErr))
			continue
		}
		if changed {
			res.Changed = append(res.Changed, file)
		}
	}

	res.ElapsedMS = time.Since(start).Milliseconds()

	if len(res.Errors) > 0 {
		return res, errors.New(strings.Join(res.Errors, "; "))
	}
	return res, nil
}

func CollectFiles(repoRoot string, scope Scope, explicitPaths []string) ([]string, error) {
	return CollectFilesWithExclusions(repoRoot, scope, explicitPaths, nil)
}

func CollectFilesWithExclusions(repoRoot string, scope Scope, explicitPaths []string, excludedDirs []string) ([]string, error) {
	var raw []string
	var err error

	switch scope {
	case ScopeAll:
		raw, err = allCFiles(repoRoot, excludedDirs)
	case ScopeStaged:
		raw, err = gitLines(repoRoot, "diff", "--cached", "--name-only", "--diff-filter=ACMRTUXB", "--", "*.c", "*.h")
	case ScopeChanged:
		raw, err = changedCFiles(repoRoot)
	case ScopePaths:
		raw = explicitPaths
	default:
		return nil, fmt.Errorf("unsupported cstyle scope: %s", scope)
	}
	if err != nil {
		return nil, err
	}

	excluded := excludedSet(excludedDirs)
	seen := map[string]bool{}
	var out []string
	for _, item := range raw {
		rel, ok := normalizeCandidate(repoRoot, item, excluded)
		if !ok {
			continue
		}
		if seen[rel] {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}

	sort.Strings(out)
	return out, nil
}

func changedCFiles(repoRoot string) ([]string, error) {
	changed, err := gitLines(repoRoot, "diff", "--name-only", "--diff-filter=ACMRTUXB", "HEAD", "--", "*.c", "*.h")
	if err != nil {
		changed, err = gitLines(repoRoot, "diff", "--name-only", "--diff-filter=ACMRTUXB", "--", "*.c", "*.h")
		if err != nil {
			return nil, err
		}
	}

	untracked, err := gitLines(repoRoot, "ls-files", "--others", "--exclude-standard", "--", "*.c", "*.h")
	if err != nil {
		return changed, nil
	}

	return append(changed, untracked...), nil
}

func allCFiles(repoRoot string, excludedDirs []string) ([]string, error) {
	excluded := excludedSet(excludedDirs)
	var out []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel != "." && isExcluded(rel, excluded) {
				return filepath.SkipDir
			}
			return nil
		}

		if isCHeaderOrSource(rel) && !isExcluded(rel, excluded) {
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}

func normalizeCandidate(repoRoot string, item string, excluded exclusionSet) (string, bool) {
	item = strings.TrimSpace(item)
	if item == "" {
		return "", false
	}

	path := item
	if filepath.IsAbs(item) {
		rel, err := filepath.Rel(repoRoot, item)
		if err != nil {
			return "", false
		}
		path = rel
	}

	rel := filepath.ToSlash(filepath.Clean(path))
	if rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return "", false
	}
	if !isCHeaderOrSource(rel) || isExcluded(rel, excluded) {
		return "", false
	}

	if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel))); err != nil {
		return "", false
	}
	return rel, true
}

func isCHeaderOrSource(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".c" || ext == ".h"
}

func exclusionPathKey(path string) string {
	key := filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}

func isExcluded(rel string, excluded exclusionSet) bool {
	rel = exclusionPathKey(rel)
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if excluded.directoryNames[p] {
			return true
		}
	}
	for _, prefix := range excluded.pathPrefixes {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return false
}

func excludedSet(excludedDirs []string) exclusionSet {
	out := exclusionSet{
		directoryNames: map[string]bool{".git": true},
	}
	prefixes := map[string]bool{}
	source := excludedDirs
	if len(source) == 0 {
		for dir := range defaultExcludedDirs {
			source = append(source, dir)
		}
	}
	for _, dir := range source {
		dir = exclusionPathKey(strings.TrimSpace(dir))
		dir = strings.TrimPrefix(dir, "./")
		if dir == "" || dir == "." {
			continue
		}
		if strings.Contains(dir, "/") {
			if !prefixes[dir] {
				prefixes[dir] = true
				out.pathPrefixes = append(out.pathPrefixes, dir)
			}
			continue
		}
		out.directoryNames[dir] = true
	}
	sort.Strings(out.pathPrefixes)
	return out
}

func runOne(clangPath, formatterConfig, repoRoot, rel string, opts Options) (bool, error) {
	full := filepath.Join(repoRoot, filepath.FromSlash(rel))
	styleArg := "--style=file"
	if strings.TrimSpace(formatterConfig) != "" {
		styleArg = "--style=file:" + formatterConfig
	}

	if opts.Check {
		cmd := exec.Command(clangPath, "--dry-run", "--Werror", styleArg, full)
		cmd.Dir = repoRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return false, errors.New(msg)
		}
		return false, nil
	}

	if opts.Preview {
		before, err := os.ReadFile(full)
		if err != nil {
			return false, err
		}
		cmd := exec.Command(clangPath, styleArg, full)
		cmd.Dir = repoRoot
		after, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return false, errors.New(strings.TrimSpace(string(ee.Stderr)))
			}
			return false, err
		}
		return !bytes.Equal(before, after), nil
	}

	cmd := exec.Command(clangPath, "-i", styleArg, full)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return false, errors.New(msg)
	}
	return true, nil
}

func resolveRepoRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) != "" {
		abs, err := filepath.Abs(repoRoot)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			return filepath.Abs(root)
		}
	}

	return os.Getwd()
}

func gitLines(repoRoot string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}

	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
