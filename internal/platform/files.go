package platform

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func ResolveRepoRoot(repoArg string) (string, error) {
	if repoArg != "" {
		p, err := filepath.Abs(repoArg)
		if err != nil {
			return "", err
		}
		return p, nil
	}
	// Most CLI and hook calls start at the repository root. Avoid a redundant
	// Git process there; subdirectories and unusual layouts retain the existing
	// authoritative rev-parse fallback.
	if cwd, err := os.Getwd(); err == nil {
		dotGit := filepath.Join(cwd, ".git")
		if info, statErr := os.Stat(dotGit); statErr == nil {
			validRoot := info.IsDir() && IsFile(filepath.Join(dotGit, "HEAD"))
			if !info.IsDir() {
				_, commonErr := gitx.CommonDir(cwd)
				validRoot = commonErr == nil
			}
			if validRoot {
				return filepath.Abs(cwd)
			}
		}
	}
	stdout, err := gitx.Run("", "rev-parse", "--show-toplevel")
	if err != nil {
		return stdout, err
	}
	return strings.TrimSpace(stdout), nil
}

func RepoPath(repo, rel string) string {
	return filepath.Join(repo, filepath.FromSlash(rel))
}

func ReadText(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func IsDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
