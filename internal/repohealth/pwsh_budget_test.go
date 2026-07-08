package repohealth

import (
	"path/filepath"
	"testing"
)

func TestScanPwshBudgetClassifiesInvocationPoints(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "tasks:\n  smoke:\n    cmds:\n      - bin/aicoding.exe kit verify --all --profile Smoke --json\n  full:\n    cmds:\n      - bin/aicoding.exe full --json\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit --json\n")
	mustWrite(t, filepath.Join(repo, "docs", "POWERSHELL_MIGRATION.md"), "pwsh -File scripts/verify-release-governance-overlay.ps1\n")

	budget, errs := ScanPwshBudget(repo)
	if len(errs) != 0 {
		t.Fatalf("ScanPwshBudget errs = %v", errs)
	}
	if budget.Counts["hot-path"] != 0 || budget.Counts["slow-path"] != 0 || budget.Counts["fallback"] != 0 {
		t.Fatalf("expected no PowerShell hot/slow/fallback routes after Go routing, got %#v", budget.Counts)
	}
	if budget.Counts["documentation-only"] == 0 {
		t.Fatalf("expected documentation-only PowerShell inventory, got %#v", budget.Counts)
	}
}
