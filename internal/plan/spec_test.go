package plan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifySpecsAcceptsValidSpecsAndListIsDeterministic(t *testing.T) {
	repo := t.TempDir()
	writePlanTestFile(t, repo, "docs/spec/z-plan/PLAN.md", validPlanFrontmatter("z-plan", StatusArchived, ""))
	writePlanTestFile(t, repo, "docs/spec/a-plan/PLAN.md", validBoundPlanFrontmatter("a-plan", StatusApproved, strings.Repeat("a", 40)))

	verification, err := VerifySpecs(repo)
	if err != nil || !verification.OK || len(verification.Specs) != 2 {
		t.Fatalf("VerifySpecs() = %#v, %v", verification, err)
	}
	if len(verification.Warnings) != 0 {
		t.Fatalf("unexpected verification warnings: %#v", verification.Warnings)
	}
	first, err := ListSpecs(repo)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ListSpecs(repo)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) || first[0].ID != "a-plan" || first[1].ID != "z-plan" {
		t.Fatalf("ListSpecs is not deterministic: %s / %s", firstJSON, secondJSON)
	}
}

func TestVerifySpecsReportsEveryBadFixture(t *testing.T) {
	repo := t.TempDir()
	fixtureRoot := filepath.Join("..", "..", "testdata", "plan")
	entries, err := os.ReadDir(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		source := filepath.Join(fixtureRoot, entry.Name())
		target := filepath.Join(repo, "docs", "spec", entry.Name())
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		files, err := os.ReadDir(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			content, err := os.ReadFile(filepath.Join(source, file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, file.Name()), content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	verification, err := VerifySpecs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if verification.OK || len(verification.Errors) != 4 {
		t.Fatalf("bad fixtures were not reported individually: %#v", verification.Errors)
	}
	for _, fixture := range []string{"id-mismatch", "invalid-status", "missing-decision", "missing-frontmatter"} {
		found := false
		for _, message := range verification.Errors {
			if strings.Contains(message, "/"+fixture+"/PLAN.md") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("fixture %s has no error: %#v", fixture, verification.Errors)
		}
	}
}

func writePlanTestFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validPlanFrontmatter(id, status, decision string) string {
	return validBoundPlanFrontmatter(id, status, "")
}

func validBoundPlanFrontmatter(id, status, approvedTree string) string {
	return "---\n" +
		"id: " + id + "\n" +
		"status: " + status + "\n" +
		"scope:\n  - internal/plan/**\n" +
		"approvedTree: \"" + approvedTree + "\"\n" +
		"decision: \"\"\n" +
		"gates:\n  - profile: full\n" +
		"---\n\n# Test plan\n"
}
