package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckLayoutRejectsForbiddenRoot(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, repositoryLayoutPath), `{
  "schemaVersion": 1,
  "root": {"allowDirectories": ["config"], "forbiddenDirectories": ["scripts"], "transientDirectories": []},
  "directoryClasses": {},
  "documentation": {"root": "docs", "allowedRootFiles": [], "allowedOutsideRoots": []},
  "prompts": {"allowedRoots": [], "forbiddenRoot": "prompts"},
  "testFixtures": {"root": "testdata", "forbiddenRoot": "tests"},
  "generatedArtifacts": {"directories": [], "extensions": []},
  "skills": {"authoritativeRoots": [], "excludedRoots": [], "runtimeMirrors": []}
}`)
	if err := os.Mkdir(filepath.Join(repo, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	report := CheckLayout(repo)
	if !hasErrorContaining(report.Errors, "forbidden root directory exists: scripts") {
		t.Fatalf("expected forbidden root error, got %#v", report.Errors)
	}
}

func TestCheckLayoutRejectsDuplicateSkillSources(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, repositoryLayoutPath), `{
  "schemaVersion": 1,
  "root": {"allowDirectories": ["config", "skills"], "forbiddenDirectories": [], "transientDirectories": []},
  "directoryClasses": {},
  "documentation": {"root": "docs", "allowedRootFiles": [], "allowedOutsideRoots": ["skills"]},
  "prompts": {"allowedRoots": [], "forbiddenRoot": "prompts"},
  "testFixtures": {"root": "testdata", "forbiddenRoot": "tests"},
  "generatedArtifacts": {"directories": [], "extensions": []},
  "skills": {"authoritativeRoots": ["skills"], "excludedRoots": [], "runtimeMirrors": []}
}`)
	mustWrite(t, filepath.Join(repo, "skills", "first", "SKILL.md"), "---\nname: same-skill\n---\n")
	mustWrite(t, filepath.Join(repo, "skills", "second", "SKILL.md"), "---\nname: same-skill\n---\n")
	report := CheckLayout(repo)
	if !hasErrorContaining(report.Errors, "skill has multiple source-of-truth paths: same-skill") {
		t.Fatalf("expected duplicate skill source error, got %#v", report.Errors)
	}
}

func TestTaskRuntimeDirectoryIsDeclaredByLayoutAndNavigation(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	var layout struct {
		Root struct {
			AllowDirectories []string `json:"allowDirectories"`
		} `json:"root"`
	}
	readJSONFile(t, filepath.Join(repo, "config", "repository-layout.json"), &layout)

	var navigation struct {
		Root struct {
			AllowedDirectories []string `json:"allowedDirectories"`
		} `json:"root"`
	}
	readJSONFile(t, filepath.Join(repo, "config", "repository-navigation.json"), &navigation)

	if !contains(layout.Root.AllowDirectories, ".task") || !contains(navigation.Root.AllowedDirectories, ".task") {
		t.Fatalf(".task must be declared by layout and navigation: layout=%v navigation=%v", layout.Root.AllowDirectories, navigation.Root.AllowedDirectories)
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
