package governance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintIssueGovernanceRejectsBlankIssues(t *testing.T) {
	repo := t.TempDir()
	writeMinimalGovernanceRepo(t, repo)
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "config.yml"), "blank_issues_enabled: true\n")
	errs := Lint(repo, "commit-msg", filepath.Join(repo, "COMMIT_EDITMSG"))
	if !hasErrorContaining(errs, "disable contributor blank Issues") {
		t.Fatalf("expected blank Issue error, got %#v", errs)
	}
}

func TestLintIssueGovernanceRequiresResolutionLabels(t *testing.T) {
	repo := t.TempDir()
	writeMinimalGovernanceRepo(t, repo)
	path := filepath.Join(repo, ".github", "issue-labels.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, strings.ReplaceAll(string(content), "resolution:invalid", "resolution:unsupported"))
	errs := Lint(repo, "commit-msg", filepath.Join(repo, "COMMIT_EDITMSG"))
	if !hasErrorContaining(errs, "missing required label: resolution:invalid") {
		t.Fatalf("expected resolution label error, got %#v", errs)
	}
}
