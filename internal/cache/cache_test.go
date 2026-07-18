package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatusReportsCacheWithoutAffectingPassFail(t *testing.T) {
	repo := t.TempDir()
	status, err := Status(repo)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Path != ".aicoding/cache/fast-path" {
		t.Fatalf("Path = %q", status.Path)
	}
	if status.Exists || status.EntryCount != 0 {
		t.Fatalf("unexpected empty cache status: %#v", status)
	}
}

func TestCleanRemovesFastPathCacheOnly(t *testing.T) {
	repo := t.TempDir()
	cacheDir := filepath.Join(repo, ".aicoding", "cache", "fast-path")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "state.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Clean(repo)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if !result.Removed || result.EntryCount != 1 {
		t.Fatalf("unexpected clean result: %#v", result)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("cache dir still exists or unexpected stat error: %v", err)
	}
}
