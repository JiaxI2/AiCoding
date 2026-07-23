package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRaceScopeRejectsUnregisteredConcurrentPackage(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeRaceScopeFixture(t, repo, []string{"internal/runner"})
	writeRaceTestFile(t, filepath.Join(repo, "internal", "runner", "runner.go"), "package runner\n")
	writeRaceTestFile(t, filepath.Join(repo, "internal", "unregistered", "probe.go"), "package unregistered\nfunc probe() { go func() {}() }\n")

	err := checkRaceScope(repo)
	if err == nil || !strings.Contains(err.Error(), "internal/unregistered") {
		t.Fatalf("GO-007 did not reject the unregistered concurrent package: %v", err)
	}
	t.Log(err)
}

func TestRaceCommandScopesFullAndRelease(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeRaceScopeFixture(t, repo, []string{"internal/runner", "internal/testengine"})

	full := raceTestCommand(Config{Repo: repo, Profile: ProfileFull})
	release := raceTestCommand(Config{Repo: repo, Profile: ProfileRelease})
	if got := strings.Join(full, " "); got != "go test -race ./internal/runner ./internal/testengine" {
		t.Fatalf("Full GO-002 command = %q", got)
	}
	if got := strings.Join(release, " "); got != "go test -race ./internal/runner ./internal/testengine" {
		t.Fatalf("Release GO-002 command = %q", got)
	}
	t.Logf("Full GO-002: %s", strings.Join(full, " "))
	t.Logf("Release GO-002: %s", strings.Join(release, " "))
}

func TestRepositoryRaceScopeCoversConcurrentPackages(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRaceScope(repo); err != nil {
		t.Fatal(err)
	}
}

func writeRaceScopeFixture(t *testing.T, repo string, packages []string) {
	t.Helper()
	var lines []string
	for _, packageDir := range packages {
		lines = append(lines, `      "`+packageDir+`"`)
	}
	content := "{\n  \"schemaVersion\": 1,\n  \"raceScope\": {\n    \"packages\": [\n" + strings.Join(lines, ",\n") + "\n    ],\n    \"reason\": \"test fixture\"\n  }\n}\n"
	writeRaceTestFile(t, filepath.Join(repo, filepath.FromSlash(impactPolicyPath)), content)
	for _, packageDir := range packages {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(packageDir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeRaceTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
