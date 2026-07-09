package docsync

import (
	"os"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func LintStaged(repo string) []string {
	staged, err := gitx.StagedFiles(repo)
	if err != nil {
		return []string{err.Error()}
	}
	if len(staged) == 0 {
		return nil
	}
	docChanged := false
	riskChanged := false
	for _, f := range staged {
		if IsDocPath(f) {
			docChanged = true
		}
		if IsDocSyncRiskPath(f) {
			riskChanged = true
		}
	}
	if riskChanged && !docChanged && os.Getenv("AICODING_SKIP_DOCSYNC") != "1" {
		return []string{"documentation sync fast gate: source/script/config/hook/skill changes require staged docs or AICODING_SKIP_DOCSYNC=1; CI still runs full DocSync Plus"}
	}
	return nil
}

func IsDocPath(f string) bool {
	f = strings.ReplaceAll(f, "\\", "/")
	if strings.HasSuffix(f, ".md") {
		return true
	}
	return f == "README.md" || f == "README_CN.md" || f == "README_EN.md" || f == "CHANGELOG.md" || strings.HasPrefix(f, "docs/") || strings.HasPrefix(f, "config/") && strings.HasSuffix(f, ".md")
}

func IsDocSyncRiskPath(f string) bool {
	f = strings.ReplaceAll(f, "\\", "/")
	if strings.HasPrefix(f, ".git/") || strings.Contains(f, "/__pycache__/") || strings.Contains(f, "/.pytest_cache/") {
		return false
	}
	if strings.HasPrefix(f, "cmd/") || strings.HasPrefix(f, "internal/") || strings.HasPrefix(f, "scripts/") || strings.HasPrefix(f, "src/") || strings.HasPrefix(f, "config/") || strings.HasPrefix(f, ".githooks/") || strings.HasPrefix(f, ".github/workflows/") || strings.HasPrefix(f, ".agents/") || strings.HasPrefix(f, "CodingKit/") || strings.HasPrefix(f, "skills/") || strings.HasPrefix(f, "codex-skills/") {
		return true
	}
	for _, ext := range []string{".c", ".h", ".cpp", ".hpp", ".py", ".ps1", ".sh"} {
		if strings.HasSuffix(f, ext) {
			return true
		}
	}
	return false
}
