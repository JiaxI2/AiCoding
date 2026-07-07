package workflow

import "testing"

func TestPlanForFilesSelectsFastChecksByFileType(t *testing.T) {
	plan := PlanForFiles([]string{
		"docs/FAST_PATH_COMMANDS.md",
		".github/RELEASE_TEMPLATE.md",
		".githooks/pre-commit",
		"config/kits/aicoding-platform.json",
		"cmd/aicoding/main.go",
	})

	for _, id := range []string{"go-test", "kit-smoke", "governance-lint", "verify-hooks", "verify-repo-text", "verify-release-notes"} {
		if !plan.HasCheck(id) {
			t.Fatalf("expected check %q in %#v", id, plan.Checks)
		}
	}
	if plan.HasCheck("full") || plan.HasCheck("release") {
		t.Fatalf("smart verify must not select Full/Release slow path: %#v", plan.Checks)
	}
}

func TestPlanForFilesUsesSmokePlanWhenNoChanges(t *testing.T) {
	plan := PlanForFiles(nil)
	if !plan.HasCheck("kit-smoke") || !plan.HasCheck("doctor-perf") {
		t.Fatalf("expected default smoke checks, got %#v", plan.Checks)
	}
}
