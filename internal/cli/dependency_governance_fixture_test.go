package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceDependenciesReportsGitProcessBoundaryChecks(t *testing.T) {
	repo := t.TempDir()
	writeGoControlFixture(t, repo)

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"governance", "dependencies", "--repo-root", repo, "--json"}, &stdout, &stderr)
	if code != ExitSuccess {
		t.Fatalf("governance dependencies failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, check := range []string{"git process ownership", "gitx importer allowlist"} {
		if !strings.Contains(stdout.String(), `"name": "`+check+`"`) {
			t.Fatalf("governance dependencies JSON is missing %q: %s", check, stdout.String())
		}
	}
}

func writeDependencyGovernanceFixture(t *testing.T, repo string) {
	t.Helper()
	governancePath := filepath.Join(repo, ".github", "repository-governance.toml")
	governance, err := os.ReadFile(governancePath)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, governancePath, string(governance)+`
[architecture]
principle = "lower-must-not-depend-on-or-observe-upper"
dependency_policy = "config/dependency-governance.json"
dependency_validator = "bin/aicoding.exe governance dependencies --json"
readme_version_surface = "badges-only"
version_badge_policy = "config/dependency-governance.json#versionVisibility.readmeBadges"
`)
	mustWrite(t, filepath.Join(repo, "config", "schemas", "dependency-governance.schema.json"), "{}\n")
	mustWrite(t, filepath.Join(repo, "config", "dependency-mcp-registry.json"), `{"components":[]}`+"\n")
	mustWrite(t, filepath.Join(repo, "config", "dependency-governance.json"), `{
  "schemaVersion": 1,
  "name": "fixture",
  "direction": "higher-rank-may-depend-on-equal-or-lower-rank",
  "layers": [
    {"id": "platform", "rank": 400, "description": "platform"},
    {"id": "integration", "rank": 300, "description": "integration"},
    {"id": "capability", "rank": 200, "description": "capability"},
    {"id": "runtime", "rank": 100, "description": "runtime"}
  ],
  "reservedNamespaces": [{"value": "aicoding-", "ownerLayer": "platform"}],
  "scan": {"extensions":[".go"],"fileNames":[],"excludeDirectories":[".git"]},
  "versionVisibility": {
    "identityPattern": "(?i)(?:^|[^a-z0-9])(?:v|version|ver)[_-]?[0-9]+(?:[._-][0-9a-z]+)+(?:$|[^a-z0-9])",
    "codeSelfVersionPattern": "(?i)(?:__version__|(?:asset|kit|component|service|package)_?version)\\s*=\\s*[\"'][0-9]+(?:\\.[0-9]+)+",
    "readmeBodyVersionPattern": "\\b[0-9]+\\.[0-9]+(?:\\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?\\b",
    "codeExtensions": [".go"],
    "codeFileNames": [],
    "documentationDirectories": ["docs", "references"],
    "authorityFiles": ["CHANGELOG.md", "MANIFEST.json", "pyproject.toml"],
    "readmeFiles": [],
    "readmeBadges": []
  },
  "kitRegistry": {
    "path": "config/kit-registry.json",
    "bindings": [
      {"id":"sample-kit","layer":"platform","platformAgnostic":false,"roots":[],"dependsOn":[]}
    ]
  },
  "mcpRegistry": {
    "path": "config/dependency-mcp-registry.json",
    "idPattern": "^[a-z0-9]+(?:-[a-z0-9]+)*-mcp$",
    "promptPolicy": "forbid-workflow-prompts",
    "bindings": []
  },
  "skills": {
    "runtimeConfig": "config/codex-kit.json",
    "pluginRoot": "CodingKit/agents/skills/plugins/AiCoding/skills",
    "pluginLayer": "integration",
    "pluginRequiredPrefix": "aicoding-",
    "standaloneLayer": "capability",
    "standaloneForbiddenPrefixes": ["aicoding-"]
  },
  "externalDependencies": [],
  "gitProcessBoundary": {
    "ownerPackage": "internal/gitx",
    "scanRoots": ["cmd", "internal"],
    "allowedImporters": ["internal/cli"]
  },
  "goPackageBoundaries": [
    {
      "path": "internal/gitx",
      "forbiddenImports": ["internal/platform"]
    }
  ]
}
`)
	mustWrite(t, filepath.Join(repo, "cmd", "aicoding", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(repo, "internal", "gitx", "git.go"), `package gitx
import "os/exec"
func run() { _ = exec.Command("git", "version") }
`)
	mustWrite(t, filepath.Join(repo, "internal", "cli", "cli.go"), `package cli
import _ "github.com/JiaxI2/AiCoding/internal/gitx"
`)
}
