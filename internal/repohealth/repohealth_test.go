package repohealth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCategorizePwshPriority(t *testing.T) {
	cases := []struct {
		path string
		line string
		want string
	}{
		{"Taskfile.yml", "pwsh -File scripts/legacy/fast-path-replaced/status-codex-kit.ps1 -Json", "status"},
		{"Taskfile.yml", "pwsh -File scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json", "test"},
		{"Taskfile.yml", "pwsh -File scripts/test-kit-fresh-clone.ps1 -Profile Release -Json", "release"},
		{"README.md", "pwsh -File scripts/uninstall-codex-kit.ps1", "uninstall"},
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
		{"pwsh -File scripts/aicoding-kit.ps1", true},
		{"\"type\": \"powershell-script\"", true},
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
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit\npwsh -File scripts/lint.ps1\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "commit-msg"), "go run ./cmd/aicoding hook commit-msg --file $1\npwsh -File scripts/lint.ps1\n")
	checks, errs := VerifyHooks(repo)
	if len(errs) != 0 {
		t.Fatalf("VerifyHooks errs = %v", errs)
	}
	if len(checks) != 2 {
		t.Fatalf("VerifyHooks checks = %d, want 2", len(checks))
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
	mustWrite(t, filepath.Join(repo, "docs", "TAGGING_POLICY.md"), "vMAJOR.MINOR.PATCH\nkit/<kit-id>/vMAJOR.MINOR.PATCH\nmilestone/YYYY.MM.DD-<name>\n")
	mustWrite(t, filepath.Join(repo, "docs", "RELEASE_POLICY.md"), "Platform Release\nKit / Component Release\nMilestone / Historical Snapshot\n")
	for _, rel := range []string{
		"docs/RELEASE_GOVERNANCE_OVERLAY.md",
		"scripts/aicoding-tag-governance.ps1",
		"scripts/verify-release-governance-overlay.ps1",
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
