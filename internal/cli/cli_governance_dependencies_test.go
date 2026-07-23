package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunGovernanceDependenciesReturnsStructuredFailure(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/dependency-fixture\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "config", "dependency-governance.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := runGovernance([]string{"dependencies", "--repo-root", repo}, time.Now())

	if err == nil || result.OK {
		t.Fatalf("expected dependency governance failure, got result=%#v err=%v", result, err)
	}
	if result.Command != "governance dependencies" {
		t.Fatalf("unexpected command: %s", result.Command)
	}
}
