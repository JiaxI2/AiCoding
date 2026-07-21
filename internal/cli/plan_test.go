package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	plancheck "github.com/JiaxI2/AiCoding/internal/plan"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func TestRunPlanCheckStagedSensitiveAndExemptPaths(t *testing.T) {
	repo := initWorkTestRepo(t)
	mustWrite(t, filepath.Join(repo, filepath.FromSlash(plancheck.PolicyPath)), `{
  "schemaVersion": 1,
  "sensitivePaths": [{"pattern":"internal/cli/**","reason":"frozen kernel"}],
  "exemptPaths": ["docs/todolist/**"]
}`)
	mustWrite(t, filepath.Join(repo, "docs", "spec", "coverage-plan", "PLAN.md"), `---
id: coverage-plan
status: draft
scope:
  - internal/cli/**
approvedTree: ""
decision: ""
gates:
  - profile: full
---

# Coverage
`)
	gitWorkTest(t, repo, "add", plancheck.PolicyPath, "docs/spec/coverage-plan/PLAN.md")
	gitWorkTest(t, repo, "commit", "-m", "test: add plan policy")

	mustWrite(t, filepath.Join(repo, "internal", "cli", "nested", "change.go"), "package nested\n")
	gitWorkTest(t, repo, "add", "internal/cli/nested/change.go")
	sensitive, err := runPlan([]string{"check", "--staged", "--repo-root", repo, "--json"}, time.Now())
	if err == nil || sensitive.OK || sensitive.ErrorKind != report.ErrorKindValidation {
		t.Fatalf("sensitive staged path was not blocked: result=%#v err=%v", sensitive, err)
	}
	check, ok := sensitive.Data.(plancheck.Check)
	if !ok || len(check.Sensitive) != 1 || check.Sensitive[0].Path != "internal/cli/nested/change.go" || check.RequiredAction == "" {
		t.Fatalf("unexpected sensitive projection: %#v", sensitive.Data)
	}
	mustWrite(t, filepath.Join(repo, "docs", "spec", "coverage-plan", "PLAN.md"), `---
id: coverage-plan
status: approved
scope:
  - internal/cli/**
approvedTree: "`+strings.Repeat("a", 40)+`"
decision: ""
gates:
  - profile: full
---

# Coverage
`)
	covered, err := runPlan([]string{"check", "--staged", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !covered.OK {
		t.Fatalf("approved scope did not cover sensitive path: result=%#v err=%v", covered, err)
	}
	check = covered.Data.(plancheck.Check)
	if len(check.ApprovedPlans) != 1 || check.ApprovedPlans[0] != "coverage-plan" || len(check.Uncovered) != 0 {
		t.Fatalf("unexpected approved coverage: %#v", check)
	}

	gitWorkTest(t, repo, "reset")
	gitWorkTest(t, repo, "restore", "docs/spec/coverage-plan/PLAN.md")
	mustWrite(t, filepath.Join(repo, "docs", "todolist", "0004.md"), "exempt\n")
	gitWorkTest(t, repo, "add", "docs/todolist/0004.md")
	exempt, err := runPlan([]string{"check", "--staged", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !exempt.OK {
		t.Fatalf("exempt staged path failed: result=%#v err=%v", exempt, err)
	}
	check = exempt.Data.(plancheck.Check)
	if len(check.Sensitive) != 0 || len(check.Exempt) != 1 || check.Exempt[0] != "docs/todolist/0004.md" {
		t.Fatalf("unexpected exempt projection: %#v", check)
	}
}

func TestRunPlanCheckRequiresOnePathSource(t *testing.T) {
	for _, args := range [][]string{
		{"check"},
		{"check", "--staged", "--paths", "README.md"},
		{"approve", "--staged"},
	} {
		if _, err := runPlan(args, time.Now()); err == nil {
			t.Fatalf("runPlan(%v) unexpectedly succeeded", args)
		}
	}
}

func TestRunPlanVerifyAndStatusProjectFrontmatter(t *testing.T) {
	repo := t.TempDir()
	planFile := filepath.Join(repo, "docs", "spec", "sample-plan", "PLAN.md")
	mustWrite(t, planFile, `---
id: sample-plan
status: archived
scope:
  - internal/plan/**
approvedTree: ""
decision: ""
gates:
  - profile: full
---

# Sample
`)
	verified, err := runPlan([]string{"verify", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !verified.OK {
		t.Fatalf("plan verify failed: result=%#v err=%v", verified, err)
	}
	status, err := runPlan([]string{"status", "--all", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !status.OK {
		t.Fatalf("plan status failed: result=%#v err=%v", status, err)
	}
	views, ok := status.Data.([]planStatusView)
	if !ok || len(views) != 1 || views[0].Spec.ID != "sample-plan" || views[0].Spec.Status != plancheck.StatusArchived {
		t.Fatalf("unexpected status data: %#v", status.Data)
	}
	selected, err := runPlan([]string{"status", "--id", "sample-plan", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !selected.OK || len(selected.Data.([]planStatusView)) != 1 {
		t.Fatalf("plan status --id failed: result=%#v err=%v", selected, err)
	}
}

func TestRunPlanApproveBindsCleanTree(t *testing.T) {
	repo := initWorkTestRepo(t)
	mustWrite(t, filepath.Join(repo, "docs", "spec", "sample-plan", "PLAN.md"), `---
id: sample-plan
status: draft
scope:
  - internal/plan/**
approvedTree: ""
decision: ""
gates:
  - profile: full
---

# Sample
`)
	gitWorkTest(t, repo, "add", "docs/spec/sample-plan/PLAN.md")
	gitWorkTest(t, repo, "commit", "-m", "test: add plan")
	wantTree := gitWorkTest(t, repo, "rev-parse", "HEAD^{tree}")
	mustWrite(t, filepath.Join(repo, "dirty.txt"), "dirty\n")
	dirty, dirtyErr := runPlan([]string{"approve", "--id", "sample-plan", "--repo-root", repo, "--json"}, time.Now())
	if dirtyErr == nil || dirty.OK || !strings.Contains(dirtyErr.Error(), "clean worktree") {
		t.Fatalf("dirty plan approve was not rejected: result=%#v err=%v", dirty, dirtyErr)
	}
	if err := os.Remove(filepath.Join(repo, "dirty.txt")); err != nil {
		t.Fatal(err)
	}
	result, err := runPlan([]string{"approve", "--id", "sample-plan", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !result.OK {
		t.Fatalf("plan approve failed: result=%#v err=%v", result, err)
	}
	spec := result.Data.(plancheck.Spec)
	if spec.Status != plancheck.StatusApproved || spec.ApprovedTree != wantTree {
		t.Fatalf("unexpected approved spec: %#v", spec)
	}
}
