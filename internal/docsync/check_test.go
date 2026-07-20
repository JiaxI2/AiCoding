package docsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckModesExposeFileClasses(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	writeDocSyncTestFile(t, repo, "internal/docsync/docsync.go", "package docsync\n")
	writeDocSyncTestFile(t, repo, "internal/docsync/check.go", "package docsync\n")
	writeDocSyncTestFile(t, repo, "config/docs-sync.policy.json", "{}\n")
	writeDocSyncTestFile(t, repo, "config/docs-sync.semantic.json", "{}\n")
	writeDocSyncTestFile(t, repo, ".github/workflows/aicoding-ci.yml", "name: docs\n")
	writeDocSyncTestFile(t, repo, "docs/COMMANDS.md", "# Commands\n")
	writeDocSyncTestFile(t, repo, "docs/architecture/DOC_SYNC_PLUS_SPEC.md", "# Spec\n\nStatus: Accepted and Frozen\n")
	writeDocSyncTestFile(t, repo, "docs/operations/DOC_SYNC_PLUS_VALIDATION_PLAN.md", "# Plan\n")
	if out, err := exec.Command("git", "-C", repo, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	for _, mode := range []string{"staged", "all", "ci", "release"} {
		t.Run(mode, func(t *testing.T) {
			res := Check(repo, mode)
			if !res.OK {
				t.Fatalf("expected %s to pass: %#v", mode, res.Errors)
			}
			if res.Mode != mode || len(res.Checked) == 0 {
				t.Fatalf("unexpected mode/check payload: %#v", res)
			}
			if !containsDocSyncTestValue(res.RiskFiles, "internal/docsync/check.go") {
				t.Fatalf("riskFiles missing internal docsync file: %#v", res.RiskFiles)
			}
			if !containsDocSyncTestValue(res.DocFiles, "docs/COMMANDS.md") {
				t.Fatalf("docFiles missing docs file: %#v", res.DocFiles)
			}
			if mode != "staged" && (len(res.Checks) != 1 || res.Checks[0].Name != "architecture status headers" || !res.Checks[0].OK) {
				t.Fatalf("architecture status check missing for %s: %#v", mode, res.Checks)
			}
		})
	}

	bad := Check(repo, "unknown")
	if bad.OK || len(bad.Errors) == 0 {
		t.Fatalf("unsupported mode should fail with errors: %#v", bad)
	}
}

func TestArchitectureStatusGateFailsClosedOutsideStagedMode(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	writeDocSyncTestFile(t, repo, "docs/architecture/MISSING.md", "# Missing\n")
	writeDocSyncTestFile(t, repo, "docs/architecture/README.md", "# Index\n")

	for _, mode := range []string{"all", "ci", "release"} {
		result := Check(repo, mode)
		if result.OK || len(result.Checks) != 1 || result.Checks[0].OK || !containsDocSyncSubstring(result.Errors, "MISSING.md is missing a Status header") {
			t.Fatalf("%s did not fail closed: %#v", mode, result)
		}
	}
}

func writeDocSyncTestFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsDocSyncTestValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsDocSyncSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
