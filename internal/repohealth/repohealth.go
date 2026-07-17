package repohealth

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type PwshCall struct {
	Path                 string `json:"path"`
	Line                 int    `json:"line"`
	Text                 string `json:"text"`
	Category             string `json:"category"`
	Recommendation       string `json:"recommendation"`
	SuggestedReplacement string `json:"suggestedReplacement,omitempty"`
}

type HookCheck struct {
	Path       string   `json:"path"`
	Exists     bool     `json:"exists"`
	FastFirst  bool     `json:"fastFirst"`
	FastTokens []string `json:"fastTokens,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

type TextCheck struct {
	Path       string   `json:"path"`
	OK         bool     `json:"ok"`
	Size       int      `json:"size"`
	LineEnding string   `json:"lineEnding"`
	Warnings   []string `json:"warnings,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

type ReleaseNotesCheck struct {
	Path   string   `json:"path"`
	OK     bool     `json:"ok"`
	Errors []string `json:"errors,omitempty"`
}

type ToolStatus struct {
	Name    string `json:"name"`
	Found   bool   `json:"found"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

type ManifestStatus struct {
	ID       string   `json:"id"`
	Enabled  bool     `json:"enabled"`
	Manifest string   `json:"manifest"`
	OK       bool     `json:"ok"`
	Errors   []string `json:"errors,omitempty"`
}

type RepoStatus struct {
	RepoRoot         string           `json:"repoRoot"`
	Branch           string           `json:"branch,omitempty"`
	BinAicoding      bool             `json:"binAicoding"`
	BinAicodingPath  string           `json:"binAicodingPath,omitempty"`
	GoVersion        string           `json:"goVersion,omitempty"`
	GitVersion       string           `json:"gitVersion,omitempty"`
	Tools            []ToolStatus     `json:"tools"`
	Manifests        []ManifestStatus `json:"manifests"`
	RegistryKitCount int              `json:"registryKitCount"`
	EnabledKitCount  int              `json:"enabledKitCount"`
}

func ScanPwsh(repo string) ([]PwshCall, []string) {
	files, errs := pwshScanFiles(repo)
	calls := []PwshCall{}
	for _, rel := range files {
		content, err := os.ReadFile(platform.RepoPath(repo, rel))
		if err != nil {
			errs = append(errs, "cannot read "+rel+": "+err.Error())
			continue
		}
		for i, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
			lower := strings.ToLower(line)
			if !isPwshInvocationLine(lower) {
				continue
			}
			category := categorizePwsh(rel, line)
			recommendation, replacement := routeAdvice(category, rel, line)
			calls = append(calls, PwshCall{
				Path:                 rel,
				Line:                 i + 1,
				Text:                 strings.TrimSpace(line),
				Category:             category,
				Recommendation:       recommendation,
				SuggestedReplacement: replacement,
			})
		}
	}
	return calls, errs
}

func isPwshInvocationLine(lower string) bool {
	trimmed := strings.TrimSpace(lower)
	if strings.Contains(lower, "specialty-pwsh") {
		return true
	}
	for _, token := range []string{"powershell.exe", "powershell -", "powershell\t-", "powershell`"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	if strings.HasPrefix(trimmed, "pwsh ") || strings.HasPrefix(trimmed, "pwsh\t") || strings.HasPrefix(trimmed, "pwsh -") {
		return true
	}
	for _, token := range []string{" pwsh ", " pwsh\t", " pwsh -", "`pwsh ", "\"pwsh\""} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func VerifyHooks(repo string) ([]HookCheck, []string) {
	hooks := []string{".githooks/pre-commit", ".githooks/commit-msg"}
	checks := []HookCheck{}
	allErrs := []string{}
	for _, rel := range hooks {
		check := HookCheck{Path: rel}
		p := platform.RepoPath(repo, rel)
		if !platform.IsFile(p) {
			check.Errors = append(check.Errors, "hook missing")
			allErrs = append(allErrs, rel+": hook missing")
			checks = append(checks, check)
			continue
		}
		check.Exists = true
		content, err := platform.ReadText(p)
		if err != nil {
			check.Errors = append(check.Errors, err.Error())
			allErrs = append(allErrs, rel+": "+err.Error())
			checks = append(checks, check)
			continue
		}
		fastTokens := []string{"bin/aicoding", "bin/aicoding.exe", "go run ./cmd/aicoding"}
		slowTokens := []string{"pwsh", "powershell.exe"}
		fastAt := firstIndex(content, fastTokens)
		slowAt := firstIndex(content, slowTokens)
		for _, token := range fastTokens {
			if strings.Contains(content, token) {
				check.FastTokens = append(check.FastTokens, token)
			}
		}
		check.FastFirst = fastAt >= 0 && (slowAt < 0 || fastAt < slowAt)
		if !check.FastFirst {
			check.Errors = append(check.Errors, "hook does not prefer bin/aicoding or Go fast path before PowerShell fallback")
			allErrs = append(allErrs, rel+": hook does not prefer Go fast path")
		}
		checks = append(checks, check)
	}
	return checks, allErrs
}

func VerifyRepoText(repo string) ([]TextCheck, []string) {
	files, errs := repoTextFiles(repo)
	checks := []TextCheck{}
	for _, rel := range files {
		check := checkTextFile(repo, rel)
		if !check.OK {
			for _, e := range check.Errors {
				errs = append(errs, rel+": "+e)
			}
		}
		checks = append(checks, check)
	}
	return checks, errs
}

func VerifyReleaseNotes(repo string) ([]ReleaseNotesCheck, []string) {
	items := []ReleaseNotesCheck{}
	errs := []string{}
	checkContains := func(rel string, needles ...string) {
		item := ReleaseNotesCheck{Path: rel, OK: true}
		text, err := platform.ReadText(platform.RepoPath(repo, rel))
		if err != nil {
			item.OK = false
			item.Errors = append(item.Errors, "missing or unreadable: "+err.Error())
		} else {
			for _, needle := range needles {
				if !strings.Contains(text, needle) {
					item.OK = false
					item.Errors = append(item.Errors, "missing required text: "+needle)
				}
			}
		}
		if !item.OK {
			for _, e := range item.Errors {
				errs = append(errs, rel+": "+e)
			}
		}
		items = append(items, item)
	}
	checkExists := func(rel string) {
		item := ReleaseNotesCheck{Path: rel, OK: platform.IsFile(platform.RepoPath(repo, rel))}
		if !item.OK {
			item.Errors = []string{"missing required release-governance overlay file"}
			errs = append(errs, rel+": missing required release-governance overlay file")
		}
		items = append(items, item)
	}
	checkReleaseNotesBody := func(rel string, needles ...string) {
		item := ReleaseNotesCheck{Path: rel, OK: true}
		text, err := platform.ReadText(platform.RepoPath(repo, rel))
		if err != nil {
			item.OK = false
			item.Errors = append(item.Errors, "missing or unreadable: "+err.Error())
		} else {
			for _, needle := range needles {
				if !strings.Contains(text, needle) {
					item.OK = false
					item.Errors = append(item.Errors, "missing required text: "+needle)
				}
			}
			for _, e := range releaseNotesBodyErrors(text) {
				item.OK = false
				item.Errors = append(item.Errors, e)
			}
		}
		if !item.OK {
			for _, e := range item.Errors {
				errs = append(errs, rel+": "+e)
			}
		}
		items = append(items, item)
	}
	checkContains("CHANGELOG.md", "[Unreleased]")
	checkReleaseNotesBody(".github/RELEASE_TEMPLATE.md", "摘要 / Summary", "变更内容 / What's Changed", "可追溯性 / Traceability")
	checkContains("docs/governance/TAGGING_POLICY.md", "vMAJOR.MINOR.PATCH", "kit/<kit-id>/vMAJOR.MINOR.PATCH", "milestone/YYYY.MM.DD-<name>")
	checkContains("docs/governance/RELEASE_POLICY.md", "Platform Release", "Kit / Component Release", "Milestone Release")
	for _, rel := range []string{
		"docs/governance/RELEASE_GOVERNANCE_OVERLAY.md",
		"tools/specialty/aicoding-tag-governance.ps1",
		"tools/specialty/verify-release-governance-overlay.ps1",
		"config/tagging-policy.json",
		"config/kits/release-governance-overlay-kit.json",
		"Taskfile.yml",
		".aicoding/templates/perf-cache-plan.json",
	} {
		checkExists(rel)
	}
	return items, errs
}

func StatusAll(repo string) (RepoStatus, []string) {
	status := RepoStatus{RepoRoot: repo}
	errs := []string{}
	branch, err := gitx.Run(repo, "branch", "--show-current")
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		status.Branch = strings.TrimSpace(branch)
	}
	for _, rel := range []string{"bin/aicoding.exe", "bin/aicoding"} {
		if platform.IsFile(platform.RepoPath(repo, rel)) {
			status.BinAicoding = true
			status.BinAicodingPath = rel
			break
		}
	}
	status.GoVersion = commandVersion("go", "version")
	status.GitVersion = commandVersion("git", "--version")
	status.Tools = discoverTools([]string{"python", "python3", "apatch", "airepair", "task", "pwsh"})
	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		errs = append(errs, "cannot load kit registry: "+err.Error())
		return status, errs
	}
	status.RegistryKitCount = len(entries)
	for _, e := range entries {
		if e.Enabled {
			status.EnabledKitCount++
		}
		ms := ManifestStatus{ID: e.ID, Enabled: e.Enabled, Manifest: e.Manifest, OK: true}
		m, err := kit.LoadManifest(repo, e.Manifest)
		if err != nil {
			ms.OK = false
			ms.Errors = append(ms.Errors, "cannot load manifest: "+err.Error())
		} else {
			if m.ID != e.ID {
				ms.OK = false
				ms.Errors = append(ms.Errors, "manifest id mismatch: "+m.ID)
			}
			for action, cmd := range m.Commands {
				for _, rel := range cmd.RequiredPaths {
					if !platform.Exists(platform.RepoPath(repo, rel)) {
						ms.OK = false
						ms.Errors = append(ms.Errors, action+": missing required path: "+rel)
					}
				}
				if cmd.Type == "specialty-pwsh" && cmd.Path != "" && !platform.IsFile(platform.RepoPath(repo, cmd.Path)) {
					ms.OK = false
					ms.Errors = append(ms.Errors, action+": missing script: "+cmd.Path)
				}
			}
		}
		if !ms.OK {
			for _, e := range ms.Errors {
				errs = append(errs, ms.ID+": "+e)
			}
		}
		status.Manifests = append(status.Manifests, ms)
	}
	return status, errs
}

func pwshScanFiles(repo string) ([]string, []string) {
	seen := map[string]bool{}
	files := []string{}
	errs := []string{}
	add := func(rel string) {
		rel = filepath.ToSlash(rel)
		if !seen[rel] && platform.IsFile(platform.RepoPath(repo, rel)) {
			seen[rel] = true
			files = append(files, rel)
		}
	}
	for _, rel := range []string{"README.md", "README_CN.md", "README_EN.md", "Taskfile.yml"} {
		add(rel)
	}
	addGlob := func(pattern string) {
		matches, err := filepath.Glob(platform.RepoPath(repo, pattern))
		if err != nil {
			errs = append(errs, "bad glob "+pattern+": "+err.Error())
			return
		}
		for _, p := range matches {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				rel, _ := filepath.Rel(repo, p)
				add(rel)
			}
		}
	}
	addGlob(".githooks/*")
	addGlob("config/kits/*.json")
	addGlob("tools/specialty/*.ps1")
	sort.Strings(files)
	return files, errs
}

func repoTextFiles(repo string) ([]string, []string) {
	seen := map[string]bool{}
	files := []string{}
	errs := []string{}
	add := func(rel string) {
		rel = filepath.ToSlash(rel)
		if !seen[rel] && platform.IsFile(platform.RepoPath(repo, rel)) {
			seen[rel] = true
			files = append(files, rel)
		}
	}
	for _, rel := range []string{"README.md", "README_CN.md", "README_EN.md", "CHANGELOG.md"} {
		add(rel)
	}
	docsRoot := platform.RepoPath(repo, "docs")
	if platform.IsDir(docsRoot) {
		if err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".md") {
				rel, _ := filepath.Rel(repo, path)
				add(rel)
			}
			return nil
		}); err != nil {
			errs = append(errs, "walk docs: "+err.Error())
		}
	}
	sort.Strings(files)
	return files, errs
}

func checkTextFile(repo, rel string) TextCheck {
	p := platform.RepoPath(repo, rel)
	b, err := os.ReadFile(p)
	check := TextCheck{Path: rel, OK: true}
	if err != nil {
		check.OK = false
		check.Errors = append(check.Errors, err.Error())
		return check
	}
	check.Size = len(b)
	if len(bytes.TrimSpace(b)) == 0 {
		check.OK = false
		check.Errors = append(check.Errors, "empty file")
	}
	if bytes.Contains(b, []byte{0}) {
		check.OK = false
		check.Errors = append(check.Errors, "NUL byte found")
	}
	if !utf8.Valid(b) {
		check.OK = false
		check.Errors = append(check.Errors, "invalid UTF-8")
	}
	s := string(b)
	check.LineEnding = lineEnding(s)
	if check.LineEnding == "mixed" {
		check.Warnings = append(check.Warnings, "mixed CRLF/LF line endings")
	}
	for _, marker := range []string{"<<<<<<< ", "=======", ">>>>>>> "} {
		if strings.Contains(s, marker) {
			check.OK = false
			check.Errors = append(check.Errors, "conflict marker found: "+strings.TrimSpace(marker))
		}
	}
	return check
}

func releaseNotesBodyErrors(s string) []string {
	errs := []string{}
	for _, r := range s {
		if r == '\uFFFD' || (r < 0x20 && r != '\n' && r != '\r' && r != '\t') {
			errs = append(errs, "control or replacement character found")
			break
		}
	}
	if strings.Count(s, "```")%2 != 0 {
		errs = append(errs, "unbalanced Markdown code fences")
	}
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "`") && !strings.HasPrefix(trimmed, "```") {
			switch trimmed {
			case "`", "`powershell", "`bash", "`text", "`json":
				errs = append(errs, "single-backtick command fence found")
				return errs
			}
		}
	}
	return errs
}

func lineEnding(s string) string {
	crlf := strings.Contains(s, "\r\n")
	withoutCRLF := strings.ReplaceAll(s, "\r\n", "")
	lf := strings.Contains(withoutCRLF, "\n")
	switch {
	case crlf && lf:
		return "mixed"
	case crlf:
		return "crlf"
	case lf:
		return "lf"
	default:
		return "none"
	}
}

func categorizePwsh(path, line string) string {
	lower := strings.ToLower(path + " " + line)
	if containsAny(lower, "dss", "xds", "flash", "erase", "write-memory", "loadprogram", "halt", "reset") {
		return "dss"
	}
	if containsAny(lower, "release", "profile release", "fresh-clone") {
		return "release"
	}
	if strings.Contains(lower, "uninstall") {
		return "uninstall"
	}
	if strings.Contains(lower, "install") {
		return "install"
	}
	if strings.Contains(lower, "rollback") {
		return "rollback"
	}
	if strings.Contains(lower, "export") {
		return "export"
	}
	if strings.Contains(lower, "status") {
		return "status"
	}
	if containsAny(lower, "verify", "check", "lint", "doctor") {
		return "verify"
	}
	if strings.Contains(lower, "test") {
		return "test"
	}
	return "unknown"
}

func routeAdvice(category, path, line string) (string, string) {
	lower := strings.ToLower(path + " " + line)
	if containsAny(lower, "profile full", "profile release", "test-kit-fresh-clone", " export ", " install", " uninstall", " rollback", "dss", "xds", "flash", "erase", "write-memory") {
		return "keep-pwsh", "keep PowerShell/Python slow path"
	}
	if strings.Contains(lower, "aicoding-kit.ps1") && strings.Contains(lower, "profile smoke") {
		return "go-now", "bin/aicoding.exe kit verify --all --profile Smoke --json"
	}
	if strings.Contains(lower, "governance lint") {
		return "go-now", "bin/aicoding.exe governance lint --json"
	}
	if strings.Contains(lower, "check-documentation-sync.ps1") && (strings.Contains(lower, "pre-commit") || strings.Contains(lower, "staged")) {
		return "go-now", "bin/aicoding.exe hook pre-commit --json"
	}
	if strings.Contains(lower, "verify hooks") {
		return "go-now", "bin/aicoding.exe verify --profile Smoke --json"
	}
	if strings.Contains(lower, "verify-release-governance-overlay.ps1") {
		return "keep-pwsh", "keep PowerShell release-governance overlay slow path"
	}
	if strings.Contains(lower, "verify release-notes") {
		return "go-now", "bin/aicoding.exe verify release-notes --json"
	}
	if category == "status" {
		return "go-now", "bin/aicoding.exe doctor --all --json"
	}
	if category == "verify" || category == "test" {
		return "go-orchestrate", "prefer bin/aicoding.exe verify --profile Smoke or test --profile Smoke before PowerShell fallback"
	}
	if category == "unknown" {
		return "go-orchestrate", "review whether this can become a Go orchestration check"
	}
	return "keep-pwsh", "keep PowerShell/Python slow path"
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func firstIndex(s string, tokens []string) int {
	best := -1
	lower := strings.ToLower(s)
	for _, token := range tokens {
		idx := strings.Index(lower, strings.ToLower(token))
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
}

func commandVersion(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func discoverTools(names []string) []ToolStatus {
	out := []ToolStatus{}
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		path, err := exec.LookPath(name)
		item := ToolStatus{Name: name, Found: err == nil}
		if err == nil {
			item.Path = path
		}
		out = append(out, item)
	}
	return out
}

func ErrorOrNil(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func FormatCount(name string, count int) string {
	return fmt.Sprintf("%s=%d", name, count)
}

type PwshBudget struct {
	Calls  []PwshBudgetCall `json:"calls"`
	Counts map[string]int   `json:"counts"`
}

type PwshBudgetCall struct {
	Path                 string `json:"path"`
	Line                 int    `json:"line"`
	Text                 string `json:"text"`
	Category             string `json:"category"`
	Budget               string `json:"budget"`
	Recommendation       string `json:"recommendation"`
	SuggestedReplacement string `json:"suggestedReplacement,omitempty"`
}

func ScanPwshBudget(repo string) (PwshBudget, []string) {
	files, errs := pwshBudgetScanFiles(repo)
	budget := PwshBudget{Counts: map[string]int{}}
	for _, rel := range files {
		content, err := os.ReadFile(platform.RepoPath(repo, rel))
		if err != nil {
			errs = append(errs, "cannot read "+rel+": "+err.Error())
			continue
		}
		for i, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
			lower := strings.ToLower(line)
			if !isPwshInvocationLine(lower) {
				continue
			}
			category := categorizePwsh(rel, line)
			recommendation, replacement := routeAdvice(category, rel, line)
			bucket := classifyPwshBudget(rel, line, category, recommendation)
			budget.Calls = append(budget.Calls, PwshBudgetCall{
				Path:                 rel,
				Line:                 i + 1,
				Text:                 strings.TrimSpace(line),
				Category:             category,
				Budget:               bucket,
				Recommendation:       recommendation,
				SuggestedReplacement: replacement,
			})
			budget.Counts[bucket]++
		}
	}
	for _, bucket := range []string{"hot-path", "slow-path", "fallback", "documentation-only"} {
		if _, ok := budget.Counts[bucket]; !ok {
			budget.Counts[bucket] = 0
		}
	}
	return budget, errs
}

func pwshBudgetScanFiles(repo string) ([]string, []string) {
	seen := map[string]bool{}
	files := []string{}
	errs := []string{}
	add := func(rel string) {
		rel = filepath.ToSlash(rel)
		if !seen[rel] && platform.IsFile(platform.RepoPath(repo, rel)) {
			seen[rel] = true
			files = append(files, rel)
		}
	}
	add("Taskfile.yml")
	addGlob := func(pattern string) {
		matches, err := filepath.Glob(platform.RepoPath(repo, pattern))
		if err != nil {
			errs = append(errs, "bad glob "+pattern+": "+err.Error())
			return
		}
		for _, p := range matches {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				rel, _ := filepath.Rel(repo, p)
				add(rel)
			}
		}
	}
	addGlob(".githooks/*")
	addGlob(".github/workflows/*")
	addGlob("tools/specialty/*.ps1")
	docsRoot := platform.RepoPath(repo, "docs")
	if platform.IsDir(docsRoot) {
		if err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".md") {
				rel, _ := filepath.Rel(repo, path)
				add(rel)
			}
			return nil
		}); err != nil {
			errs = append(errs, "walk docs: "+err.Error())
		}
	}
	sort.Strings(files)
	return files, errs
}

func classifyPwshBudget(path, line, category, recommendation string) string {
	lowerPath := strings.ToLower(path)
	lower := strings.ToLower(path + " " + line)
	if strings.HasPrefix(lowerPath, "docs/") {
		return "documentation-only"
	}
	if strings.HasPrefix(lowerPath, ".githooks/") {
		return "fallback"
	}
	if strings.Contains(lower, "||") || strings.Contains(lower, "fallback") {
		return "fallback"
	}
	if strings.Contains(lower, "verify-release-governance-overlay.ps1") || strings.Contains(lower, "verify-skills") {
		return "slow-path"
	}

	if containsAny(lower, "profile full", "profile release", "test-kit-fresh-clone", " export ", " install", " uninstall", " rollback", "dss", "xds", "flash", "erase", "write-memory", "psscriptanalyzer") {
		return "slow-path"
	}
	if recommendation == "go-now" || containsAny(lower, "profile smoke", "task smoke", "governance lint", "doctor --all", "verify --profile", "verify release-notes", "verify repo-text") {
		return "hot-path"
	}
	if category == "release" || category == "install" || category == "uninstall" || category == "rollback" || category == "export" || category == "dss" {
		return "slow-path"
	}
	return "fallback"
}
