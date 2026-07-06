package kit

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSmokeKitBuiltinCheck(t *testing.T) {
	repo := fixtureRepo(t, "minimal-kit")
	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	res := SmokeKits(repo, entries)
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("unexpected smoke result: %#v", res)
	}
}

func TestSmokeKitMissingRequiredPath(t *testing.T) {
	repo := fixtureRepo(t, "missing-required")
	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	res := SmokeKits(repo, entries)
	if len(res) != 1 || res[0].OK {
		t.Fatalf("expected smoke failure, got %#v", res)
	}
	if !containsError(res[0].Errors, "missing required path: README.md") {
		t.Fatalf("expected missing required path error, got %#v", res[0].Errors)
	}
}

func fixtureRepo(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "repos", name)
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func containsError(errs []string, needle string) bool {
	for _, err := range errs {
		if strings.Contains(err, needle) {
			return true
		}
	}
	return false
}
