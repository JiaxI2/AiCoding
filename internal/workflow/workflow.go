package workflow

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/governance"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/repohealth"
)

type ChangedFile struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

type CheckPlan struct {
	ID      string `json:"id"`
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

type CheckResult struct {
	ID     string   `json:"id"`
	OK     bool     `json:"ok"`
	Errors []string `json:"errors,omitempty"`
}

type Plan struct {
	Files             []ChangedFile `json:"files"`
	Checks            []CheckPlan   `json:"checks"`
	ExcludedSlowPaths []string      `json:"excludedSlowPaths"`
}

type Result struct {
	Plan    Plan          `json:"plan"`
	Results []CheckResult `json:"results"`
}

func (p Plan) HasCheck(id string) bool {
	for _, c := range p.Checks {
		if c.ID == id {
			return true
		}
	}
	return false
}

func PlanForFiles(files []string) Plan {
	plan := Plan{ExcludedSlowPaths: []string{"Full", "Release", "install", "uninstall", "export", "rollback", "fresh-clone", "DSS", "PSScriptAnalyzer"}}
	seenFiles := map[string]bool{}
	for _, f := range files {
		rel := filepath.ToSlash(strings.TrimSpace(f))
		if rel == "" || seenFiles[rel] {
			continue
		}
		seenFiles[rel] = true
		plan.Files = append(plan.Files, ChangedFile{Path: rel, Source: "git"})
	}
	if len(plan.Files) == 0 {
		addCheck(&plan, "kit-smoke", "bin/aicoding.exe kit verify --all --profile Smoke --json", "default smoke plan")
		addCheck(&plan, "governance-lint", "bin/aicoding.exe governance lint --json", "default smoke plan")
		addCheck(&plan, "verify-hooks", "bin/aicoding.exe verify hooks --json", "default smoke plan")
		addCheck(&plan, "verify-repo-text", "bin/aicoding.exe verify repo-text --json", "default smoke plan")
		addCheck(&plan, "verify-release-notes", "bin/aicoding.exe verify release-notes --json", "default smoke plan")
		addCheck(&plan, "doctor-perf", "bin/aicoding.exe doctor perf --json", "default smoke plan")
		return plan
	}
	for _, f := range plan.Files {
		rel := f.Path
		lower := strings.ToLower(rel)
		ext := strings.ToLower(filepath.Ext(rel))
		switch {
		case ext == ".go" || rel == "go.mod":
			addCheck(&plan, "go-test", "go test ./...", "Go source changed")
		}
		if strings.HasPrefix(lower, "config/kits/") || lower == "config/kit-registry.json" || strings.HasPrefix(lower, "codingkit/") || lower == "taskfile.yml" {
			addCheck(&plan, "kit-smoke", "bin/aicoding.exe kit verify --all --profile Smoke --json", "kit registry, manifest, Taskfile, or CodingKit surface changed")
		}
		if strings.HasPrefix(lower, ".githooks/") || lower == "taskfile.yml" {
			addCheck(&plan, "verify-hooks", "bin/aicoding.exe verify hooks --json", "hook or task routing changed")
		}
		if strings.HasPrefix(lower, "docs/") || strings.HasPrefix(lower, "readme") || lower == "changelog.md" {
			addCheck(&plan, "verify-repo-text", "bin/aicoding.exe verify repo-text --json", "maintained documentation changed")
		}
		if strings.Contains(lower, "release") || strings.Contains(lower, "tag") || strings.HasPrefix(lower, ".github/") {
			addCheck(&plan, "verify-release-notes", "bin/aicoding.exe verify release-notes --json", "release or tag governance changed")
		}
		if strings.HasPrefix(lower, "readme") || lower == "changelog.md" || strings.HasPrefix(lower, ".github/") || lower == "taskfile.yml" {
			addCheck(&plan, "governance-lint", "bin/aicoding.exe governance lint --json", "governance-controlled surface changed")
		}
	}
	if len(plan.Checks) == 0 {
		addCheck(&plan, "governance-lint", "bin/aicoding.exe governance lint --json", "fallback structural check")
	}
	return plan
}

func SmartVerify(repo string) (Result, []string) {
	files, errs := ChangedFiles(repo)
	plan := PlanForFiles(filePaths(files))
	result := Result{Plan: plan}
	allErrs := append([]string{}, errs...)
	for _, check := range plan.Checks {
		cr := runCheck(repo, check.ID)
		result.Results = append(result.Results, cr)
		if !cr.OK {
			for _, e := range cr.Errors {
				allErrs = append(allErrs, check.ID+": "+e)
			}
		}
	}
	return result, allErrs
}

func ChangedFiles(repo string) ([]ChangedFile, []string) {
	errs := []string{}
	out := []ChangedFile{}
	seen := map[string]bool{}
	add := func(path, source string) {
		rel := filepath.ToSlash(strings.TrimSpace(path))
		if rel == "" || seen[rel] {
			return
		}
		seen[rel] = true
		out = append(out, ChangedFile{Path: rel, Source: source})
	}
	staged, err := gitx.StagedFiles(repo)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		for _, f := range staged {
			add(f, "staged")
		}
	}
	if diff, err := gitx.Run(repo, "diff", "--name-only", "--diff-filter=ACMR"); err != nil {
		errs = append(errs, err.Error())
	} else {
		for _, line := range strings.Split(diff, "\n") {
			add(line, "changed")
		}
	}
	if untracked, err := gitx.Run(repo, "ls-files", "--others", "--exclude-standard"); err != nil {
		errs = append(errs, err.Error())
	} else {
		for _, line := range strings.Split(untracked, "\n") {
			add(line, "untracked")
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, errs
}

func addCheck(plan *Plan, id, command, reason string) {
	for _, c := range plan.Checks {
		if c.ID == id {
			return
		}
	}
	plan.Checks = append(plan.Checks, CheckPlan{ID: id, Command: command, Reason: reason})
}

func filePaths(files []ChangedFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

func runCheck(repo, id string) CheckResult {
	errs := []string{}
	switch id {
	case "go-test":
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			errs = append(errs, strings.TrimSpace(string(out)))
		}
	case "kit-smoke":
		entries, err := kit.LoadRegistry(repo)
		if err != nil {
			errs = append(errs, err.Error())
			break
		}
		selected, err := kit.SelectKits(entries, "", true)
		if err != nil {
			errs = append(errs, err.Error())
			break
		}
		for _, r := range kit.SmokeKits(repo, selected) {
			if !r.OK {
				for _, e := range r.Errors {
					errs = append(errs, r.ID+": "+e)
				}
			}
		}
	case "governance-lint":
		errs = append(errs, governance.Lint(repo, "all", "")...)
	case "verify-hooks":
		_, es := repohealth.VerifyHooks(repo)
		errs = append(errs, es...)
	case "verify-repo-text":
		_, es := repohealth.VerifyRepoText(repo)
		errs = append(errs, es...)
	case "verify-release-notes":
		_, es := repohealth.VerifyReleaseNotes(repo)
		errs = append(errs, es...)
	case "doctor-perf":
		_ = docsync.LintStaged(repo)
	default:
		errs = append(errs, "unknown check: "+id)
	}
	return CheckResult{ID: id, OK: len(errs) == 0, Errors: compactErrors(errs)}
}

func compactErrors(errs []string) []string {
	out := []string{}
	for _, e := range errs {
		e = strings.TrimSpace(e)
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}
