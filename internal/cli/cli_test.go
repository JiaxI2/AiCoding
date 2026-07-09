package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunNewFastPathCommands(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/repo\n\ngo 1.22\n")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "tasks:\n  smoke:\n    cmds:\n      - bin/aicoding.exe kit verify --all --profile Smoke --json\n")
	mustWrite(t, filepath.Join(repo, "config", "tagging-policy.json"), `{"schemaVersion":1}`)
	writeReleaseFixture(t, repo)

	start := time.Now()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"bootstrap", func() error {
			res, err := runBootstrap([]string{"--repo-root", repo, "--no-build", "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"cache status", func() error {
			res, err := runCache([]string{"status", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"doctor pwsh-budget", func() error {
			res, err := runDoctor([]string{"pwsh-budget", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"tag audit", func() error {
			res, err := runTag([]string{"audit", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"release verify", func() error {
			res, err := runRelease([]string{"verify", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func resultErr(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return os.ErrInvalid
	}
	return nil
}

func writeReleaseFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "CHANGELOG.md"), "# CHANGELOG\n\n## [Unreleased]\n\n- **docs**: test fixture.\n")
	mustWrite(t, filepath.Join(repo, ".github", "RELEASE_TEMPLATE.md"), "## 摘要 / Summary\n\n## 变更内容 / What's Changed\n\n## 可追溯性 / Traceability\n")
	mustWrite(t, filepath.Join(repo, "docs", "TAGGING_POLICY.md"), "vMAJOR.MINOR.PATCH\nkit/<kit-id>/vMAJOR.MINOR.PATCH\nmilestone/YYYY.MM.DD-<name>\n")
	mustWrite(t, filepath.Join(repo, "docs", "RELEASE_POLICY.md"), "Platform Release\nKit / Component Release\nMilestone Release\n")
	for _, rel := range []string{
		"docs/RELEASE_GOVERNANCE_OVERLAY.md",
		"scripts/aicoding-tag-governance.ps1",
		"scripts/verify-release-governance-overlay.ps1",
		"config/kits/release-governance-overlay-kit.json",
		".aicoding/templates/perf-cache-plan.json",
	} {
		mustWrite(t, filepath.Join(repo, filepath.FromSlash(rel)), "ok\n")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMainSwitchRoutesNewCommands(t *testing.T) {
	repo := t.TempDir()
	cmd := exec.Command("go", "run", "../../cmd/aicoding", "cache", "status", "--repo-root", repo, "--json")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run cache status: %v: %s", err, out)
	}
	if !strings.Contains(string(out), `"command": "cache status"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestMainSwitchWiresGoFirstTopLevelCommands(t *testing.T) {
	b, err := os.ReadFile("cli.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(b)
	for _, needle := range []string{
		`case "smoke":`,
		`res, err = runSmoke`,
		`case "ci":`,
		`res, err = runCI`,
		`case "docsync":`,
		`res, err = runDocSync`,
		`case "skill":`,
		`res, err = runSkill`,
		`case "lifecycle":`,
		`res, err = runLifecycle`,
		`case "export":`,
		`res, err = runExport`,
		`case "fresh-clone":`,
		`res, err = runFreshClone`,
		`case "full":`,
		`res, err = runFull`,
		`case "release":`,
		`res, err = runReleaseCommand`,
		`aicoding release gate`,
		`aicoding skill c99-standard-c status`,
	} {
		if !strings.Contains(source, needle) {
			t.Fatalf("cli.go is missing %q", needle)
		}
	}
	outdated := "Full/Release gates remain" + " in PowerShell/Python"
	if strings.Contains(source, outdated) {
		t.Fatal("usage still describes Full/Release as PowerShell/Python gates")
	}
	for _, forbidden := range []string{`case "workflow":`, `case "cstyle":`} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("cli.go still exposes removed entry %q", forbidden)
		}
	}
}

func TestGoControlPlaneCommandsUseRealGoImplementations(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	writeGoControlFixture(t, repo)
	if out, err := exec.Command("git", "-C", repo, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	start := time.Now()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"docsync staged", func() error {
			res, err := runDocSync([]string{"staged", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "docsync staged", err)
		}},
		{"docsync all", func() error {
			res, err := runDocSync([]string{"all", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "docsync all", err)
		}},
		{"docsync ci", func() error {
			res, err := runDocSync([]string{"ci", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "docsync ci", err)
		}},
		{"docsync release", func() error {
			res, err := runDocSync([]string{"release", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "docsync release", err)
		}},
		{"skill verify", func() error {
			res, err := runSkill([]string{"verify", "--all", "--profile", "Smoke", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "skill verify", err)
		}},
		{"lifecycle plan", func() error {
			res, err := runLifecycle([]string{"plan", "--action", "install", "--all", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "lifecycle plan", err)
		}},
		{"smoke", func() error {
			res, err := runSmoke([]string{"--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "smoke", err)
		}},
		{"ci", func() error {
			res, err := runCI([]string{"--profile", "Smoke", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "ci", err)
		}},
		{"full", func() error {
			res, err := runFull([]string{"--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "full", err)
		}},
		{"release gate", func() error {
			t.Setenv("AICODING_SKIP_FRESH_CLONE", "1")
			res, err := runReleaseCommand([]string{"gate", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "release gate", err)
		}},
		{"export", func() error {
			res, err := runExport([]string{"--all", "--zip", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK && res.Command == "export --all --zip", err)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFreshCloneCommandReportsGoPathErrors(t *testing.T) {
	missingRepo := filepath.Join(t.TempDir(), "missing")
	res, err := runFreshClone([]string{"--repo-root", missingRepo, "--json"}, time.Now())
	if err == nil || res.OK || res.Command != "fresh-clone" {
		t.Fatalf("expected fresh-clone to report a Go command error, res=%#v err=%v", res, err)
	}
}

func TestC99StandardCSkillCommandsRouteToCStyle(t *testing.T) {
	repo := t.TempDir()
	writeC99SkillFixture(t, repo)

	res, err := runSkill([]string{"c99-standard-c", "templates", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "skill c99-standard-c templates" {
		t.Fatalf("skill c99-standard-c templates failed: res=%#v err=%v", res, err)
	}

}

func writeC99SkillFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "config", "skills", "c99-standard-c", "skill.json"), `{
  "schemaVersion": 1,
  "id": "c99-standard-c",
  "title": "C99 Standard C Skill",
  "language": "c",
  "standard": "c99",
  "formatter": { "id": "clang-format", "config": "style/clang-format.yaml" },
  "commentTemplates": "templates/comment-templates.json",
  "rules": "rules/embedded-c-rules.md",
  "excludedDirectories": ["vendor", "third_party", "generated", "Drivers", "device", "build", "out", "dist"]
}
`)
	mustWrite(t, filepath.Join(repo, "config", "skills", "c99-standard-c", "style", "clang-format.yaml"), "BasedOnStyle: LLVM\n")
	mustWrite(t, filepath.Join(repo, "config", "skills", "c99-standard-c", "templates", "comment-templates.json"), `{
  "schemaVersion": 1,
  "templates": [
    {
      "id": "c-file-header-cn",
      "title": "C File Header (CN)",
      "description": "中文 C 文件头注释模板。",
      "language": "c",
      "kind": "file-header",
      "body": ["/**", " * @brief {{brief}}", " */"],
      "variables": { "author": { "description": "作者。", "default": "HU JIAXUAN" } }
    }
  ]
}
`)
	mustWrite(t, filepath.Join(repo, "config", "skills", "c99-standard-c", "rules", "embedded-c-rules.md"), "# rules\n")
}

func writeGoControlFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/aicoding-fixture\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(repo, "README.md"), "# AiCoding\n\nAiCoding is the local AI coding platform.\n\nGit Governance Standard\n\nfeat fix docs style refactor perf test build ci chore\n\nmain develop feature test release hotfix\n\nRelease typed notes\n")
	mustWrite(t, filepath.Join(repo, "README_EN.md"), "# AiCoding\n\nGit Governance Standard\n\nfeat fix docs style refactor perf test build ci chore\n")
	writeReleaseFixture(t, repo)
	mustWrite(t, filepath.Join(repo, ".github", "repository-governance.toml"), "[readme]\nprimary_language = \"zh-CN\"\nsecondary_language_surface = \"top-file-language-switch-and-github-about\"\nenglish_language_file = \"README_EN.md\"\nquick_environment_preview = true\n\n[github_about]\nrequire_bilingual = true\n\n[release]\nnotes_template = \".github/RELEASE_TEMPLATE.md\"\nnotes_validator = \"bin/aicoding.exe verify release-notes --json\"\nrequired_bilingual_sections = [\"Summary\"]\n\n[changelog]\nmode = \"unreleased\"\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit --json\npwsh -File scripts/fallback.ps1\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "commit-msg"), "go run ./cmd/aicoding hook commit-msg --file $1\npwsh -File scripts/fallback.ps1\n")
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "version: '3'\n")
	mustWrite(t, filepath.Join(repo, "config", "tagging-policy.json"), "{\"schemaVersion\":1}\n")
	mustWrite(t, filepath.Join(repo, "config", "docs-sync.policy.json"), "{}\n")
	mustWrite(t, filepath.Join(repo, "config", "docs-sync.semantic.json"), "{}\n")
	mustWrite(t, filepath.Join(repo, ".github", "workflows", "aicoding-ci.yml"), "name: docs\n")
	mustWrite(t, filepath.Join(repo, "internal", "docsync", "docsync.go"), "package docsync\n")
	mustWrite(t, filepath.Join(repo, "internal", "docsync", "check.go"), "package docsync\n")
	mustWrite(t, filepath.Join(repo, "docs", "COMMANDS.md"), "# Commands\n")
	mustWrite(t, filepath.Join(repo, "docs", "DOC_SYNC_PLUS_SPEC.md"), "# DocSync Spec\n")
	mustWrite(t, filepath.Join(repo, "docs", "DOC_SYNC_PLUS_VALIDATION_PLAN.md"), "# DocSync Validation\n")
	mustWrite(t, filepath.Join(repo, "config", "codex-kit.json"), minimalCodexKitConfig())
	mustWrite(t, filepath.Join(repo, ".agents", "plugins", "marketplace.json"), "{\"plugins\":[{\"name\":\"aicoding\",\"source\":{\"path\":\"CodingKit/agents/skills/plugins/AiCoding\"}}]}\n")
	mustWrite(t, filepath.Join(repo, "config", "kit-registry.json"), "{\"schemaVersion\":1,\"name\":\"test\",\"defaultMode\":\"all\",\"kits\":[{\"id\":\"sample-kit\",\"enabled\":true,\"order\":1,\"manifest\":\"config/kits/sample-kit.json\"}]}\n")
	mustWrite(t, filepath.Join(repo, "config", "kits", "sample-kit.json"), minimalKitManifest())
	mustWrite(t, filepath.Join(repo, "skills", "sample", "SKILL.md"), "---\nname: sample-skill\ndescription: Sample skill for tests.\n---\n\n# Sample\n")
	for _, dir := range []string{"CodingKit/agents/skills", "CodingKit/examples", "CodingKit/modules", "CodingKit/platforms", "CodingKit/tests", "CodingKit/tools"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func minimalCodexKitConfig() string {
	return "{\n" +
		"  \"name\": \"AiCoding\",\n" +
		"  \"version\": \"0.1.0\",\n" +
		"  \"codingKitRoot\": \"./CodingKit\",\n" +
		"  \"agents\": {\n" +
		"    \"skillsSubmodule\": \"./CodingKit/agents/skills\",\n" +
		"    \"pluginPath\": \"./CodingKit/agents/skills/plugins/AiCoding\",\n" +
		"    \"marketplacePath\": \"./.agents/plugins/marketplace.json\"\n" +
		"  },\n" +
		"  \"assets\": {\n" +
		"    \"examples\": \"./CodingKit/examples\",\n" +
		"    \"modules\": \"./CodingKit/modules\",\n" +
		"    \"platforms\": \"./CodingKit/platforms\",\n" +
		"    \"tests\": \"./CodingKit/tests\",\n" +
		"    \"tools\": \"./CodingKit/tools\"\n" +
		"  },\n" +
		"  \"rules\": {\n" +
		"    \"buildPluginInSubmodule\": false,\n" +
		"    \"pluginInstallUsesMarketplace\": true,\n" +
		"    \"hooksAreAuxiliaryConstraints\": true\n" +
		"  }\n" +
		"}\n"
}

func minimalKitManifest() string {
	return "{\n" +
		"  \"schemaVersion\": 2,\n" +
		"  \"id\": \"sample-kit\",\n" +
		"  \"name\": \"Sample Kit\",\n" +
		"  \"version\": \"0.1.0\",\n" +
		"  \"kind\": [\"test\"],\n" +
		"  \"mode\": \"go-builtin\",\n" +
		"  \"commands\": {\n" +
		"    \"install\": {\"type\": \"builtin-lifecycle\", \"supportsDryRun\": true, \"requiredPaths\": [\"README.md\"]},\n" +
		"    \"update\": {\"type\": \"builtin-lifecycle\", \"supportsDryRun\": true, \"requiredPaths\": [\"README.md\"]},\n" +
		"    \"uninstall\": {\"type\": \"builtin-lifecycle\", \"supportsDryRun\": true, \"requiredPaths\": [\"README.md\"]},\n" +
		"    \"status\": {\"type\": \"builtin-check\", \"requiredPaths\": [\"README.md\"]}\n" +
		"  },\n" +
		"  \"skills\": {\n" +
		"    \"umbrella\": {\"id\": \"sample-skill\", \"role\": \"router\", \"path\": \"skills/sample/SKILL.md\"}\n" +
		"  }\n" +
		"}\n"
}
