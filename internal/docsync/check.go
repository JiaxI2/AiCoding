package docsync

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/capability"
	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type CheckResult struct {
	OK        bool        `json:"ok"`
	Command   string      `json:"command"`
	Mode      string      `json:"mode"`
	Checks    []CheckItem `json:"checks"`
	Checked   []string    `json:"checked"`
	RiskFiles []string    `json:"riskFiles"`
	DocFiles  []string    `json:"docFiles"`
	Warnings  []string    `json:"warnings"`
	Errors    []string    `json:"errors"`
}

type CheckItem struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
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
		Checks:    []CheckItem{},
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
	if checked, schemaErrors := CheckPolicySchemas(repo); checked {
		result.Checks = append(result.Checks, CheckItem{Name: "policy schema closure", OK: len(schemaErrors) == 0})
		result.Errors = append(result.Errors, schemaErrors...)
	}
	if mode == "all" || mode == "ci" || mode == "release" {
		statusErrors := architectureStatusErrors(repo)
		result.Checks = append(result.Checks, CheckItem{Name: "architecture status headers", OK: len(statusErrors) == 0})
		result.Errors = append(result.Errors, statusErrors...)
	}
	if checked, generatedErrors := capabilityGeneratedIndexErrors(repo); checked {
		result.Checks = append(result.Checks, CheckItem{Name: "capability generated index", OK: len(generatedErrors) == 0})
		result.Errors = append(result.Errors, generatedErrors...)
	}
	if mode == "ci" || mode == "release" {
		result.Errors = append(result.Errors, requiredPathErrors(repo, []string{
			"internal/docsync/docsync.go",
			"internal/docsync/check.go",

			"config/docs-sync.policy.json",
			"config/docs-sync.semantic.json",
			".github/workflows/aicoding-ci.yml",
		})...)
	}
	if mode == "release" {
		result.Errors = append(result.Errors, requiredPathErrors(repo, []string{
			"docs/architecture/DOC_SYNC_PLUS_SPEC.md",
			"docs/operations/DOC_SYNC_PLUS_VALIDATION_PLAN.md",
		})...)
	}
	result.Errors = compact(result.Errors)
	result.Warnings = compact(result.Warnings)
	result.OK = len(result.Errors) == 0
	return result
}

func capabilityGeneratedIndexErrors(repo string) (bool, []string) {
	registryPath := filepath.Join(repo, filepath.FromSlash(capability.CatalogPath))
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return true, []string{"capability generated index: " + err.Error()}
	}
	catalog, err := capability.Load(repo)
	if err != nil {
		return true, []string{"capability generated index: " + err.Error()}
	}
	readme, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		return true, []string{"capability generated index: " + err.Error()}
	}
	rendered, err := capability.RenderIndex(catalog, string(readme))
	if err != nil {
		return true, []string{"capability generated index: " + err.Error()}
	}
	errs := []string{}
	if normalizeDocSyncNewlines(string(readme)) != normalizeDocSyncNewlines(rendered.README) {
		errs = append(errs, "capability generated index: README.md is stale; run `aicoding capability index --write`")
	}
	document, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(capability.CapabilitiesPath)))
	if err != nil || normalizeDocSyncNewlines(string(document)) != normalizeDocSyncNewlines(rendered.Document) {
		errs = append(errs, "capability generated index: docs/CAPABILITIES.md is stale; run `aicoding capability index --write`")
	}
	return true, errs
}

func normalizeDocSyncNewlines(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func architectureStatusErrors(repo string) []string {
	dir := filepath.Join(repo, "docs", "architecture")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{"architecture status check: " + err.Error()}
	}
	errs := []string{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || strings.EqualFold(entry.Name(), "README.md") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			errs = append(errs, "architecture status check: "+entry.Name()+": "+readErr.Error())
			continue
		}
		found := false
		for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
			if strings.HasPrefix(line, "Status: ") && strings.TrimSpace(strings.TrimPrefix(line, "Status: ")) != "" {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, "architecture status check: docs/architecture/"+entry.Name()+" is missing a Status header")
		}
	}
	return errs
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
