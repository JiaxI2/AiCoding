package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFreezeChecksCurrentRepository(t *testing.T) {
	repo := filepath.Join("..", "..")
	if err := checkFrozenSchemas(repo); err != nil {
		t.Fatal(err)
	}
	if err := checkUniqueProductionType(repo, "internal/report", "Result"); err != nil {
		t.Fatal(err)
	}
	if err := checkUniqueProductionType(repo, "internal/validationevidence", "Receipt"); err != nil {
		t.Fatal(err)
	}
}

func TestUniqueProductionTypeRejectsDuplicate(t *testing.T) {
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "internal/report/result.go", "package report\n\ntype Result struct{}\n")
	writeFreezeTestFile(t, repo, "internal/report/nested/duplicate.go", "package nested\n\ntype Result struct{}\n")
	err := checkUniqueProductionType(repo, "internal/report", "Result")
	if err == nil || !strings.Contains(err.Error(), "found 2") {
		t.Fatalf("duplicate Result was not rejected: %v", err)
	}
}

func TestRegistryContainsFreezeGates(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]TestCase{}
	for _, testCase := range Registry(cfg) {
		if strings.HasPrefix(testCase.ID, "FREEZE-") {
			found[testCase.ID] = testCase
		}
	}
	for _, id := range []string{"FREEZE-001", "FREEZE-002", "FREEZE-003"} {
		testCase, ok := found[id]
		if !ok || testCase.Kind != "static" || testCase.Severity != Required || len(testCase.Profiles) != len(allProfiles()) {
			t.Fatalf("invalid %s registry case: %#v", id, testCase)
		}
	}
}

func writeFreezeTestFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
