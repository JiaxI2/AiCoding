package plan

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestApproveBindsProvidedTree(t *testing.T) {
	repo := initPlanGitRepo(t)
	writePlanTestFile(t, repo, "docs/spec/sample-plan/PLAN.md", validPlanFrontmatter("sample-plan", StatusDraft, ""))
	gitPlanTest(t, repo, "add", "docs/spec/sample-plan/PLAN.md")
	gitPlanTest(t, repo, "commit", "-m", "test: add plan")
	wantTree := gitPlanTest(t, repo, "rev-parse", "HEAD^{tree}")

	approved, err := Approve(repo, "sample-plan", wantTree)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != StatusApproved || approved.ApprovedTree != wantTree {
		t.Fatalf("Approve() = %#v, want tree %s", approved, wantTree)
	}
	status := gitPlanTest(t, repo, "status", "--short")
	if status != "M docs/spec/sample-plan/PLAN.md" {
		t.Fatalf("approve modified unexpected files: %q", status)
	}
}

func TestEvaluateBindingAndApprovedCoverage(t *testing.T) {
	policy := Policy{SchemaVersion: 1, ExemptPaths: []string{"docs/spec/**"}}
	spec := Spec{
		ID: "sample-plan", Status: StatusApproved, ApprovedTree: strings.Repeat("a", 40),
		Scope: []string{"internal/plan/**", "internal/cli/**"},
	}
	status, err := EvaluateBinding(policy, spec, strings.Repeat("b", 40), []string{
		"README.md", "internal/cli/plan.go", "docs/spec/sample-plan/PLAN.md", "internal/plan/binding.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !status.ScopeCovered || !reflect.DeepEqual(status.Drift, []string{"internal/cli/plan.go", "internal/plan/binding.go"}) ||
		!reflect.DeepEqual(status.Exempt, []string{"docs/spec/sample-plan/PLAN.md"}) || !reflect.DeepEqual(status.OutOfScope, []string{"README.md"}) {
		t.Fatalf("unexpected binding projection: %#v", status)
	}

	sensitive := []SensitiveMatch{{Path: "internal/cli/plan.go"}, {Path: "config/schemas/plan.json"}}
	ids, uncovered, err := ApprovedCoverage([]Spec{spec}, sensitive)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"sample-plan"}) || len(uncovered) != 1 || uncovered[0].Path != "config/schemas/plan.json" {
		t.Fatalf("unexpected coverage: ids=%v uncovered=%#v", ids, uncovered)
	}
}

func initPlanGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitPlanTest(t, repo, "init")
	gitPlanTest(t, repo, "config", "user.email", "plan@example.invalid")
	gitPlanTest(t, repo, "config", "user.name", "Plan Test")
	writePlanTestFile(t, repo, "README.md", "# fixture\n")
	gitPlanTest(t, repo, "add", "README.md")
	gitPlanTest(t, repo, "commit", "-m", "test: initialize")
	return repo
}

func gitPlanTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestDiffTreeFilesIncludesCommittedDeletion(t *testing.T) {
	repo := initPlanGitRepo(t)
	before := gitPlanTest(t, repo, "rev-parse", "HEAD^{tree}")
	writePlanTestFile(t, repo, filepath.Join("src", "value.txt"), "value\n")
	gitPlanTest(t, repo, "add", "src/value.txt")
	gitPlanTest(t, repo, "commit", "-m", "test: add value")
	after := gitPlanTest(t, repo, "rev-parse", "HEAD^{tree}")
	files, err := gitx.DiffTreeFiles(repo, before, after)
	if err != nil || !reflect.DeepEqual(files, []string{"src/value.txt"}) {
		t.Fatalf("tree diff = %v, %v", files, err)
	}
}
