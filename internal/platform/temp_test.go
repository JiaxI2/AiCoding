package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestTempDirectoryLifecycleIsAppendOnlyAndBounded(t *testing.T) {
	repo := newPlatformGitRepo(t)
	path, err := CreateTempDir(repo, "fresh-clone")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	if !sameFilesystemPath(filepath.Dir(path), os.TempDir()) || !strings.HasPrefix(filepath.Base(path), TempDirectoryPrefix+"fresh-clone-") {
		t.Fatalf("unexpected temp path: %s", path)
	}
	if err := os.WriteFile(filepath.Join(path, "evidence.txt"), []byte("failed evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordTempOutcome(repo, path, "fresh-clone", "failed"); err != nil {
		t.Fatal(err)
	}
	records, err := ReadTempLedger(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].Outcome != "created" || records[1].Outcome != "failed" || records[1].SizeBytes == 0 {
		t.Fatalf("unexpected ledger before release: %#v", records)
	}
	if err := ReleaseTempDir(repo, path, "fresh-clone"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("released temp still exists: %v", err)
	}
	records, err = ReadTempLedger(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 || records[2].Outcome != "releasing" || records[3].Outcome != "released" {
		t.Fatalf("release events missing: %#v", records)
	}
}

func TestReleaseTempDirRejectsNonAiCodingAndNestedPaths(t *testing.T) {
	repo := newPlatformGitRepo(t)
	unrelated, err := os.MkdirTemp("", "unrelated-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(unrelated) })
	if err := ReleaseTempDir(repo, unrelated, "fresh-clone"); err == nil || !strings.Contains(err.Error(), "refuse to release") {
		t.Fatalf("non-aicoding path was accepted: %v", err)
	}

	parent, err := os.MkdirTemp("", TempDirectoryPrefix+"parent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	nested := filepath.Join(parent, TempDirectoryPrefix+"nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ReleaseTempDir(repo, nested, "fresh-clone"); err == nil || !strings.Contains(err.Error(), "refuse to release") {
		t.Fatalf("nested path was accepted: %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("negative fixture was touched: %v", err)
	}
}

func newPlatformGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if _, err := gitx.Run(repo, "init", "-q"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	return repo
}
