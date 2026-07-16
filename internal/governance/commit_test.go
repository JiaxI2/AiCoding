package governance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirstCommitSubject(t *testing.T) {
	got := FirstCommitSubject("\n# comment\nfeat(core): add fast path\nbody")
	if got != "feat(core): add fast path" {
		t.Fatalf("unexpected subject: %q", got)
	}
}

func TestLintBadCommitMessage(t *testing.T) {
	repo := t.TempDir()
	writeMinimalGovernanceRepo(t, repo)
	msgPath := filepath.Join(repo, "COMMIT_EDITMSG")
	mustWrite(t, msgPath, "bad message\n")
	errs := Lint(repo, "commit-msg", msgPath)
	if !hasErrorContaining(errs, "commit subject must start") {
		t.Fatalf("expected bad commit subject error, got %#v", errs)
	}
}

func TestLintRequiresChineseReadmeAndBadges(t *testing.T) {
	repo := t.TempDir()
	writeMinimalGovernanceRepo(t, repo)
	mustWrite(t, filepath.Join(repo, "README.md"), "# AiCoding\n\nAiCoding is the platform repository.\n\n[中文](README_CN.md) | [English](README_EN.md)\n\nGit 治理标准\n\nfeat fix docs style refactor perf test chore\n\nmain develop feature test release hotfix\n\nRelease typed notes\n")
	errs := Lint(repo, "all", "")
	if !hasErrorContaining(errs, "Chinese-first default") {
		t.Fatalf("expected Chinese-first README error, got %#v", errs)
	}
	if !hasErrorContaining(errs, "Release badge") {
		t.Fatalf("expected badge error, got %#v", errs)
	}
}

func writeMinimalGovernanceRepo(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "README.md"), "# AiCoding\n\n[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=release)](https://github.com/JiaxI2/AiCoding/releases/latest) [![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/) [![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://learn.microsoft.com/powershell/) [![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://www.python.org/) [![Taskfile](https://img.shields.io/badge/Taskfile-optional-29BEB0?logo=task&logoColor=white)](https://taskfile.dev/) [![License](https://img.shields.io/github/license/JiaxI2/AiCoding)](LICENSE)\n\nAiCoding 是本地 AI coding 工作流的平台集成仓库。\n\n[中文](README_CN.md) | [English](README_EN.md)\n\nGit 治理标准\n\nfeat fix docs style refactor perf test chore\n\nmain develop feature test release hotfix\n\nRelease typed notes\n")
	mustWrite(t, filepath.Join(repo, "README_CN.md"), "# 中文\n")
	mustWrite(t, filepath.Join(repo, "README_EN.md"), "# AiCoding\n\nGit Governance Standard\n\nfeat fix docs style refactor perf test chore\n")
	mustWrite(t, filepath.Join(repo, "CHANGELOG.md"), "# CHANGELOG\n\n## [Unreleased]\n\n- **docs**: fixture entry\n")
	mustWrite(t, filepath.Join(repo, ".github", "RELEASE_TEMPLATE.md"), "# Release\n")
	mustWrite(t, filepath.Join(repo, ".github", "repository-governance.toml"), `[readme]
primary_language = "zh-CN"
secondary_language_surface = "top-file-language-switch-and-github-about"
english_language_file = "README_EN.md"
quick_environment_preview = true

[github_about]
require_bilingual = true

[release]
notes_template = ".github/RELEASE_TEMPLATE.md"
notes_validator = "bin/aicoding.exe verify release-notes --json"
required_bilingual_sections = ["摘要 / Summary"]

[changelog]
mode = "unreleased"

[governance_standard]
id = "aicoding-git-governance"
version = "2026.07.16"
source_url = "https://github.com/JiaxI2/Codex-Skills/blob/main/platform/aicoding-git-governance/references/aicoding-git-governance-standard.md"
sync_policy = "track-canonical-url"

[issues]
enabled = true
profile = "managed-lifecycle"
templates_directory = ".github/ISSUE_TEMPLATE"
label_manifest = ".github/issue-labels.json"
workflow = ".github/workflows/issue-governance.yml"
allow_blank = false
required_label_axes = ["type", "area", "priority", "status"]
closure_requires_resolution = true
closure_requires_summary = true
auto_close_stale = false
`)
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "config.yml"), "blank_issues_enabled: false\n")
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "bug.yml"), "name: Bug\ndescription: Bug\ntitle: Bug\nlabels: [\"type:bug\", \"status:needs-triage\"]\nbody:\n  - id: existing\n  - id: current_behavior\n  - id: expected_behavior\n  - id: reproduction\n  - id: impact\n  - id: environment\n  - id: done_condition\n")
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "feature.yml"), "name: Feature\ndescription: Feature\ntitle: Feature\nlabels: [\"type:feature\", \"status:needs-triage\"]\nbody:\n  - id: existing\n  - id: problem\n  - id: outcome\n  - id: scope\n  - id: acceptance\n  - id: alternatives\n  - id: traceability\n")
	mustWrite(t, filepath.Join(repo, ".github", "ISSUE_TEMPLATE", "governance.yml"), "name: Governance\ndescription: Governance\ntitle: Governance\nlabels: [\"type:governance\", \"status:needs-triage\"]\nbody:\n  - id: existing\n  - id: gap\n  - id: proposed_rule\n  - id: lifecycle_impact\n  - id: verification\n  - id: compatibility\n  - id: rollback\n")
	mustWrite(t, filepath.Join(repo, ".github", "issue-labels.json"), `{
  "schema_version": 1,
  "labels": [
    {"name":"type:bug","color":"d73a4a","description":"bug"},
    {"name":"type:feature","color":"a2eeef","description":"feature"},
    {"name":"type:governance","color":"5319e7","description":"governance"},
    {"name":"area:test","color":"c5def5","description":"area"},
    {"name":"priority:p0","color":"b60205","description":"p0"},
    {"name":"priority:p1","color":"d93f0b","description":"p1"},
    {"name":"priority:p2","color":"fbca04","description":"p2"},
    {"name":"priority:p3","color":"0e8a16","description":"p3"},
    {"name":"status:needs-triage","color":"ededed","description":"triage"},
    {"name":"status:needs-info","color":"fef2c0","description":"info"},
    {"name":"status:ready","color":"0e8a16","description":"ready"},
    {"name":"status:in-progress","color":"1d76db","description":"progress"},
    {"name":"status:blocked","color":"b60205","description":"blocked"},
    {"name":"resolution:completed","color":"0e8a16","description":"completed"},
    {"name":"resolution:duplicate","color":"cfd3d7","description":"duplicate"},
    {"name":"resolution:not-planned","color":"ffffff","description":"not planned"},
    {"name":"resolution:invalid","color":"e4e669","description":"invalid"}
  ]
}`)
	mustWrite(t, filepath.Join(repo, ".github", "workflows", "issue-governance.yml"), "name: Issue governance\nopened\nreopened\nlabeled\nclosed\npermissions:\n  issues: write\nmanifest: .github/issue-labels.json\nuses: actions/github-script@v9\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "commit-msg"), "#!/bin/sh\n")
}

func hasErrorContaining(errs []string, needle string) bool {
	for _, err := range errs {
		if strings.Contains(err, needle) {
			return true
		}
	}
	return false
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
