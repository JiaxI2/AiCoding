package platform

import (
	"os"
	"os/exec"
	"testing"
)

func TestResolveRepoRootUsesCurrentRootFastPath(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	resolved, err := ResolveRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != repo {
		t.Fatalf("ResolveRepoRoot() = %q, want %q", resolved, repo)
	}
}
