package cli

import (
	"path/filepath"
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
	gitWorkTest(t, repo, "add", plancheck.PolicyPath)
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

	gitWorkTest(t, repo, "reset")
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
	specs, ok := status.Data.([]plancheck.Spec)
	if !ok || len(specs) != 1 || specs[0].ID != "sample-plan" || specs[0].Status != plancheck.StatusArchived {
		t.Fatalf("unexpected status data: %#v", status.Data)
	}
	selected, err := runPlan([]string{"status", "--id", "sample-plan", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !selected.OK || len(selected.Data.([]plancheck.Spec)) != 1 {
		t.Fatalf("plan status --id failed: result=%#v err=%v", selected, err)
	}
}
