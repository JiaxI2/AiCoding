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
	mustWrite(t, filepath.Join(repo, "README.md"), "# AiCoding\n\n[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=release)](https://github.com/JiaxI2/AiCoding/releases/latest) [![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/) [![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://learn.microsoft.com/powershell/) [![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://www.python.org/) [![Taskfile](https://img.shields.io/badge/Taskfile-optional-29BEB0?logo=task&logoColor=white)](https://taskfile.dev/) [![License](https://img.shields.io/github/license/JiaxI2/AiCoding)](LICENSE)\n\nAiCoding 是本地 AI coding 工作流的平台集成仓库。\n\n[中文](README_CN.md) | [English](README_EN.md)\n\n## 环境 URL / Environment URLs\n\nhttps://github.com/JiaxI2/AiCoding/releases/latest\nhttps://github.com/JiaxI2/AiCoding/releases\nhttps://github.com/JiaxI2/AiCoding/tags\n[CHANGELOG.md](CHANGELOG.md)\n[CodingKit/README.md](CodingKit/README.md)\n\nGit 治理标准\n\nfeat fix docs style refactor perf test chore\n\nmain develop feature test release hotfix\n\nRelease typed notes\n")
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
notes_validator = "scripts/legacy/fast-path-replaced/verify-release-notes.ps1"
required_bilingual_sections = ["摘要 / Summary"]

[changelog]
mode = "unreleased"
`)
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "commit-msg"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(repo, "scripts", "legacy", "fast-path-replaced", "verify-release-notes.ps1"), "# fixture\n")
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
