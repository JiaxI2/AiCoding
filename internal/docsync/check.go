package docsync

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type CheckResult struct {
	OK        bool     `json:"ok"`
	Command   string   `json:"command"`
	Mode      string   `json:"mode"`
	Checked   []string `json:"checked"`
	RiskFiles []string `json:"riskFiles"`
	DocFiles  []string `json:"docFiles"`
	Warnings  []string `json:"warnings"`
	Errors    []string `json:"errors"`
}

func Check(repo, mode string) CheckResult {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "staged"
	}
	result := CheckResult{
		OK:        true,
		Command:   "docsync " + mode,
		Mode:      mode,
		Checked:   []string{},
		RiskFiles: []string{},
		DocFiles:  []string{},
		Warnings:  []string{},
		Errors:    []string{},
	}
	var files []string
	var errs []string
	switch mode {
	case "staged":
		files, errs = stagedFiles(repo)
	case "all":
		files, errs = allFiles(repo)
	case "ci":
		files, errs = changedFiles(repo)
	case "release":
		files, errs = allFiles(repo)
	default:
		result.OK = false
		result.Errors = []string{"unsupported docsync mode: " + mode}
		return result
	}
	result.Checked = append(result.Checked, files...)
	result.RiskFiles, result.DocFiles = classifyFiles(files)
	result.Errors = append(result.Errors, errs...)
	result.Errors = append(result.Errors, policyErrors(files)...)
	if mode == "ci" || mode == "release" {
		result.Errors = append(result.Errors, requiredPathErrors(repo, []string{
			"internal/docsync/docsync.go",
			"internal/docsync/check.go",

			"config/docs-sync.policy.json",
			"config/docs-sync.semantic.json",
			".github/workflows/docs-sync.yml",
		})...)
	}
	if mode == "release" {
		result.Errors = append(result.Errors, requiredPathErrors(repo, []string{
			"docs/DOC_SYNC_PLUS_SPEC.md",
			"docs/DOC_SYNC_PLUS_VALIDATION_PLAN.md",
		})...)
	}
	result.Errors = compact(result.Errors)
	result.Warnings = compact(result.Warnings)
	result.OK = len(result.Errors) == 0
	return result
}

func stagedFiles(repo string) ([]string, []string) {
	files, err := gitx.StagedFiles(repo)
	if err != nil {
		return nil, []string{err.Error()}
	}
	return uniqueSorted(files), nil
}

func changedFiles(repo string) ([]string, []string) {
	out := []string{}
	errs := []string{}
	if files, err := gitx.StagedFiles(repo); err != nil {
		errs = append(errs, err.Error())
	} else {
		out = append(out, files...)
	}
	for _, args := range [][]string{
		{"-c", "core.quotePath=false", "diff", "--name-only", "--diff-filter=ACMR"},
		{"-c", "core.quotePath=false", "ls-files", "--others", "--exclude-standard"},
	} {
		text, err := gitx.Run(repo, args...)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		for _, line := range strings.Split(text, "\n") {
			if line = strings.TrimSpace(line); line != "" && platform.Exists(platform.RepoPath(repo, filepath.ToSlash(line))) {
				out = append(out, line)
			}
		}
	}
	return uniqueSorted(out), compact(errs)
}

func allFiles(repo string) ([]string, []string) {
	out := []string{}
	errs := []string{}
	for _, args := range [][]string{
		{"-c", "core.quotePath=false", "ls-files"},
		{"-c", "core.quotePath=false", "ls-files", "--others", "--exclude-standard"},
	} {
		text, err := gitx.Run(repo, args...)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		for _, line := range strings.Split(text, "\n") {
			if line = strings.TrimSpace(line); line != "" && platform.Exists(platform.RepoPath(repo, filepath.ToSlash(line))) {
				out = append(out, line)
			}
		}
	}
	return uniqueSorted(out), compact(errs)
}

func classifyFiles(files []string) ([]string, []string) {
	riskFiles := []string{}
	docFiles := []string{}
	for _, file := range files {
		if IsDocSyncRiskPath(file) {
			riskFiles = append(riskFiles, file)
		}
		if IsDocPath(file) {
			docFiles = append(docFiles, file)
		}
	}
	return uniqueSorted(riskFiles), uniqueSorted(docFiles)
}

func policyErrors(files []string) []string {
	if len(files) == 0 || os.Getenv("AICODING_SKIP_DOCSYNC") == "1" {
		return nil
	}
	docChanged := false
	riskChanged := false
	for _, file := range files {
		if IsDocPath(file) {
			docChanged = true
		}
		if IsDocSyncRiskPath(file) {
			riskChanged = true
		}
	}
	if riskChanged && !docChanged {
		return []string{"documentation sync gate: source/script/config/hook/skill changes require documentation changes or AICODING_SKIP_DOCSYNC=1"}
	}
	return nil
}

func requiredPathErrors(repo string, paths []string) []string {
	errs := []string{}
	for _, rel := range paths {
		if !platform.Exists(platform.RepoPath(repo, rel)) {
			errs = append(errs, "missing required DocSync path: "+rel)
		}
	}
	return errs
}

func uniqueSorted(files []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, file := range files {
		rel := filepath.ToSlash(strings.TrimSpace(file))
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

func compact(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
