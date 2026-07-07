package releasegate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksReleaseStructure(t *testing.T) {
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

	result, errs := Verify(repo)
	if len(errs) != 0 {
		t.Fatalf("Verify errs = %v", errs)
	}
	if !result.OK || len(result.Checks) == 0 {
		t.Fatalf("unexpected verify result: %#v", result)
	}
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
