package repohealth

import (
	"path/filepath"
	"testing"
)

func TestScanPwshBudgetClassifiesInvocationPoints(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "tasks:\n  smoke:\n    cmds:\n      - pwsh -File scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json\n  full:\n    cmds:\n      - pwsh -File scripts/aicoding-kit.ps1 test -All -Profile Full -Json\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit --json || pwsh -File scripts/lint-git-governance.ps1\n")
	mustWrite(t, filepath.Join(repo, "docs", "POWERSHELL_MIGRATION.md"), "pwsh -File scripts/install-codex-kit.ps1\n")

	budget, errs := ScanPwshBudget(repo)
	if len(errs) != 0 {
		t.Fatalf("ScanPwshBudget errs = %v", errs)
	}
	for _, category := range []string{"hot-path", "slow-path", "fallback", "documentation-only"} {
		if budget.Counts[category] == 0 {
			t.Fatalf("expected category %q in %#v", category, budget.Counts)
		}
	}
}
