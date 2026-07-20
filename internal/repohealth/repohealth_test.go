package repohealth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestHooksWiredDetectsUnwiredThenWired(t *testing.T) {
	repo := t.TempDir()
	if _, err := gitx.Run(repo, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	// Files existing is not enough — an unwired clone must be flagged.
	data, warnings := HooksWired(repo)
	if len(warnings) == 0 || data["wired"] != false {
		t.Fatalf("expected unwired warning, got data=%v warnings=%v", data, warnings)
	}
	// Leverage git's own config: once core.hooksPath points to .githooks, it's wired.
	if _, err := gitx.Run(repo, "config", "core.hooksPath", ".githooks"); err != nil {
		t.Fatal(err)
	}
	data, warnings = HooksWired(repo)
	if len(warnings) != 0 || data["wired"] != true {
		t.Fatalf("expected wired with no warning, got data=%v warnings=%v", data, warnings)
	}
}

func TestCategorizePwshPriority(t *testing.T) {
	cases := []struct {
		path string
		line string
		want string
	}{
		{"Taskfile.yml", "bin/aicoding.exe docsync ci --json", "unknown"},
		{"Taskfile.yml", "bin/aicoding.exe kit verify --all --profile Smoke --json", "verify"},
		{"Taskfile.yml", "bin/aicoding.exe fresh-clone --profile Release --json", "release"},
		{"README.md", "pwsh -File tools/specialty/uninstall-safety-profile.ps1", "uninstall"},
		{"README.md", "TI DSS / XDS / flash / erase / write-memory", "dss"},
	}
	for _, tc := range cases {
		if got := categorizePwsh(tc.path, tc.line); got != tc.want {
			t.Fatalf("categorizePwsh(%q, %q) = %q, want %q", tc.path, tc.line, got, tc.want)
		}
	}
}

func TestIsPwshInvocationLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"pwsh -File tools/specialty/verify-release-governance-overlay.ps1", true},
		{"\"type\": \"specialty-pwsh\"", true},
		{"if (Get-Command pwsh -ErrorAction SilentlyContinue) {}", true},
		{"PowerShell / Python slow path remains available", false},
		{"默认使用 PowerShell 7（`pwsh`）执行仓库安装", false},
	}
	for _, tc := range cases {
		if got := isPwshInvocationLine(strings.ToLower(tc.line)); got != tc.want {
			t.Fatalf("isPwshInvocationLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestVerifyHooksFastFirst(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit\npwsh -File tools/specialty/lint.ps1\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "commit-msg"), "bin/aicoding.exe hook commit-msg --file $1\npwsh -File tools/specialty/lint.ps1\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-push"), "bin/aicoding.exe hook pre-push\n")
	checks, errs := VerifyHooks(repo)
	if len(errs) != 0 {
		t.Fatalf("VerifyHooks errs = %v", errs)
	}
	if len(checks) != 3 {
		t.Fatalf("VerifyHooks checks = %d, want 3", len(checks))
	}
	for _, c := range checks {
		if !c.FastFirst {
			t.Fatalf("%s did not detect fast-first hook", c.Path)
		}
	}
}

func TestReleaseNotesBodyErrors(t *testing.T) {
	if errs := releaseNotesBodyErrors("```powershell\ntask smoke\n```\n"); len(errs) != 0 {
		t.Fatalf("releaseNotesBodyErrors valid body = %v", errs)
	}
	if errs := releaseNotesBodyErrors("`powershell\ntask smoke\n`\n"); !hasRepohealthError(errs, "single-backtick") {
		t.Fatalf("expected single-backtick error, got %v", errs)
	}
	if errs := releaseNotesBodyErrors("bad \uFFFD text"); !hasRepohealthError(errs, "control or replacement") {
		t.Fatalf("expected replacement character error, got %v", errs)
	}
}

func TestVerifyReleaseNotes(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "CHANGELOG.md"), "# CHANGELOG\n\n## [Unreleased]\n")
	mustWrite(t, filepath.Join(repo, ".github", "RELEASE_TEMPLATE.md"), "## 摘要 / Summary\n\n## 变更内容 / What's Changed\n\n## 可追溯性 / Traceability\n")
	mustWrite(t, filepath.Join(repo, "docs", "governance", "TAGGING_POLICY.md"), "vMAJOR.MINOR.PATCH\nkit/<kit-id>/vMAJOR.MINOR.PATCH\nmilestone/YYYY.MM.DD-<name>\n")
	mustWrite(t, filepath.Join(repo, "docs", "governance", "RELEASE_POLICY.md"), "Platform Release\nKit / Component Release\nMilestone Release\n")
	for _, rel := range []string{
		"docs/governance/RELEASE_GOVERNANCE_OVERLAY.md",
		"tools/specialty/aicoding-tag-governance.ps1",
		"tools/specialty/verify-release-governance-overlay.ps1",
		"config/tagging-policy.json",
		"config/kits/release-governance-overlay-kit.json",
		"Taskfile.yml",
		".aicoding/templates/perf-cache-plan.json",
	} {
		mustWrite(t, filepath.Join(repo, filepath.FromSlash(rel)), "ok\n")
	}
	_, errs := VerifyReleaseNotes(repo)
	if len(errs) != 0 {
		t.Fatalf("VerifyReleaseNotes errs = %v", errs)
	}
}

func hasRepohealthError(errs []string, needle string) bool {
	for _, err := range errs {
		if strings.Contains(err, needle) {
			return true
		}
	}
	return false
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
