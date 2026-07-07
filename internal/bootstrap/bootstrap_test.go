package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckReportsRepoPrerequisites(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/repo\n\ngo 1.22\n")
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	status, errs := Check(repo)
	if len(errs) != 0 {
		t.Fatalf("Check errs = %v", errs)
	}
	if status.RepoRoot != repo {
		t.Fatalf("RepoRoot = %q, want %q", status.RepoRoot, repo)
	}
	if !status.GoMod || !status.GitDir || status.BinDirExists {
		t.Fatalf("unexpected status: %#v", status)
	}
	if status.BinaryPath != "bin/aicoding.exe" {
		t.Fatalf("BinaryPath = %q", status.BinaryPath)
	}
}

func TestBootstrapWithoutBuildCreatesBinDir(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/repo\n\ngo 1.22\n")
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	status, errs := Bootstrap(repo, Options{Build: false})
	if len(errs) != 0 {
		t.Fatalf("Bootstrap errs = %v", errs)
	}
	if !status.BinDirExists {
		t.Fatalf("expected bin directory after bootstrap: %#v", status)
	}
	if status.BuildAttempted {
		t.Fatalf("BuildAttempted = true for Build=false")
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
