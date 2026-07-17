package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/cstyle"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
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

func installPassingTestEngine(t *testing.T) {
	t.Helper()
	previousRunTestEngine := runTestEngine
	t.Cleanup(func() {
		runTestEngine = previousRunTestEngine
	})
	runTestEngine = func(_ context.Context, cfg testengine.Config) (testengine.Report, error) {
		if err := os.MkdirAll(cfg.Out, 0o755); err != nil {
			return testengine.Report{}, err
		}
		testReport := testengine.Report{
			Summary: testengine.Summary{
				Repo:       cfg.Repo,
				Profile:    cfg.Profile,
				StartedAt:  "2026-07-09T00:00:00+08:00",
				EndedAt:    "2026-07-09T00:00:01+08:00",
				DurationMS: 1,
				Total:      1,
				Pass:       1,
				Conclusion: "PASS",
			},
			Results: []testengine.Result{{
				ID:        "FIX-001",
				Category:  "FIXTURE",
				Title:     "fixture",
				Status:    testengine.Pass,
				Severity:  testengine.Required,
				ExitCode:  0,
				JSONValid: true,
				Command:   "fixture",
				Reason:    "command passed",
				Profile:   cfg.Profile,
			}},
		}
		if err := testengine.Write(cfg.Out, testReport); err != nil {
			return testReport, err
		}
		return testReport, nil
	}
}

func writeReleaseFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "CHANGELOG.md"), "# CHANGELOG\n\n## [Unreleased]\n\n- **docs**: test fixture.\n")
	mustWrite(t, filepath.Join(repo, ".github", "RELEASE_TEMPLATE.md"), "## 摘要 / Summary\n\n## 变更内容 / What's Changed\n\n## 可追溯性 / Traceability\n")
	mustWrite(t, filepath.Join(repo, "docs", "governance", "TAGGING_POLICY.md"), "vMAJOR.MINOR.PATCH\nkit/<kit-id>/vMAJOR.MINOR.PATCH\nmilestone/YYYY.MM.DD-<name>\n")
	mustWrite(t, filepath.Join(repo, "docs", "governance", "RELEASE_POLICY.md"), "Platform Release\nKit / Component Release\nMilestone Release\n")
	for _, rel := range []string{
		"docs/governance/RELEASE_GOVERNANCE_OVERLAY.md",
		"tools/specialty/aicoding-tag-governance.ps1",
		"tools/specialty/verify-release-governance-overlay.ps1",
		"config/kits/release-governance-overlay-kit.json",
		".aicoding/templates/perf-cache-plan.json",
	} {
		mustWrite(t, filepath.Join(repo, filepath.FromSlash(rel)), "ok\n")
	}
}

func writeIssueGovernanceFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "config.yml"), "blank_issues_enabled: false\n")
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "bug.yml"), "name: Bug\ndescription: Bug\ntitle: Bug\nlabels: [\"type:bug\", \"status:needs-triage\"]\nbody:\n  - id: existing\n  - id: current_behavior\n  - id: expected_behavior\n  - id: reproduction\n  - id: impact\n  - id: environment\n  - id: done_condition\n")
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "feature.yml"), "name: Feature\ndescription: Feature\ntitle: Feature\nlabels: [\"type:feature\", \"status:needs-triage\"]\nbody:\n  - id: existing\n  - id: problem\n  - id: outcome\n  - id: scope\n  - id: acceptance\n  - id: alternatives\n  - id: traceability\n")
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "governance.yml"), "name: Governance\ndescription: Governance\ntitle: Governance\nlabels: [\"type:governance\", \"status:needs-triage\"]\nbody:\n  - id: existing\n  - id: gap\n  - id: proposed_rule\n  - id: lifecycle_impact\n  - id: verification\n  - id: compatibility\n  - id: rollback\n")
	labelNames := []string{
		"type:bug", "type:feature", "type:governance", "area:test",
		"priority:p0", "priority:p1", "priority:p2", "priority:p3",
		"status:needs-triage", "status:needs-info", "status:ready", "status:in-progress", "status:blocked",
		"resolution:completed", "resolution:duplicate", "resolution:not-planned", "resolution:invalid",
	}
	labels := make([]map[string]string, 0, len(labelNames))
	for _, name := range labelNames {
		labels = append(labels, map[string]string{"name": name, "color": "123abc", "description": name})
	}
	manifest, err := json.Marshal(map[string]interface{}{"schema_version": 1, "labels": labels})
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, ".github", "issue-labels.json"), string(manifest))
	mustWrite(t, filepath.Join(repo, ".github", "workflows", "issue-governance.yml"), "name: Issue governance\nopened\nreopened\nlabeled\nclosed\npermissions:\n  issues: write\nmanifest: .github/issue-labels.json\nuses: actions/github-script@v9\n")
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
		`case "test":`,
		`res, err = runTest`,
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
		`case "codex":`,
		`res, err = runCodexUsage`,
		`aicoding test full|release`,
		`aicoding release gate`,
		`aicoding codex usage parse`,
		`aicoding codex usage run`,
		`aicoding skill c99-standard-c status`,
		`aicoding skill c99-standard-c verify`,
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
	installPassingTestEngine(t)
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
			if err != nil || !res.OK || res.Command != "docsync release" {
				return resultErr(false, err)
			}
			if _, ok := res.Data.(report.StandardReport); !ok {
				return os.ErrInvalid
			}
			return nil
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
	if data, ok := res.Data.(report.StandardReport); !ok || data.Status != "PASS" || data.Profile != "c99-standard-c" {
		t.Fatalf("expected standard C99 report data, got %#v", res.Data)
	}

}

func TestC99StandardCSkillVerifyWrapsCStyleKitJSON(t *testing.T) {
	repo := t.TempDir()
	writeC99SkillFixture(t, repo)
	writeFakeCStyleKit(t, repo)
	mustWrite(t, filepath.Join(repo, "fixtures", "target.json"), "{}\n")
	mustWrite(t, filepath.Join(repo, "overlays", "project.json"), "{}\n")

	res, err := runSkill([]string{
		"c99-standard-c", "verify",
		"--profile", "fast",
		"--target", "fixtures/target.json",
		"--overlay", "overlays/project.json",
		"--timings",
		"--repo-root", repo,
		"--json",
	}, time.Now())
	if err != nil || !res.OK || res.Command != "skill c99-standard-c verify" {
		t.Fatalf("skill c99-standard-c verify failed: res=%#v err=%v", res, err)
	}
	standard, ok := res.Data.(report.StandardReport)
	if !ok || standard.Status != "PASS" || standard.Profile != "fast" {
		t.Fatalf("expected standard C99 verify report, got %#v", res.Data)
	}
	details, ok := standard.Details.(cstyle.VerifyResult)
	if !ok || details.Payload["ok"] != true || details.Target != "fixtures/target.json" {
		t.Fatalf("unexpected C Kit verify details: %#v", standard.Details)
	}
}

func TestC99StandardCSkillVerifyRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"c99-standard-c", "verify", "--bogus", "--json"},
		{"c99-standard-c", "verify", "--target"},
		{"c99-standard-c", "verify", "unexpected.json"},
	} {
		res, err := runSkill(args, time.Now())
		if err == nil || res.OK {
			t.Fatalf("invalid arguments must fail: args=%#v res=%#v err=%v", args, res, err)
		}
	}
}

func TestRunTestProfileWrapsRepoTester(t *testing.T) {
	repo := t.TempDir()
	installPassingTestEngine(t)

	res, err := runTest([]string{"full", "--repo-root", repo, "--runner-timeout-sec", "30", "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "test full" {
		t.Fatalf("test full failed: res=%#v err=%v", res, err)
	}
	data, ok := res.Data.(report.StandardReport)
	if !ok {
		t.Fatalf("expected standard report data, got %#v", res.Data)
	}
	if data.Status != "PASS" || data.Profile != "full" {
		t.Fatalf("unexpected test data: %#v", data)
	}
	if _, err := os.Stat(filepath.Join(repo, "test-results")); err != nil {
		t.Fatalf("expected test-results output: %v", err)
	}

	canonicalOut := filepath.Join(repo, "canonical-smoke-results")
	res, err = runTest([]string{"--profile", "Smoke", "--repo-root", repo, "--out", canonicalOut, "--runner-timeout-sec", "30", "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "test --profile Smoke" {
		t.Fatalf("canonical test profile failed: res=%#v err=%v", res, err)
	}
	data, ok = res.Data.(report.StandardReport)
	if !ok || data.Status != "PASS" || data.Profile != "smoke" {
		t.Fatalf("unexpected canonical test data: %#v", res.Data)
	}

	if res, err = runTest([]string{"full", "--profile", "Release", "--json"}, time.Now()); err == nil || res.OK || !isUsageError(err) {
		t.Fatalf("legacy positional profile must not accept --profile override: res=%#v err=%v", res, err)
	}
}

func TestCompatibilityTestCommandsRouteDirectlyToEngine(t *testing.T) {
	repo := t.TempDir()
	installPassingTestEngine(t)
	passingEngine := runTestEngine
	profiles := []string{}
	runTestEngine = func(ctx context.Context, cfg testengine.Config) (testengine.Report, error) {
		profiles = append(profiles, cfg.Profile)
		return passingEngine(ctx, cfg)
	}

	for _, tc := range []struct {
		name    string
		run     func() (report.Result, error)
		command string
	}{
		{"smoke", func() (report.Result, error) {
			return runSmoke([]string{"--repo-root", repo, "--json"}, time.Now())
		}, "smoke"},
		{"ci", func() (report.Result, error) {
			return runCI([]string{"--profile", "Release", "--repo-root", repo, "--json"}, time.Now())
		}, "ci"},
		{"full", func() (report.Result, error) {
			return runFull([]string{"--repo-root", repo, "--json"}, time.Now())
		}, "full"},
		{"release gate", func() (report.Result, error) {
			return runReleaseCommand([]string{"gate", "--repo-root", repo, "--json"}, time.Now())
		}, "release gate"},
	} {
		res, err := tc.run()
		if err != nil || !res.OK || res.Command != tc.command {
			t.Fatalf("%s route failed: res=%#v err=%v", tc.name, res, err)
		}
	}

	wantProfiles := []string{"smoke", "release", "full", "release"}
	if len(profiles) != len(wantProfiles) {
		t.Fatalf("engine call count = %d, want %d: %#v", len(profiles), len(wantProfiles), profiles)
	}
	for index := range wantProfiles {
		if profiles[index] != wantProfiles[index] {
			t.Fatalf("engine profile[%d] = %q, want %q", index, profiles[index], wantProfiles[index])
		}
	}
}

func TestRunTestLatestReadsLatestReport(t *testing.T) {
	repo := t.TempDir()
	older := filepath.Join(repo, "test-results", "aicoding-global-test-20260101-000000")
	newer := filepath.Join(repo, "test-results", "aicoding-global-test-20260102-000000")
	writeGlobalTestReport(t, older, "full", "PASS", 1)
	writeGlobalTestReport(t, newer, "release", "PASS", 2)

	res, err := runTest([]string{"latest", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "test latest" {
		t.Fatalf("test latest failed: res=%#v err=%v", res, err)
	}
	data := res.Data.(report.StandardReport)
	if data.Profile != "release" {
		t.Fatalf("expected latest release report, got %#v", data)
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
  "kit": {
    "id": "c-userstyle-kit",
    "version": "test",
    "root": "CodingKit/tools/c-userstyle-kit",
    "config": "CodingKit/tools/c-userstyle-kit/examples/c-kit.json",
    "snippets": "CodingKit/tools/c-userstyle-kit/examples/c-snippets.json",
    "quickTarget": "CodingKit/tools/c-userstyle-kit/examples/verify-target.json"
  },
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

func writeFakeCStyleKit(t *testing.T, repo string) {
	t.Helper()
	root := filepath.Join(repo, filepath.FromSlash(cstyle.DefaultKitRoot))
	mustWrite(t, filepath.Join(root, "go.mod"), "module c-userstyle-kit\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(root, "cmd", "cstylekit", "main.go"), `package main

import (
	"encoding/json"
	"os"
)

func main() {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		"schema":  "cstylekit.verify.v1",
		"ok":      true,
		"profile": "fast",
	})
}
`)
	mustWrite(t, filepath.Join(root, filepath.FromSlash(cstyle.DefaultKitConfig)), "{}\n")
	mustWrite(t, filepath.Join(root, filepath.FromSlash(cstyle.DefaultKitSnippets)), "{}\n")
	mustWrite(t, filepath.Join(root, filepath.FromSlash(cstyle.DefaultKitQuickTarget)), "{}\n")
}

func writeGoControlFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/aicoding-fixture\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(repo, "README.md"), "# AiCoding\n\n[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=release)](https://github.com/JiaxI2/AiCoding/releases/latest) [![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/go1.22) [![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://github.com/PowerShell/PowerShell/releases/tag/v7.0.0) [![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://docs.python.org/3.10/whatsnew/3.10.html) [![Taskfile](https://img.shields.io/badge/Taskfile-optional-29BEB0?logo=task&logoColor=white)](https://taskfile.dev/) [![License](https://img.shields.io/github/license/JiaxI2/AiCoding)](LICENSE)\n\nAiCoding 是本地 AI coding 工作流的平台集成仓库。\n\n[中文](README_CN.md) | [English](README_EN.md)\n\nGit Governance Standard\n\nfeat fix docs style refactor perf test build ci chore\n\nmain develop feature test release hotfix\n\nRelease typed notes\n")
	mustWrite(t, filepath.Join(repo, "README_EN.md"), "# AiCoding\n\nGit Governance Standard\n\nfeat fix docs style refactor perf test build ci chore\n")
	writeReleaseFixture(t, repo)
	mustWrite(t, filepath.Join(repo, ".github", "repository-governance.toml"), "[readme]\nprimary_language = \"zh-CN\"\nsecondary_language_surface = \"top-file-language-switch-and-github-about\"\nenglish_language_file = \"README_EN.md\"\nquick_environment_preview = true\n\n[github_about]\nrequire_bilingual = true\n\n[release]\nnotes_template = \".github/RELEASE_TEMPLATE.md\"\nnotes_validator = \"bin/aicoding.exe verify release-notes --json\"\nrequired_bilingual_sections = [\"Summary\"]\n\n[changelog]\nmode = \"unreleased\"\n\n[governance_standard]\nid = \"aicoding-git-governance\"\nversion = \"2026.07.16\"\nsource_url = \"https://github.com/JiaxI2/Codex-Skills/blob/main/platform/aicoding-git-governance/references/aicoding-git-governance-standard.md\"\nsync_policy = \"track-canonical-url\"\n\n[issues]\nenabled = true\nprofile = \"managed-lifecycle\"\ntemplates_directory = \".github/ISSUE_TEMPLATE\"\nlabel_manifest = \".github/issue-labels.json\"\nworkflow = \".github/workflows/issue-governance.yml\"\nallow_blank = false\nrequired_label_axes = [\"type\", \"area\", \"priority\", \"status\"]\nclosure_requires_resolution = true\nclosure_requires_summary = true\nauto_close_stale = false\n")
	writeIssueGovernanceFixture(t, repo)
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit --json\npwsh -File tools/specialty/fallback.ps1\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "commit-msg"), "go run ./cmd/aicoding hook commit-msg --file $1\npwsh -File tools/specialty/fallback.ps1\n")
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "version: '3'\n")
	mustWrite(t, filepath.Join(repo, "config", "tagging-policy.json"), "{\"schemaVersion\":1}\n")
	mustWrite(t, filepath.Join(repo, "config", "docs-sync.policy.json"), "{}\n")
	mustWrite(t, filepath.Join(repo, "config", "docs-sync.semantic.json"), "{}\n")
	mustWrite(t, filepath.Join(repo, ".github", "workflows", "aicoding-ci.yml"), "name: docs\n")
	mustWrite(t, filepath.Join(repo, "internal", "docsync", "docsync.go"), "package docsync\n")
	mustWrite(t, filepath.Join(repo, "internal", "docsync", "check.go"), "package docsync\n")
	mustWrite(t, filepath.Join(repo, "docs", "COMMANDS.md"), "# Commands\n")
	mustWrite(t, filepath.Join(repo, "docs", "architecture", "DOC_SYNC_PLUS_SPEC.md"), "# DocSync Spec\n")
	mustWrite(t, filepath.Join(repo, "docs", "operations", "DOC_SYNC_PLUS_VALIDATION_PLAN.md"), "# DocSync Validation\n")
	mustWrite(t, filepath.Join(repo, "docs", "operations", "THIRD_PARTY_REUSE_GOVERNANCE.md"), "DocSync\n")
	mustWrite(t, filepath.Join(repo, "config", "codex-kit.json"), minimalCodexKitConfig())
	writeDependencyGovernanceFixture(t, repo)
	mustWrite(t, filepath.Join(repo, ".agents", "plugins", "marketplace.json"), "{\"plugins\":[{\"name\":\"aicoding\",\"source\":{\"path\":\"CodingKit/agents/skills/plugins/AiCoding\"}}]}\n")
	mustWrite(t, filepath.Join(repo, "config", "kit-registry.json"), "{\"schemaVersion\":1,\"name\":\"test\",\"defaultMode\":\"all\",\"kits\":[{\"id\":\"sample-kit\",\"enabled\":true,\"order\":1,\"manifest\":\"config/kits/sample-kit.json\"}]}\n")
	mustWrite(t, filepath.Join(repo, "config", "reuse-governance.json"), minimalReuseGovernanceConfig())
	mustWrite(t, filepath.Join(repo, "config", "kits", "sample-kit.json"), minimalKitManifest())
	mustWrite(t, filepath.Join(repo, "skills", "sample", "SKILL.md"), "---\nname: sample-skill\ndescription: Sample skill for tests.\n---\n\n# Sample\n")
	for _, dir := range []string{"CodingKit/agents/skills", "CodingKit/examples", "CodingKit/modules", "CodingKit/platforms", "CodingKit/tests", "CodingKit/tools"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeGlobalTestReport(t *testing.T, dir string, profile string, conclusion string, pass int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary := testengine.Summary{
		Repo:       filepath.Dir(filepath.Dir(dir)),
		Profile:    profile,
		StartedAt:  "2026-07-09T00:00:00+08:00",
		EndedAt:    "2026-07-09T00:00:01+08:00",
		DurationMS: 1,
		Total:      pass,
		Pass:       pass,
		Conclusion: conclusion,
	}
	fileReport := testengine.Report{Summary: summary, Results: []testengine.Result{{
		ID:       "FIX-001",
		Category: "FIXTURE",
		Title:    "fixture",
		Status:   testengine.Pass,
		Severity: testengine.Required,
		Reason:   "command passed",
		Profile:  profile,
	}}}
	if err := testengine.Write(dir, fileReport); err != nil {
		t.Fatal(err)
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

func minimalReuseGovernanceConfig() string {
	return `{
  "schemaVersion": 1,
  "policy": {
    "requireAttributionForCopiedContent": true,
    "requireIndependentRuntime": true,
    "requireRollback": true,
    "requireNoPublicAPI": true
  },
  "modules": [
    {
      "id": "evidence-gate",
      "classification": "reimplemented",
      "state": "pilot",
      "literalExternalContent": false,
      "runtimeDependency": false,
      "publicAPI": false,
      "integrations": ["go-cli", "skill-verify", "hook", "ci", "docsync", "lifecycle"],
      "requiredPaths": ["config/reuse-governance.json"],
      "evidence": [
        {"integration": "go-cli", "path": "README.md", "contains": "AiCoding"},
        {"integration": "skill-verify", "path": "docs/COMMANDS.md", "contains": "Commands"},
        {"integration": "hook", "path": ".githooks/pre-commit", "contains": "hook pre-commit"},
        {"integration": "ci", "path": ".github/workflows/aicoding-ci.yml", "contains": "name: docs"},
        {"integration": "docsync", "path": "docs/operations/THIRD_PARTY_REUSE_GOVERNANCE.md", "contains": "DocSync"},
        {"integration": "lifecycle", "path": "config/kit-registry.json", "contains": "sample-kit"}
      ],
      "rollback": {"strategy": "remove", "statePath": ".aicoding/state/kits/reuse-governance"}
    }
  ]
}
`
}
