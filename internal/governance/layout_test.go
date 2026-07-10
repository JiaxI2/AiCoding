package governance

import (
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
