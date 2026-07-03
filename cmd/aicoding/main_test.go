package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstCommitSubject(t *testing.T) {
	got := firstCommitSubject("\n# comment\nfeat(core): add fast path\nbody")
	if got != "feat(core): add fast path" {
		t.Fatalf("unexpected subject: %q", got)
	}
}

func TestDocPathClassifiers(t *testing.T) {
	if !isDocPath("docs/FAST.md") || !isDocPath("README.md") {
		t.Fatalf("doc path classifier rejected known docs")
	}
	if !isDocSyncRiskPath("scripts/test.ps1") || !isDocSyncRiskPath(".github/workflows/fast-path.yml") {
		t.Fatalf("risk path classifier rejected known risk paths")
	}
	if isDocSyncRiskPath("docs/FAST.md") {
		t.Fatalf("doc path should not be treated as risk source path")
	}
}

func TestSmokeKitBuiltinCheck(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "config", "kit-registry.json"), `{
  "schemaVersion": 1,
  "name": "test",
  "defaultMode": "repo-scoped",
  "kits": [
    {"id":"sample-kit","enabled":true,"order":10,"manifest":"config/kits/sample-kit.json"}
  ]
}`)
	mustWrite(t, filepath.Join(repo, "config", "kits", "sample-kit.json"), `{
  "schemaVersion": 1,
  "id": "sample-kit",
  "name": "Sample Kit",
  "version": "0.1.0",
  "kind": ["test"],
  "mode": "declarative",
  "commands": {
    "verify": {"type":"builtin-check", "requiredPaths":["README.md"]}
  }
}`)
	mustWrite(t, filepath.Join(repo, "README.md"), "# sample\n")
	entries, err := loadRegistry(repo)
	if err != nil {
		t.Fatalf("loadRegistry: %v", err)
	}
	res := smokeKits(repo, entries)
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("unexpected smoke result: %#v", res)
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
