package kit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	kittemplates "github.com/JiaxI2/AiCoding/config/templates/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

var kitInitIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

type InitOptions struct {
	External bool
	DryRun   bool
}

type InitReport struct {
	SchemaVersion int        `json:"schemaVersion"`
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	External      bool       `json:"external"`
	DryRun        bool       `json:"dryRun"`
	Enabled       bool       `json:"enabled"`
	Order         int        `json:"order"`
	OK            bool       `json:"ok"`
	Files         []InitFile `json:"files"`
	Errors        []string   `json:"errors,omitempty"`
}

type InitFile struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Digest string `json:"digest"`
	Bytes  int    `json:"bytes"`
}

type kitInitTemplateData struct {
	ID           string
	Name         string
	ManifestPath string
	BoundaryPath string
	WorkSpecPath string
	WorkSpecRoot string
}

type initPlannedFile struct {
	reportIndex int
	path        string
	content     []byte
	update      bool
	original    []byte
}

type kitInitDependencyPolicy struct {
	SchemaVersion        int                       `json:"schemaVersion"`
	Name                 string                    `json:"name"`
	Direction            string                    `json:"direction"`
	Layers               json.RawMessage           `json:"layers,omitempty"`
	ReservedNamespaces   json.RawMessage           `json:"reservedNamespaces,omitempty"`
	Scan                 json.RawMessage           `json:"scan,omitempty"`
	VersionVisibility    json.RawMessage           `json:"versionVisibility,omitempty"`
	KitRegistry          kitInitDependencyRegistry `json:"kitRegistry"`
	MCPRegistry          json.RawMessage           `json:"mcpRegistry,omitempty"`
	Skills               json.RawMessage           `json:"skills,omitempty"`
	ExternalDependencies json.RawMessage           `json:"externalDependencies,omitempty"`
	AcquisitionBoundary  json.RawMessage           `json:"acquisitionBoundary,omitempty"`
	GitProcessBoundary   json.RawMessage           `json:"gitProcessBoundary,omitempty"`
	GoPackageBoundaries  json.RawMessage           `json:"goPackageBoundaries,omitempty"`
}

type kitInitDependencyRegistry struct {
	Path         string            `json:"path"`
	IDPattern    string            `json:"idPattern,omitempty"`
	PromptPolicy string            `json:"promptPolicy,omitempty"`
	Bindings     []json.RawMessage `json:"bindings"`
}

type kitInitDependencyBinding struct {
	ID               string   `json:"id"`
	Layer            string   `json:"layer"`
	PlatformAgnostic bool     `json:"platformAgnostic"`
	Roots            []string `json:"roots"`
	DependsOn        []string `json:"dependsOn"`
}

func Init(repo, id string, opts InitOptions) (InitReport, error) {
	id = strings.TrimSpace(id)
	report := InitReport{
		SchemaVersion: 1,
		ID:            id,
		External:      opts.External,
		DryRun:        opts.DryRun,
		Enabled:       false,
		Files:         []InitFile{},
	}
	if !kitInitIDPattern.MatchString(id) {
		return failKitInit(report, fmt.Errorf("kit id must match ^[a-z0-9][a-z0-9-]*[a-z0-9]$: %s", id))
	}
	if strings.HasPrefix(id, "aicoding-") {
		return failKitInit(report, fmt.Errorf("kit id uses reserved aicoding- namespace: %s", id))
	}

	registryPath := "config/kit-registry.json"
	registryContent, err := os.ReadFile(platform.RepoPath(repo, registryPath))
	if err != nil {
		return failKitInit(report, fmt.Errorf("read %s: %w", registryPath, err))
	}
	var catalog registry
	if err := decodeStrictJSON(registryContent, &catalog); err != nil {
		return failKitInit(report, fmt.Errorf("parse %s: %w", registryPath, err))
	}
	if catalog.SchemaVersion != 1 || strings.TrimSpace(catalog.Name) == "" || strings.TrimSpace(catalog.DefaultMode) == "" {
		return failKitInit(report, fmt.Errorf("%s is missing its schemaVersion, name, or defaultMode", registryPath))
	}
	maxOrder := 0
	manifestPath := "config/kits/" + id + ".json"
	for _, entry := range catalog.Kits {
		if entry.ID == id {
			return failKitInit(report, fmt.Errorf("kit id is already registered and no files were changed: %s", id))
		}
		if entry.Manifest == manifestPath {
			return failKitInit(report, fmt.Errorf("kit manifest path is already registered and no files were changed: %s", manifestPath))
		}
		if entry.Order > maxOrder {
			maxOrder = entry.Order
		}
	}
	report.Order = maxOrder + 10
	report.Name = kitInitDisplayName(id)

	data := kitInitTemplateData{
		ID:           id,
		Name:         report.Name,
		ManifestPath: manifestPath,
		BoundaryPath: "docs/reference/kits/" + id + "-BOUNDARY.md",
		WorkSpecPath: "testdata/kits/" + id + "/workspec-example.json",
		WorkSpecRoot: "testdata/kits/" + id,
	}
	manifestTemplate := "manifest.tmpl.json"
	if opts.External {
		manifestTemplate = "manifest-external.tmpl.json"
	}
	manifestContent, err := renderKitInitTemplate(manifestTemplate, data)
	if err != nil {
		return failKitInit(report, err)
	}
	if err := validateKitInitManifest(manifestContent, data, opts.External); err != nil {
		return failKitInit(report, fmt.Errorf("validate %s: %w", manifestTemplate, err))
	}
	workSpecContent, err := renderKitInitTemplate("workspec-example.tmpl.json", data)
	if err != nil {
		return failKitInit(report, err)
	}
	var workSpec map[string]interface{}
	if err := decodeStrictJSON(workSpecContent, &workSpec); err != nil {
		return failKitInit(report, fmt.Errorf("validate workspec-example.tmpl.json: %w", err))
	}

	catalog.Kits = append(catalog.Kits, RegistryKit{
		ID: id, Enabled: false, Order: report.Order, Manifest: manifestPath,
	})
	updatedRegistry, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return failKitInit(report, fmt.Errorf("render %s: %w", registryPath, err))
	}
	updatedRegistry = append(updatedRegistry, '\n')
	dependencyPath := "config/dependency-governance.json"
	dependencyContent, err := os.ReadFile(platform.RepoPath(repo, dependencyPath))
	if err != nil {
		return failKitInit(report, fmt.Errorf("read %s: %w", dependencyPath, err))
	}
	updatedDependency, err := addKitInitDependencyBinding(dependencyContent, id)
	if err != nil {
		return failKitInit(report, fmt.Errorf("update %s: %w", dependencyPath, err))
	}

	planned := []initPlannedFile{
		{path: manifestPath, content: manifestContent},
		{path: data.WorkSpecPath, content: workSpecContent},
	}
	if opts.External {
		boundaryContent, renderErr := renderKitInitTemplate("boundary-card.tmpl.md", data)
		if renderErr != nil {
			return failKitInit(report, renderErr)
		}
		planned = append(planned, initPlannedFile{path: data.BoundaryPath, content: boundaryContent})
	}
	planned = append(planned,
		initPlannedFile{path: registryPath, content: updatedRegistry, update: true, original: registryContent},
		initPlannedFile{path: dependencyPath, content: updatedDependency, update: true, original: dependencyContent},
	)
	for index := range planned {
		planned[index].reportIndex = len(report.Files)
		action := "planned-create"
		if planned[index].update {
			action = "planned-update"
		}
		report.Files = append(report.Files, InitFile{
			Path: planned[index].path, Action: action,
			Digest: initContentDigest(planned[index].content), Bytes: len(planned[index].content),
		})
	}
	for _, file := range planned {
		if file.update {
			continue
		}
		if _, statErr := os.Lstat(platform.RepoPath(repo, file.path)); statErr == nil {
			return failKitInit(report, fmt.Errorf("target already exists and will not be overwritten: %s", file.path))
		} else if !os.IsNotExist(statErr) {
			return failKitInit(report, fmt.Errorf("inspect %s: %w", file.path, statErr))
		}
	}
	if opts.DryRun {
		report.OK = true
		return report, nil
	}

	created := []string{}
	for index := range planned {
		file := &planned[index]
		if file.update {
			continue
		}
		if err := writeKitInitNewFile(platform.RepoPath(repo, file.path), file.content); err != nil {
			rollbackKitInitFiles(repo, created)
			return failKitInit(report, fmt.Errorf("create %s: %w", file.path, err))
		}
		created = append(created, file.path)
		report.Files[file.reportIndex].Action = "created"
	}
	updated := []initPlannedFile{}
	for index := range planned {
		file := &planned[index]
		if !file.update {
			continue
		}
		if err := writeKitInitAtomic(platform.RepoPath(repo, file.path), file.content); err != nil {
			rollbackErr := rollbackKitInitAuthorities(repo, updated)
			rollbackKitInitFiles(repo, created)
			if rollbackErr != nil {
				return failKitInit(report, fmt.Errorf("update %s: %w; authority rollback failed: %v", file.path, err, rollbackErr))
			}
			return failKitInit(report, fmt.Errorf("update %s: %w", file.path, err))
		}
		updated = append(updated, *file)
		report.Files[file.reportIndex].Action = "updated"
	}
	report.OK = true
	return report, nil
}

func failKitInit(report InitReport, err error) (InitReport, error) {
	report.OK = false
	report.Errors = append(report.Errors, err.Error())
	return report, err
}

func kitInitDisplayName(id string) string {
	parts := strings.Split(id, "-")
	for index, part := range parts {
		if part != "" {
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func renderKitInitTemplate(name string, data kitInitTemplateData) ([]byte, error) {
	content, err := kittemplates.Files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read Kit init template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse Kit init template %s: %w", name, err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render Kit init template %s: %w", name, err)
	}
	return rendered.Bytes(), nil
}

func validateKitInitManifest(content []byte, data kitInitTemplateData, external bool) error {
	var manifest Manifest
	if err := decodeStrictJSON(content, &manifest); err != nil {
		return err
	}
	if manifest.SchemaVersion != 2 || manifest.ID != data.ID || strings.TrimSpace(manifest.Name) == "" || manifest.Version != "0.1.0" {
		return fmt.Errorf("manifest identity is incomplete")
	}
	if len(manifest.Kind) == 0 || !allowedManifestModes[manifest.Mode] {
		return fmt.Errorf("manifest kind or mode is invalid")
	}
	if strings.TrimSpace(manifest.Description) == "" || implementationLedDescription(manifest.Description) {
		return fmt.Errorf("manifest description must be user-result oriented")
	}
	verify, ok := manifest.Commands["verify"]
	if !ok || verify.Type != "builtin-check" || !containsString(verify.RequiredPaths, data.ManifestPath) {
		return fmt.Errorf("manifest must self-verify through a builtin-check")
	}
	if _, skillErrors := Skills(manifest); len(skillErrors) != 0 {
		return fmt.Errorf("manifest skills are invalid: %s", strings.Join(skillErrors, "; "))
	}
	thirdParty, _ := manifest.Trust["thirdParty"].(bool)
	updatePolicy, _ := manifest.Trust["updatePolicy"].(string)
	if thirdParty != external || !validKitUpdatePolicy(updatePolicy) {
		return fmt.Errorf("manifest trust does not match the scaffold variant")
	}
	if external && updatePolicy != "pinned" {
		return fmt.Errorf("external scaffold updatePolicy must be pinned")
	}
	for _, profile := range []string{"Smoke", "Full", "Release"} {
		if _, ok := manifest.Profiles[profile]; !ok {
			return fmt.Errorf("manifest profile is missing: %s", profile)
		}
	}
	return nil
}

func decodeStrictJSON(content []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func addKitInitDependencyBinding(content []byte, id string) ([]byte, error) {
	var policy kitInitDependencyPolicy
	if err := decodeStrictJSON(content, &policy); err != nil {
		return nil, err
	}
	if policy.SchemaVersion != 1 || strings.TrimSpace(policy.Name) == "" ||
		policy.Direction != "higher-rank-may-depend-on-equal-or-lower-rank" ||
		policy.KitRegistry.Path != "config/kit-registry.json" {
		return nil, fmt.Errorf("dependency governance identity is incomplete")
	}
	for _, raw := range policy.KitRegistry.Bindings {
		var binding struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &binding); err != nil {
			return nil, fmt.Errorf("parse existing Kit binding: %w", err)
		}
		if binding.ID == id {
			return nil, fmt.Errorf("kit dependency binding already exists: %s", id)
		}
	}
	binding, err := json.Marshal(kitInitDependencyBinding{
		ID: id, Layer: "capability", PlatformAgnostic: true,
		Roots: []string{}, DependsOn: []string{},
	})
	if err != nil {
		return nil, err
	}
	policy.KitRegistry.Bindings = append(policy.KitRegistry.Bindings, binding)
	updated, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(updated, '\n'), nil
}

func initContentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeKitInitNewFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func writeKitInitAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".kit-init-*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o644); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func rollbackKitInitFiles(repo string, paths []string) {
	for index := len(paths) - 1; index >= 0; index-- {
		_ = os.Remove(platform.RepoPath(repo, paths[index]))
	}
}

func rollbackKitInitAuthorities(repo string, files []initPlannedFile) error {
	var firstErr error
	for index := len(files) - 1; index >= 0; index-- {
		if err := writeKitInitAtomic(platform.RepoPath(repo, files[index].path), files[index].original); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("restore %s: %w", files[index].path, err)
		}
	}
	return firstErr
}
