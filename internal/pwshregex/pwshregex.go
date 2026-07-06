package pwshregex

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type Issue struct {
	File           string `json:"file"`
	Line           int    `json:"line"`
	Rule           string `json:"rule"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
	Snippet        string `json:"snippet,omitempty"`
}

var (
	doubleQuotedCaptureReplacement = regexp.MustCompile(`(?i)-(?:c|i)?replace\b[^,\r\n]*,\s*"[^"\r\n]*\$(?:\d+|\{[A-Za-z_][A-Za-z0-9_]*\})`)
	dynamicCallbackReplacement     = regexp.MustCompile(`(?i)-(?:c|i)?replace\b[^,\r\n]*,\s*\{`)
	ps7Requires                    = regexp.MustCompile(`(?im)^\s*#requires\s+-Version\s+7(?:\.0)?\b`)
)

func IsPowerShellPath(path string) bool {
	path = strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.HasSuffix(path, ".ps1") || strings.HasSuffix(path, ".psm1") || strings.HasSuffix(path, ".psd1")
}

func isBadFixturePath(path string) bool {
	path = strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(path, "/tests/cases/bad/") || strings.HasSuffix(path, "/tests/cases/bad")
}

func LintText(name, text string) []Issue {
	issues := []Issue{}
	normalizedText := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(normalizedText, "\n")
	hasPS7Require := ps7Requires.MatchString(normalizedText)

	for i, line := range lines {
		lineNo := i + 1
		if doubleQuotedCaptureReplacement.MatchString(line) {
			issues = append(issues, Issue{
				File:           name,
				Line:           lineNo,
				Rule:           "PSRegex001.DoubleQuotedCaptureReplacement",
				Severity:       "error",
				Message:        "regex replacement capture tokens must not be written inside double quotes",
				Recommendation: "Use single-quoted replacement tokens such as '$1' or '${Name}'.",
				Snippet:        strings.TrimSpace(line),
			})
		}

		lower := strings.ToLower(line)
		if strings.Contains(lower, "get-content") &&
			strings.Contains(lower, "foreach-object") &&
			strings.Contains(lower, "-replace") &&
			!strings.Contains(lower, "-raw") {
			issues = append(issues, Issue{
				File:           name,
				Line:           lineNo,
				Rule:           "PSRegex002.LinePipelineReplace",
				Severity:       "error",
				Message:        "line-by-line pipeline regex replacement is not allowed for agent edits",
				Recommendation: "Use Get-Content -Raw, perform one in-memory replacement, then Set-Content -NoNewline.",
				Snippet:        strings.TrimSpace(line),
			})
		}

		if dynamicCallbackReplacement.MatchString(line) && !hasPS7Require {
			issues = append(issues, Issue{
				File:           name,
				Line:           lineNo,
				Rule:           "PSRegex003.DynamicCallbackRequiresPS7",
				Severity:       "warning",
				Message:        "scriptblock regex replacement requires PowerShell 7+",
				Recommendation: "Add '#requires -Version 7.0' to scripts that rely on dynamic regex callbacks.",
				Snippet:        strings.TrimSpace(line),
			})
		}
	}
	return issues
}

func LintFile(repo, rel string) ([]Issue, error) {
	if !IsPowerShellPath(rel) {
		return nil, nil
	}
	path := rel
	if !filepath.IsAbs(path) {
		path = platform.RepoPath(repo, rel)
	}
	content, err := platform.ReadText(path)
	if err != nil {
		return nil, err
	}
	name := strings.ReplaceAll(rel, "\\", "/")
	if filepath.IsAbs(rel) {
		name = rel
	}
	return LintText(name, content), nil
}

func LintPath(repo, rel string) ([]Issue, error) {
	if rel == "" {
		return nil, fmt.Errorf("path is required")
	}
	root := rel
	if !filepath.IsAbs(root) {
		root = platform.RepoPath(repo, rel)
	}
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		display := rel
		if filepath.IsAbs(rel) {
			display = root
		}
		return LintFile(repo, display)
	}

	issues := []Issue{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "bin" || base == ".pytest_cache" || base == "__pycache__" {
				return filepath.SkipDir
			}
			if isBadFixturePath(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !IsPowerShellPath(path) {
			return nil
		}
		display := path
		if r, e := filepath.Rel(repo, path); e == nil {
			display = filepath.ToSlash(r)
		}
		got, e := LintFile(repo, display)
		if e != nil {
			return e
		}
		issues = append(issues, got...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func LintStaged(repo string) ([]Issue, error) {
	files, err := gitx.StagedFiles(repo)
	if err != nil {
		return nil, err
	}
	issues := []Issue{}
	for _, rel := range files {
		if !IsPowerShellPath(rel) || isBadFixturePath(rel) {
			continue
		}
		got, err := LintFile(repo, rel)
		if err != nil {
			return issues, err
		}
		issues = append(issues, got...)
	}
	return issues, nil
}

func BlockingMessages(issues []Issue) []string {
	errs := []string{}
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, "error") {
			errs = append(errs, fmt.Sprintf("%s:%d: %s: %s", issue.File, issue.Line, issue.Rule, issue.Message))
		}
	}
	return errs
}
