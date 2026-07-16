package cli

import (
	"os"
	"path/filepath"
	"testing"
)

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
  "externalDependencies": []
}
`)
}
