package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDependenciesRejectsLowerLayerPlatformReference(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "CodingKit", "tools", "visio-mcp", "server.py"), `SERVICE = "aicoding-visio-mcp"`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "contains upper-layer namespace") {
		t.Fatalf("expected upper-layer namespace error, got %#v", report.Errors)
	}
}

func TestCheckDependenciesRejectsReverseDependency(t *testing.T) {
	repo := dependencyFixture(t)
	path := filepath.Join(repo, dependencyGovernancePath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatal(err)
	}
	mcp := policy["mcpRegistry"].(map[string]interface{})
	binding := mcp["bindings"].([]interface{})[0].(map[string]interface{})
	binding["dependsOn"] = []interface{}{"kit:platform-kit"}
	updated, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, string(updated))

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "must not depend on higher layer") {
		t.Fatalf("expected reverse dependency error, got %#v", report.Errors)
	}
}

func TestCheckDependenciesRejectsStandalonePlatformPrefix(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "config", "codex-kit.json"), `{
  "profiles": {"full": {"standaloneSkills": ["aicoding-visio-diagram"]}},
  "standaloneSkillRegistry": {"skills": ["aicoding-visio-diagram"], "sourcePaths": {}}
}`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "standalone Skill must not use platform prefix") {
		t.Fatalf("expected standalone Skill prefix error, got %#v", report.Errors)
	}
}

func TestCheckDependenciesRejectsCapabilityMCPWorkflowPrompt(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "CodingKit", "tools", "visio-mcp", "prompts", "workflow.md"), "# workflow\n")

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "must not own workflow prompt directory") {
		t.Fatalf("expected MCP prompt ownership error, got %#v", report.Errors)
	}
}

func TestCheckDependenciesAcceptsHigherToLowerBinding(t *testing.T) {
	repo := dependencyFixture(t)

	report := CheckDependencies(repo)

	if len(report.Errors) != 0 {
		t.Fatalf("expected valid dependency policy, got %#v", report.Errors)
	}
}

func TestCheckDependenciesRejectsVersionInCapabilityCode(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "CodingKit", "tools", "visio-mcp", "server.py"), `PID_VERSION_STR = "1.6.0"`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "code observes an asset version") {
		t.Fatalf("expected asset version opacity error, got %#v", report.Errors)
	}
}

func TestCheckDependenciesAllowsExternalProtocolVersion(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "CodingKit", "tools", "visio-mcp", "server.py"), `PROTOCOL_VERSION = "1.2.0"`)

	report := CheckDependencies(repo)

	if len(report.Errors) != 0 {
		t.Fatalf("expected external protocol version authority to remain valid, got %#v", report.Errors)
	}
}

func dependencyFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
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
  "reservedNamespaces": [{"value": "aicoding", "ownerLayer": "platform"}],
  "scan": {
    "extensions": [".py", ".md"],
    "fileNames": [],
    "excludeDirectories": [".venv", "test-results"]
  },
  "versionVisibility": {
    "identityPattern": "(?i)(?:^|[^a-z0-9])(?:v|version|ver)[_-]?[0-9]+(?:[._-][0-9a-z]+)+(?:$|[^a-z0-9])",
    "codeSelfVersionPattern": "(?i)(?:__version__|(?:[a-z][a-z0-9_]*_)?version(?:_(?:major|minor|patch|str|string))?)\\s*(?:=|\\s+)\\s*(?:[\"']?[0-9]+(?:\\.[0-9]+)+[\"']?|[0-9]+[uUlL]?)",
    "codeSelfVersionAllowedSymbols": ["PROTOCOL_VERSION", "MCP_PROTOCOL_VERSION", "SCHEMA_VERSION"],
    "readmeBodyVersionPattern": "\\b[0-9]+\\.[0-9]+(?:\\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?\\b",
    "codeExtensions": [".py"],
    "codeFileNames": [],
    "documentationDirectories": ["docs", "references"],
    "authorityFiles": ["CHANGELOG.md", "MANIFEST.json", "pyproject.toml"],
    "readmeFiles": [],
    "readmeBadges": []
  },
  "kitRegistry": {
    "path": "config/kit-registry.json",
    "bindings": [
      {
        "id": "platform-kit",
        "layer": "platform",
        "platformAgnostic": false,
        "roots": [],
        "dependsOn": ["mcp:visio-mcp"]
      }
    ]
  },
  "mcpRegistry": {
    "path": "config/mcp-registry.json",
    "idPattern": "^[a-z0-9]+(?:-[a-z0-9]+)*-mcp$",
    "promptPolicy": "forbid-workflow-prompts",
    "bindings": [
      {
        "id": "visio-mcp",
        "layer": "capability",
        "platformAgnostic": true,
        "roots": ["CodingKit/tools/visio-mcp"],
        "dependsOn": ["runtime:mcp"]
      }
    ]
  },
  "skills": {
    "runtimeConfig": "config/codex-kit.json",
    "pluginRoot": "plugin/skills",
    "pluginLayer": "integration",
    "pluginRequiredPrefix": "aicoding-",
    "standaloneLayer": "capability",
    "standaloneForbiddenPrefixes": ["aicoding-"]
  },
  "externalDependencies": [{"id": "runtime:mcp", "layer": "runtime"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "schemas", "dependency-governance.schema.json"), "{}")
	mustWrite(t, filepath.Join(repo, "config", "kit-registry.json"), `{
  "kits": [{"id": "platform-kit", "manifest": "config/kits/platform-kit.json"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "kits", "platform-kit.json"), `{"id":"platform-kit"}`)
	mustWrite(t, filepath.Join(repo, "config", "mcp-registry.json"), `{
  "components": [{"id": "visio-mcp", "manifest": "config/mcp/components/visio-mcp.json"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "mcp", "components", "visio-mcp.json"), `{
  "id": "visio-mcp",
  "name": "Visio MCP",
  "description": "Generic Visio component",
  "runtime": {"module": "visio_mcp", "serverArgs": ["-m", "visio_mcp"]},
  "codex": {"serverName": "visio-mcp"}
}`)
	mustWrite(t, filepath.Join(repo, "config", "codex-kit.json"), `{
  "profiles": {"full": {"standaloneSkills": ["visio-diagram"]}},
  "standaloneSkillRegistry": {"skills": ["visio-diagram"], "sourcePaths": {}}
}`)
	mustWrite(t, filepath.Join(repo, "CodingKit", "tools", "visio-mcp", "server.py"), `SERVICE = "visio-mcp"`)
	mustWrite(t, filepath.Join(repo, "plugin", "skills", "aicoding-platform", "SKILL.md"), "---\nname: aicoding-platform\n---\n")
	return repo
}

func hasDependencyErrorContaining(errs []string, needle string) bool {
	for _, err := range errs {
		if strings.Contains(err, needle) {
			return true
		}
	}
	return false
}
