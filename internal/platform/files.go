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
