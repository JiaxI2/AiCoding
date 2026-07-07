package cache

import (
	"os"
	"path/filepath"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

const relPath = ".aicoding/cache/fast-path-v2"

type StatusResult struct {
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	EntryCount int    `json:"entryCount"`
	SizeBytes  int64  `json:"sizeBytes"`
}

type CleanResult struct {
	Path       string `json:"path"`
	Removed    bool   `json:"removed"`
	EntryCount int    `json:"entryCount"`
	SizeBytes  int64  `json:"sizeBytes"`
}

func Status(repo string) (StatusResult, error) {
	return stat(repo)
}

func Clean(repo string) (CleanResult, error) {
	status, err := stat(repo)
	if err != nil {
		return CleanResult{Path: relPath}, err
	}
	result := CleanResult{Path: relPath, EntryCount: status.EntryCount, SizeBytes: status.SizeBytes}
	if status.Exists {
		if err := os.RemoveAll(platform.RepoPath(repo, relPath)); err != nil {
			return result, err
		}
		result.Removed = true
	}
	return result, nil
}

func stat(repo string) (StatusResult, error) {
	root := platform.RepoPath(repo, relPath)
	result := StatusResult{Path: filepath.ToSlash(relPath)}
	st, err := os.Stat(root)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Exists = st.IsDir()
	if !st.IsDir() {
		result.EntryCount = 1
		result.SizeBytes = st.Size()
		return result, nil
	}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		result.EntryCount++
		if info, err := d.Info(); err == nil && !info.IsDir() {
			result.SizeBytes += info.Size()
		}
		return nil
	})
	return result, err
}
