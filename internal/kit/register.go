package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type RegisterReport struct {
	SchemaVersion  int           `json:"schemaVersion"`
	OK             bool          `json:"ok"`
	ID             string        `json:"id"`
	Manifest       string        `json:"manifest"`
	Enabled        bool          `json:"enabled"`
	Order          int           `json:"order"`
	Source         *PinnedSource `json:"source,omitempty"`
	SourceIdentity string        `json:"sourceIdentity,omitempty"`
	Files          []InitFile    `json:"files"`
	Errors         []string      `json:"errors,omitempty"`
}

func Register(repo, manifestPath string) (RegisterReport, error) {
	report := RegisterReport{SchemaVersion: 1, Enabled: true, Files: []InitFile{}}
	relative, err := repositoryManifestPath(repo, manifestPath)
	if err != nil {
		return failRegister(report, err)
	}
	report.Manifest = relative
	manifest, err := LoadManifest(repo, relative)
	if err != nil {
		return failRegister(report, fmt.Errorf("parse pinned Kit manifest: %w", err))
	}
	report.ID = manifest.ID
	report.Source = clonePinnedSource(manifest.Source)
	if manifest.SchemaVersion != 2 || !kitInitIDPattern.MatchString(manifest.ID) || strings.TrimSpace(manifest.Name) == "" ||
		strings.TrimSpace(manifest.Version) == "" || len(manifest.Kind) == 0 || !allowedManifestModes[manifest.Mode] || len(manifest.Commands) == 0 {
		return failRegister(report, fmt.Errorf("pinned Kit manifest identity or command shape is incomplete"))
	}
	if err := ValidatePinnedSource(manifest.Source); err != nil {
		return failRegister(report, err)
	}
	report.SourceIdentity, err = PinnedSourceIdentity(manifest.Source)
	if err != nil {
		return failRegister(report, err)
	}
	thirdParty, _ := manifest.Trust["thirdParty"].(bool)
	updatePolicy, _ := manifest.Trust["updatePolicy"].(string)
	if !thirdParty || updatePolicy != "pinned" {
		return failRegister(report, fmt.Errorf("pinned external Kit requires trust.thirdParty=true and trust.updatePolicy=pinned"))
	}

	registryPath := "config/kit-registry.json"
	registryContent, err := os.ReadFile(platform.RepoPath(repo, registryPath))
	if err != nil {
		return failRegister(report, fmt.Errorf("read %s: %w", registryPath, err))
	}
	var catalog registry
	if err := decodeStrictJSON(registryContent, &catalog); err != nil {
		return failRegister(report, fmt.Errorf("parse %s: %w", registryPath, err))
	}
	if catalog.SchemaVersion != 1 || strings.TrimSpace(catalog.Name) == "" || strings.TrimSpace(catalog.DefaultMode) == "" {
		return failRegister(report, fmt.Errorf("%s identity is incomplete", registryPath))
	}
	maxOrder := 0
	for _, entry := range catalog.Kits {
		if entry.ID == manifest.ID {
			return failRegister(report, fmt.Errorf("kit id is already registered and no files were changed: %s", manifest.ID))
		}
		if filepath.ToSlash(entry.Manifest) == relative {
			return failRegister(report, fmt.Errorf("kit manifest path is already registered and no files were changed: %s", relative))
		}
		if entry.Order > maxOrder {
			maxOrder = entry.Order
		}
	}
	report.Order = maxOrder + 10
	catalog.Kits = append(catalog.Kits, RegistryKit{ID: manifest.ID, Enabled: true, Order: report.Order, Manifest: relative})
	updatedRegistry, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return failRegister(report, fmt.Errorf("render %s: %w", registryPath, err))
	}
	updatedRegistry = append(updatedRegistry, '\n')

	dependencyPath := "config/dependency-governance.json"
	dependencyContent, err := os.ReadFile(platform.RepoPath(repo, dependencyPath))
	if err != nil {
		return failRegister(report, fmt.Errorf("read %s: %w", dependencyPath, err))
	}
	updatedDependency, err := addKitInitDependencyBinding(dependencyContent, manifest.ID)
	if err != nil {
		return failRegister(report, fmt.Errorf("update %s: %w", dependencyPath, err))
	}
	report.Files = []InitFile{
		{Path: registryPath, Action: "updated", Digest: initContentDigest(updatedRegistry), Bytes: len(updatedRegistry)},
		{Path: dependencyPath, Action: "updated", Digest: initContentDigest(updatedDependency), Bytes: len(updatedDependency)},
	}
	if err := writeKitInitAtomic(platform.RepoPath(repo, registryPath), updatedRegistry); err != nil {
		return failRegister(report, fmt.Errorf("write %s: %w", registryPath, err))
	}
	if err := writeKitInitAtomic(platform.RepoPath(repo, dependencyPath), updatedDependency); err != nil {
		rollbackErr := writeKitInitAtomic(platform.RepoPath(repo, registryPath), registryContent)
		if rollbackErr != nil {
			return failRegister(report, fmt.Errorf("write %s: %w; registry rollback failed: %v", dependencyPath, err, rollbackErr))
		}
		return failRegister(report, fmt.Errorf("write %s: %w", dependencyPath, err))
	}
	report.OK = true
	return report, nil
}

func PrefetchRegisteredKit(ctx context.Context, repo, id string) (PinStatus, error) {
	entries, err := LoadRegistry(repo)
	if err != nil {
		return PinStatus{}, err
	}
	for _, entry := range entries {
		if entry.ID != id {
			continue
		}
		manifest, err := LoadManifest(repo, entry.Manifest)
		if err != nil {
			return PinStatus{}, err
		}
		if manifest.Source == nil {
			return PinStatus{}, fmt.Errorf("registered Kit %s has no content-pinned source", id)
		}
		return PrefetchPin(ctx, repo, id, manifest.Source)
	}
	return PinStatus{}, fmt.Errorf("registered Kit not found: %s", id)
}

func repositoryManifestPath(repo, manifestPath string) (string, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		return "", fmt.Errorf("--manifest is required")
	}
	repositoryRoot, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	repositoryRoot, err = filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	absolute := manifestPath
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(repositoryRoot, filepath.FromSlash(manifestPath))
	}
	absolute, err = filepath.Abs(absolute)
	if err != nil {
		return "", err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve manifest path: %w", err)
	}
	relative, err := filepath.Rel(repositoryRoot, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("manifest must be an existing file inside the repository")
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("manifest must be an existing regular file inside the repository")
	}
	return filepath.ToSlash(relative), nil
}

func failRegister(report RegisterReport, err error) (RegisterReport, error) {
	report.OK = false
	report.Errors = append(report.Errors, err.Error())
	return report, err
}
