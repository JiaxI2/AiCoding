package kit

import (
	"errors"
	"fmt"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type requiredPathResolution struct {
	Missing         []string
	EvidenceMissing bool
	RequiredAction  string
	Error           error
}

func resolveManifestRequiredPaths(repo string, manifest Manifest, paths []string) requiredPathResolution {
	resolution := requiredPathResolution{Missing: []string{}}
	for _, relative := range paths {
		if platform.Exists(platform.RepoPath(repo, relative)) {
			continue
		}
		if manifest.Source == nil {
			resolution.Missing = append(resolution.Missing, relative)
			continue
		}
		available, err := PinnedPathExists(repo, manifest.Source, relative)
		if err != nil {
			var cacheMiss *PinCacheMissError
			if errors.As(err, &cacheMiss) {
				resolution.EvidenceMissing = true
				resolution.RequiredAction = prefetchRequiredAction(manifest.ID)
				resolution.Missing = append(resolution.Missing, relative)
				continue
			}
			resolution.Error = fmt.Errorf("resolve pinned required path %s: %w", relative, err)
			resolution.Missing = append(resolution.Missing, relative)
			continue
		}
		if !available {
			resolution.Missing = append(resolution.Missing, relative)
		}
	}
	return resolution
}

func manifestPathAvailable(repo string, manifest Manifest, relative string) (bool, error) {
	if platform.IsFile(platform.RepoPath(repo, relative)) {
		return true, nil
	}
	if manifest.Source == nil {
		return false, nil
	}
	return PinnedPathExists(repo, manifest.Source, relative)
}
