package governance

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type dependencyWalkDir func(string, fs.WalkDirFunc) error

type dependencyInventory struct {
	repo        string
	files       []string
	directories map[string]bool
	entryCounts map[string]int
	contents    map[string][]byte
	readErrors  map[string]error
}

func buildDependencyInventory(repo string, policy dependencyPolicy, walk dependencyWalkDir) (*dependencyInventory, error) {
	roots := []string{}
	for _, boundary := range policy.GoPackageBoundaries {
		roots = append(roots, boundary.Path)
	}
	roots = append(roots, policy.GitProcessBoundary.ScanRoots...)
	roots = append(roots, policy.AcquisitionBoundary.ScanRoots...)
	for _, bindings := range [][]dependencyBinding{policy.KitRegistry.Bindings, policy.MCPRegistry.Bindings} {
		for _, binding := range bindings {
			roots = append(roots, binding.Roots...)
		}
	}
	return collectDependencyInventory(repo, roots, walk)
}

func collectDependencyInventory(repo string, roots []string, walk dependencyWalkDir) (*dependencyInventory, error) {
	normalizedRoots := []string{}
	seen := map[string]bool{}
	for _, root := range roots {
		normalized, err := normalizeDependencyPath(root)
		if err != nil || seen[normalized] {
			continue
		}
		seen[normalized] = true
		normalizedRoots = append(normalizedRoots, normalized)
	}
	sort.Strings(normalizedRoots)

	inventory := &dependencyInventory{
		repo:        repo,
		files:       []string{},
		directories: map[string]bool{},
		entryCounts: map[string]int{},
		contents:    map[string][]byte{},
		readErrors:  map[string]error{},
	}
	err := walk(repo, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "." {
			return nil
		}
		if entry.IsDir() && !dependencyInventoryPathRelevant(relative, normalizedRoots) {
			return filepath.SkipDir
		}
		if !dependencyInventoryPathRelevant(relative, normalizedRoots) {
			return nil
		}
		parent := filepath.ToSlash(filepath.Dir(relative))
		inventory.entryCounts[parent]++
		if entry.IsDir() {
			inventory.directories[relative] = true
			return nil
		}
		inventory.files = append(inventory.files, relative)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(inventory.files)
	return inventory, nil
}

func dependencyInventoryPathRelevant(path string, roots []string) bool {
	for _, root := range roots {
		if isLayoutWithin(path, root) || isLayoutWithin(root, path) {
			return true
		}
	}
	return false
}

func (inventory *dependencyInventory) hasDirectory(root string) bool {
	normalized, err := normalizeDependencyPath(root)
	return err == nil && inventory.directories[normalized]
}

func (inventory *dependencyInventory) hasFile(relative string) bool {
	normalized, err := normalizeDependencyPath(relative)
	if err != nil {
		return false
	}
	index := sort.SearchStrings(inventory.files, normalized)
	return index < len(inventory.files) && inventory.files[index] == normalized
}

func (inventory *dependencyInventory) filesWithin(root string, excluded map[string]bool) []string {
	normalized, err := normalizeDependencyPath(root)
	if err != nil {
		return nil
	}
	files := []string{}
	for _, relative := range inventory.files {
		if !isLayoutWithin(relative, normalized) || dependencyPathHasExcludedDirectory(relative, normalized, excluded, true) {
			continue
		}
		files = append(files, relative)
	}
	return files
}

func (inventory *dependencyInventory) directoriesWithin(root string, excluded map[string]bool) []string {
	normalized, err := normalizeDependencyPath(root)
	if err != nil {
		return nil
	}
	directories := []string{}
	for relative := range inventory.directories {
		if !isLayoutWithin(relative, normalized) || dependencyPathHasExcludedDirectory(relative, normalized, excluded, false) {
			continue
		}
		directories = append(directories, relative)
	}
	sort.Strings(directories)
	return directories
}

func dependencyPathHasExcludedDirectory(path, root string, excluded map[string]bool, file bool) bool {
	if len(excluded) == 0 {
		return false
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
	segments := strings.Split(relative, "/")
	for index, segment := range segments {
		if file && index == len(segments)-1 {
			break
		}
		if excluded[segment] || strings.HasSuffix(strings.ToLower(segment), ".egg-info") {
			return true
		}
	}
	return false
}

func (inventory *dependencyInventory) read(relative string) ([]byte, error) {
	if data, ok := inventory.contents[relative]; ok {
		return data, nil
	}
	if err, ok := inventory.readErrors[relative]; ok {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(inventory.repo, filepath.FromSlash(relative)))
	if err != nil {
		inventory.readErrors[relative] = err
		return nil, err
	}
	inventory.contents[relative] = data
	return data, nil
}
