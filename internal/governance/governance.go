package governance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

func Lint(repo, mode, commitMsgPath string) []string {
	errs := []string{}
	fail := func(msg string) { errs = append(errs, msg) }
	requiredFiles := []string{"README.md", "README_EN.md", "CHANGELOG.md", ".github/RELEASE_TEMPLATE.md", ".github/repository-governance.toml", ".githooks/pre-commit", ".githooks/commit-msg", "scripts/legacy/fast-path-replaced/verify-release-notes.ps1"}
	for _, f := range requiredFiles {
		if !platform.IsFile(platform.RepoPath(repo, f)) {
			fail("required governance file missing: " + f)
		}
	}
	scanFiles := []string{"README.md", "README_CN.md", "README_EN.md", "CHANGELOG.md", ".github/repository-governance.toml"}
	placeholder := regexp.MustCompile(`\{\{[^}]+\}\}|UNRESOLVED_PLACEHOLDER|TODO_PLACEHOLDER`)
	for _, f := range scanFiles {
		p := platform.RepoPath(repo, f)
		if !platform.IsFile(p) {
			continue
		}
		content, err := platform.ReadText(p)
		if err != nil {
			fail("cannot read " + f + ": " + err.Error())
			continue
		}
		if placeholder.MatchString(content) {
			fail("unresolved placeholder found in " + f)
		}
	}
	readme, _ := platform.ReadText(platform.RepoPath(repo, "README.md"))
	readmeEN, _ := platform.ReadText(platform.RepoPath(repo, "README_EN.md"))
	gov, _ := platform.ReadText(platform.RepoPath(repo, ".github/repository-governance.toml"))
	changelog, _ := platform.ReadText(platform.RepoPath(repo, "CHANGELOG.md"))
	readmeHead := strings.Join(firstLines(readme, 16), "\n")
	readmeTop := strings.Join(firstLines(readme, 24), "\n")
	if platform.IsFile(platform.RepoPath(repo, "README_CN.md")) {
		if !strings.Contains(readmeHead, "README_CN.md") {
			fail("README.md must include top-of-file README_CN.md link")
		}
		if !strings.Contains(readmeHead, "README_EN.md") {
			fail("README.md must include top-of-file README_EN.md link")
		}
		if strings.Contains(readmeHead, "README.md#english") {
			fail("README.md must not use in-page English anchor")
		}
		if !strings.Contains(readmeTop, "AiCoding 是") {
			fail("README.md must be the Chinese-first default repository entry")
		}
		for _, req := range []struct {
			needle string
			msg    string
		}{
			{"img.shields.io/github/v/release/JiaxI2/AiCoding", "README.md must keep the Release badge link"},
			{"https://go.dev/", "README.md must keep the Go URL badge link"},
			{"https://learn.microsoft.com/powershell/", "README.md must keep the PowerShell URL badge link"},
			{"https://www.python.org/", "README.md must keep the Python URL badge link"},
			{"https://taskfile.dev/", "README.md must keep the Taskfile URL badge link"},
			{"github/license/JiaxI2/AiCoding", "README.md must keep the License badge link"},
		} {
			mustContain(readmeHead, req.needle, req.msg, fail)
		}
		for _, req := range []struct {
			needle string
			msg    string
		}{
			{"## 环境 URL / Environment URLs", "README.md must keep the Environment URLs section"},
			{"https://github.com/JiaxI2/AiCoding/releases/latest", "README.md must keep the latest release URL"},
			{"https://github.com/JiaxI2/AiCoding/releases", "README.md must keep the releases URL"},
			{"https://github.com/JiaxI2/AiCoding/tags", "README.md must keep the tags URL"},
			{"[CHANGELOG.md](CHANGELOG.md)", "README.md must keep the CHANGELOG link"},
			{"[CodingKit/README.md](CodingKit/README.md)", "README.md must keep the CodingKit README link"},
		} {
			mustContain(readme, req.needle, req.msg, fail)
		}
		mustContain(gov, `primary_language = "zh-CN"`, ".github/repository-governance.toml must set README primary_language to zh-CN", fail)
		mustContain(gov, `secondary_language_surface = "top-file-language-switch-and-github-about"`, ".github/repository-governance.toml must define secondary language surface", fail)
		mustContain(gov, `english_language_file = "README_EN.md"`, ".github/repository-governance.toml must define README_EN.md", fail)
		mustContain(gov, `quick_environment_preview = true`, ".github/repository-governance.toml must require quick environment preview", fail)
		if !strings.Contains(gov, "[github_about]") || !strings.Contains(gov, "require_bilingual = true") {
			fail(".github/repository-governance.toml must require bilingual GitHub About metadata")
		}
	}
	reGitGov := regexp.MustCompile(`Git Governance Standard|Git 治理标准`)
	reCommit := regexp.MustCompile(`feat.+fix.+docs.+style.+refactor.+perf.+test.+chore|feat.+fix.+docs.+build.+ci.+chore`)
	reBranch := regexp.MustCompile(`main.+master.+develop.+feature.+test.+release.+hotfix|main.+develop.+feature.+test.+release.+hotfix`)
	reRelease := regexp.MustCompile(`Release.+type|Release.+typed|按类型汇总|主类型`)
	if !reGitGov.MatchString(readme) {
		fail("README.md must document Git governance standard")
	}
	if !reCommit.MatchString(readme) {
		fail("README.md must document commit type taxonomy")
	}
	if !reBranch.MatchString(readme) {
		fail("README.md must document branch naming and environment mapping")
	}
	if !reRelease.MatchString(readme) {
		fail("README.md must document typed release notes")
	}
	if !reGitGov.MatchString(readmeEN) {
		fail("README_EN.md must document Git governance standard")
	}
	if !reCommit.MatchString(readmeEN) {
		fail("README_EN.md must document commit type taxonomy")
	}
	if !regexp.MustCompile(`notes_template\s*=\s*"\.github/RELEASE_TEMPLATE\.md"`).MatchString(gov) {
		fail("repository-governance.toml must declare release notes template")
	}
	if !regexp.MustCompile(`notes_validator\s*=\s*"scripts/legacy/fast-path-replaced/verify-release-notes\.ps1"`).MatchString(gov) {
		fail("repository-governance.toml must declare release notes validator")
	}
	if !strings.Contains(gov, "required_bilingual_sections") {
		fail("repository-governance.toml must require bilingual release notes sections")
	}
	if strings.Contains(gov, `mode = "unreleased"`) && !strings.Contains(changelog, "[Unreleased]") {
		fail("CHANGELOG.md must contain [Unreleased] when changelog.mode is unreleased")
	}
	if !regexp.MustCompile(`\*\*(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]*\))?\*\*`).MatchString(changelog) {
		fail("CHANGELOG.md must include typed entries such as **docs** or **chore**")
	}
	if mode == "all" || mode == "pre-commit" {
		staged, err := gitx.StagedFiles(repo)
		if err != nil {
			fail(err.Error())
		} else if len(staged) > 0 && !contains(staged, "CHANGELOG.md") && os.Getenv("AICODING_SKIP_CHANGELOG") != "1" {
			fail("CHANGELOG.md must be staged for normal commits; set AICODING_SKIP_CHANGELOG=1 only for approved exclusion")
		}
	}
	if mode == "commit-msg" {
		if commitMsgPath == "" {
			fail("commit message path is required")
		} else {
			p := commitMsgPath
			if !filepath.IsAbs(p) {
				p = filepath.Join(repo, p)
			}
			content, err := platform.ReadText(p)
			if err != nil {
				fail("cannot read commit message: " + err.Error())
			} else {
				subject := FirstCommitSubject(content)
				if subject == "" {
					fail("commit message subject is empty")
				}
				if !regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]+\))?: `).MatchString(subject) {
					fail("commit subject must start with allowed type and optional scope. Got: " + subject)
				}
				if !regexp.MustCompile(`: .{8,}$`).MatchString(subject) {
					fail("commit subject summary must be at least 8 characters. Got: " + subject)
				}
			}
		}
	}
	return errs
}

func FirstCommitSubject(s string) string {
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

func mustContain(content, needle, msg string, fail func(string)) {
	if !strings.Contains(content, needle) {
		fail(msg)
	}
}

func firstLines(s string, n int) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) > n {
		return lines[:n]
	}
	return lines
}

func contains(list []string, target string) bool {
	for _, x := range list {
		if x == target {
			return true
		}
	}
	return false
}
