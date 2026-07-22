package governance

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
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

func TestReadmeBadgeLabelsRejectLowercaseInitial(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "README.md"), "[![govulncheck](https://img.shields.io/badge/Govulncheck-1.6.0-gray)](https://example.com)\n")
	policy := dependencyPolicy{VersionVisibility: versionVisibility{
		ReadmeBodyVersionPattern: `\b[0-9]+\.[0-9]+(?:\.[0-9]+)?\b`,
		ReadmeFiles:              []string{"README.md"},
	}}

	errs := checkReadmeVersionBadges(repo, policy)

	if !hasErrorContaining(errs, "README.md badge label must start with an uppercase ASCII letter: govulncheck") {
		t.Fatalf("expected lowercase badge label error, got %#v", errs)
	}
}

func TestReadmeBadgeLabelsAcceptUppercaseInitial(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "README.md"), "[![Govulncheck](https://img.shields.io/badge/Govulncheck-1.6.0-gray)](https://example.com)\n")
	policy := dependencyPolicy{VersionVisibility: versionVisibility{
		ReadmeBodyVersionPattern: `\b[0-9]+\.[0-9]+(?:\.[0-9]+)?\b`,
		ReadmeFiles:              []string{"README.md"},
	}}

	if errs := checkReadmeVersionBadges(repo, policy); len(errs) != 0 {
		t.Fatalf("expected uppercase badge label to pass, got %#v", errs)
	}
}

func TestReadmeImagesAcceptSVGThemeMarkersAndSemanticBadges(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "docs", "assets", "overview-light.svg"), "<svg></svg>\n")
	mustWrite(t, filepath.Join(repo, "docs", "assets", "overview-dark.svg"), "<svg></svg>\n")
	mustWrite(t, filepath.Join(repo, "README.md"), `<p>
  <img src="docs/assets/overview-light.svg#gh-light-mode-only">
  <img src="docs/assets/overview-dark.svg#gh-dark-mode-only">
</p>
![Architecture](docs/assets/overview-light.svg)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![CI](https://img.shields.io/github/actions/workflow/status/acme/repo/ci.yml?branch=main)](https://example.com/ci)
`)

	if errs := checkReadmeImages(repo, []string{"README.md"}); len(errs) != 0 {
		t.Fatalf("expected governed SVG sources to pass, got %#v", errs)
	}
}

func TestReadmeImagesRejectRasterMermaidAndImplicitBadgeColor(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "README.md"), "```mermaid\ngraph LR\n```\n![Raster](diagram.png)\n[![Tool](https://img.shields.io/badge/Tool-stable)](https://example.com)\n")

	errs := checkReadmeImages(repo, []string{"README.md"})
	for _, want := range []string{
		"must embed exported SVG instead of Mermaid",
		"must use an SVG source: diagram.png",
		"uses an implicit/default badge color",
	} {
		if !hasErrorContaining(errs, want) {
			t.Fatalf("expected %q, got %#v", want, errs)
		}
	}
}

func TestReadmeImagesRejectMismatchedThemeMarker(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "docs", "assets", "overview-dark.svg"), "<svg></svg>\n")
	mustWrite(t, filepath.Join(repo, "README.md"), `<img src="docs/assets/overview-dark.svg#gh-light-mode-only">`)

	errs := checkReadmeImages(repo, []string{"README.md"})
	if !hasErrorContaining(errs, "must bind #gh-light-mode-only to a -light.svg asset") {
		t.Fatalf("expected mismatched theme marker error, got %#v", errs)
	}
}

func TestGoPackageBoundariesRejectReverseCoreDependency(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "internal", "registry", "bad.go"), `package registry
import _ "github.com/JiaxI2/AiCoding/internal/lifecycle"
`)
	boundaries := []goPackageBoundary{{
		Path:             "internal/registry",
		ForbiddenImports: []string{"internal/lifecycle"},
	}}
	errs := checkGoPackageBoundariesWithInventory(repo, boundaries, dependencyInventoryForTest(t, repo, "internal/registry"))
	if !hasErrorContaining(errs, "imports forbidden package") {
		t.Fatalf("expected orthogonal package boundary error, got %#v", errs)
	}
}

func TestGoPackageBoundariesAllowLowerUtilityDependency(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "internal", "kit", "good.go"), `package kit
import _ "github.com/JiaxI2/AiCoding/internal/registry"
`)
	boundaries := []goPackageBoundary{{
		Path:             "internal/kit",
		ForbiddenImports: []string{"internal/lifecycle", "internal/mcpcontrol"},
	}}
	errs := checkGoPackageBoundariesWithInventory(repo, boundaries, dependencyInventoryForTest(t, repo, "internal/kit"))
	if len(errs) != 0 {
		t.Fatalf("valid lower utility dependency was rejected: %#v", errs)
	}
}

func TestGitProcessOwnershipRejectsGitOutsideGitx(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "internal", "report", "bad.go"), `package report
import "os/exec"
func bad() { _ = exec.Command("git", "version") }
`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "git process ownership: internal/report/bad.go starts git process outside internal/gitx") {
		t.Fatalf("expected git process ownership error, got %#v", report.Errors)
	}
}

func TestGitProcessOwnershipAllowsOwnerPackage(t *testing.T) {
	report := CheckDependencies(dependencyFixture(t))

	check, ok := dependencyCheckByName(report.Checks, "git process ownership")
	if !ok || !check.OK {
		t.Fatalf("expected git process owner package to pass, got %#v", check)
	}
}

func TestGitxImporterAllowlistRejectsUnknownImporter(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "internal", "runner", "bad.go"), `package runner
import _ "github.com/JiaxI2/AiCoding/internal/gitx"
`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, "gitx importer allowlist: internal/runner/bad.go imports internal/gitx from non-allowlisted package internal/runner") {
		t.Fatalf("expected gitx importer allowlist error, got %#v", report.Errors)
	}
}

func TestGitxImporterAllowlistAllowsRegisteredImporter(t *testing.T) {
	report := CheckDependencies(dependencyFixture(t))

	check, ok := dependencyCheckByName(report.Checks, "gitx importer allowlist")
	if !ok || !check.OK {
		t.Fatalf("expected registered gitx importer to pass, got %#v", check)
	}
}

func TestActivationManifestsURLFreeAllowsLocalValues(t *testing.T) {
	report := CheckDependencies(dependencyFixture(t))

	check, ok := dependencyCheckByName(report.Checks, "activation manifests URL-free")
	if !ok || !check.OK {
		t.Fatalf("expected local activation manifest values to pass, got %#v", check)
	}
}

func TestActivationManifestsURLFreeRejectsNestedURL(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "config", "kits", "platform-kit.json"), `{
  "id": "platform-kit",
  "runtime": {"endpoint": "https://example.com/package"}
}`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, `activation manifests URL-free: config/kits/platform-kit.json $["runtime"]["endpoint"] contains URL`) {
		t.Fatalf("expected activation URL error with JSON path, got %#v", report.Errors)
	}
}

func TestCloneableSourcesRegistryAllowsDeclaredRegistries(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, ".gitmodules"), `[submodule "dependency"]
	path = dependency
	url = https://github.com/example/dependency.git
`)
	mustWrite(t, filepath.Join(repo, "config", "skill-sources.json"), `{
  "sources": [{"url": "https://github.com/example/skill.git"}]
}`)

	report := CheckDependencies(repo)

	check, ok := dependencyCheckByName(report.Checks, "cloneable sources registry")
	if !ok || !check.OK {
		t.Fatalf("expected declared acquisition registries to pass, got %#v", check)
	}
}

func TestCloneableSourcesRegistryRejectsUndeclaredFile(t *testing.T) {
	repo := dependencyFixture(t)
	mustWrite(t, filepath.Join(repo, "config", "rogue-source.json"), `{
  "source": "https://github.com/example/rogue.git"
}`)

	report := CheckDependencies(repo)

	if !hasErrorContaining(report.Errors, `cloneable sources registry: config/rogue-source.json $["source"] contains cloneable source outside acquisition registry`) {
		t.Fatalf("expected cloneable source registry error with JSON path, got %#v", report.Errors)
	}
}

func TestAcquisitionBoundaryRejectsMissingPolicy(t *testing.T) {
	err := checkActivationManifestsURLFree(acquisitionBoundary{}, nil)
	if !hasErrorContaining(err, "acquisitionBoundary policy is missing or incomplete") {
		t.Fatalf("expected missing acquisition policy error, got %#v", err)
	}
}

func TestCheckDependenciesWalksRepositoryOnce(t *testing.T) {
	repo := dependencyFixture(t)
	walks := 0
	report := checkDependencies(repo, func(root string, walkFn fs.WalkDirFunc) error {
		walks++
		return filepath.WalkDir(root, walkFn)
	})
	if len(report.Errors) != 0 {
		t.Fatalf("dependency fixture failed: %#v", report.Errors)
	}
	if walks != 1 {
		t.Fatalf("dependency inventory WalkDir calls = %d, want 1", walks)
	}
	t.Logf("dependency inventory WalkDir calls=%d", walks)
}

func TestDependencyInventoryAlwaysIncludesConfigJSON(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "config", "rogue.json"), "{}\n")
	inventory, err := buildDependencyInventory(repo, dependencyPolicy{}, filepath.WalkDir)
	if err != nil {
		t.Fatal(err)
	}
	if !inventory.hasFile("config/rogue.json") {
		t.Fatalf("config JSON was absent from the mandatory dependency inventory: %#v", inventory.files)
	}
}

func dependencyInventoryForTest(t *testing.T, repo string, roots ...string) *dependencyInventory {
	t.Helper()
	inventory, err := collectDependencyInventory(repo, roots, filepath.WalkDir)
	if err != nil {
		t.Fatal(err)
	}
	return inventory
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
  "externalDependencies": [{"id": "runtime:mcp", "layer": "runtime"}],
  "acquisitionBoundary": {
    "activationUrlFreeFiles": [
      "config/kit-registry.json",
      "config/kits",
      "config/mcp-registry.json",
      "config/mcp/components",
      "config/codex-kit.json"
    ],
    "cloneableSourcePattern": "(?i)^(((https?|ssh|git)://[^\\s]+\\.git)|(git@[^\\s:]+:[^\\s]+\\.git)|(https?://(www\\.)?(github\\.com|gitcode\\.[a-z]+)/[^/]+/[^/]+/?))$",
    "acquisitionRegistryFiles": [".gitmodules", "config/skill-sources.json"],
    "scanRoots": ["config"]
  },
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
	mustWrite(t, filepath.Join(repo, "cmd", "aicoding", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(repo, "internal", "gitx", "git.go"), `package gitx
import "os/exec"
func run() { _ = exec.Command("git", "version") }
`)
	mustWrite(t, filepath.Join(repo, "internal", "cli", "cli.go"), `package cli
import _ "github.com/JiaxI2/AiCoding/internal/gitx"
`)
	return repo
}

func dependencyCheckByName(checks []DependencyCheck, name string) (DependencyCheck, bool) {
	for _, check := range checks {
		if check.Name == name {
			return check, true
		}
	}
	return DependencyCheck{}, false
}
