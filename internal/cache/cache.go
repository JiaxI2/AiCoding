package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type Scope string

const (
	ScopeFastPath          Scope = "fast-path"
	ScopeTestResults       Scope = "test-results"
	ScopeValidationReports Scope = "validation-reports"
	ScopeWorkState         Scope = "work-state"

	DefaultTestResultKeep = 5
	testResultWarnCount   = 20
	testResultWarnBytes   = 50 * 1024 * 1024
)

type ScopeStatus struct {
	Scope      Scope  `json:"scope"`
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	EntryCount int    `json:"entryCount"`
	SizeBytes  int64  `json:"sizeBytes"`
	Policy     string `json:"policy"`
}

type StatusResult struct {
	Scopes          []ScopeStatus `json:"scopes"`
	TotalEntryCount int           `json:"totalEntryCount"`
	TotalSizeBytes  int64         `json:"totalSizeBytes"`
}

type CleanOptions struct {
	Scope  Scope
	Keep   int
	DryRun bool
}

type CleanEntry struct {
	Scope     Scope  `json:"scope"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	Reason    string `json:"reason"`
}

type ScopeCleanResult struct {
	Scope         Scope        `json:"scope"`
	Path          string       `json:"path"`
	Policy        string       `json:"policy"`
	RetainedCount int          `json:"retainedCount"`
	Planned       []CleanEntry `json:"planned"`
	Removed       []CleanEntry `json:"removed,omitempty"`
	PlannedBytes  int64        `json:"plannedBytes"`
	FreedBytes    int64        `json:"freedBytes"`
}

type CleanResult struct {
	Scope        string             `json:"scope"`
	DryRun       bool               `json:"dryRun"`
	Keep         int                `json:"keep"`
	Scopes       []ScopeCleanResult `json:"scopes"`
	PlannedCount int                `json:"plannedCount"`
	RemovedCount int                `json:"removedCount"`
	PlannedBytes int64              `json:"plannedBytes"`
	FreedBytes   int64              `json:"freedBytes"`
}

type artifactSpec struct {
	scope          Scope
	root           string
	displayPath    string
	policy         string
	cleanByDefault bool
	cleanable      bool
	match          func(string) bool
}

type artifactEntry struct {
	path      string
	display   string
	size      int64
	modTime   int64
	directory bool
}

func Status(repo string) (StatusResult, error) {
	specs := registry(repo)
	result := StatusResult{Scopes: make([]ScopeStatus, 0, len(specs))}
	for _, spec := range specs {
		status, _, err := scan(repo, spec)
		if err != nil {
			return result, err
		}
		result.Scopes = append(result.Scopes, status)
		result.TotalEntryCount += status.EntryCount
		result.TotalSizeBytes += status.SizeBytes
	}
	return result, nil
}

func BloatWarnings(status StatusResult) []string {
	for _, scope := range status.Scopes {
		if scope.Scope != ScopeTestResults {
			continue
		}
		if scope.EntryCount > testResultWarnCount || scope.SizeBytes > testResultWarnBytes {
			return []string{fmt.Sprintf("test results use %d entries / %d bytes; run `aicoding cache clean --scope test-results --json`", scope.EntryCount, scope.SizeBytes)}
		}
	}
	return nil
}

func ValidScope(scope Scope) bool {
	for _, candidate := range []Scope{ScopeFastPath, ScopeTestResults, ScopeValidationReports, ScopeWorkState} {
		if scope == candidate {
			return true
		}
	}
	return false
}

func registry(repo string) []artifactSpec {
	validationRoot := filepath.Join(repo, ".git", "aicoding", "validation", "reports")
	if commonDir, err := gitx.CommonDir(repo); err == nil {
		validationRoot = filepath.Join(commonDir, "aicoding", "validation", "reports")
	}
	return []artifactSpec{
		{
			scope: ScopeFastPath, root: platform.RepoPath(repo, ".aicoding/cache/fast-path"),
			displayPath: ".aicoding/cache/fast-path", policy: "remove-all",
			cleanByDefault: true, cleanable: true,
		},
		{
			scope: ScopeTestResults, root: platform.RepoPath(repo, "test-results"),
			displayPath: "test-results/aicoding-global-test-*", policy: "keep-latest-5-plus-all-failures",
			cleanByDefault: true, cleanable: true,
			match: func(name string) bool { return strings.HasPrefix(name, "aicoding-global-test-") },
		},
		{
			scope: ScopeValidationReports, root: validationRoot,
			displayPath: filepath.ToSlash(validationRoot), policy: "remove-only-reports-unreferenced-by-receipts-or-aliases",
			cleanByDefault: true, cleanable: true,
			match: validReportDirName,
		},
		{
			scope: ScopeWorkState, root: platform.RepoPath(repo, ".aicoding/state/work"),
			displayPath: ".aicoding/state/work/*", policy: "audit-only; list-size; never-clean-attempts.jsonl",
			cleanByDefault: false, cleanable: false,
		},
	}
}

func scan(repo string, spec artifactSpec) (ScopeStatus, []artifactEntry, error) {
	status := ScopeStatus{Scope: spec.scope, Path: spec.displayPath, Policy: spec.policy}
	info, err := os.Stat(spec.root)
	if os.IsNotExist(err) {
		return status, nil, nil
	}
	if err != nil {
		return status, nil, err
	}
	status.Exists = true
	if !info.IsDir() {
		size, err := pathSize(spec.root)
		if err != nil {
			return status, nil, err
		}
		entry := artifactEntry{path: spec.root, display: displayEntryPath(repo, spec.root), size: size, modTime: info.ModTime().UnixNano()}
		status.EntryCount = 1
		status.SizeBytes = size
		return status, []artifactEntry{entry}, nil
	}
	children, err := os.ReadDir(spec.root)
	if err != nil {
		return status, nil, err
	}
	entries := make([]artifactEntry, 0, len(children))
	for _, child := range children {
		if spec.match != nil && !spec.match(child.Name()) {
			continue
		}
		path := filepath.Join(spec.root, child.Name())
		size, err := pathSize(path)
		if err != nil {
			return status, nil, err
		}
		childInfo, err := child.Info()
		if err != nil {
			return status, nil, err
		}
		modTime := childInfo.ModTime()
		if spec.scope == ScopeTestResults {
			if summaryInfo, summaryErr := os.Stat(filepath.Join(path, "summary.json")); summaryErr == nil {
				modTime = summaryInfo.ModTime()
			}
		}
		entries = append(entries, artifactEntry{
			path: path, display: displayEntryPath(repo, path), size: size,
			modTime: modTime.UnixNano(), directory: childInfo.IsDir(),
		})
		status.EntryCount++
		status.SizeBytes += size
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].modTime == entries[j].modTime {
			return entries[i].display > entries[j].display
		}
		return entries[i].modTime > entries[j].modTime
	})
	return status, entries, nil
}

func pathSize(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}

func displayEntryPath(repo, path string) string {
	relative, err := filepath.Rel(repo, path)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(path)
}

func validReportDirName(name string) bool {
	if len(name) != 64 || name != strings.ToLower(name) {
		return false
	}
	for _, char := range name {
		if char < '0' || char > '9' {
			if char < 'a' || char > 'f' {
				return false
			}
		}
	}
	return true
}
