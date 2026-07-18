package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type StructureReport struct {
	SchemaVersion  int                  `json:"schemaVersion"`
	OK             bool                 `json:"ok"`
	Summary        StructureSummary     `json:"summary"`
	Checks         []StructureCheck     `json:"checks"`
	Kits           []StructureKitResult `json:"kits"`
	LifecyclePlans []LifecyclePlan      `json:"lifecyclePlans"`
	Errors         []string             `json:"errors"`
	Warnings       []string             `json:"warnings"`
	ElapsedMS      int64                `json:"elapsedMs"`
}

type StructureSummary struct {
	Checks      int `json:"checks"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Kits        int `json:"kits"`
	EnabledKits int `json:"enabledKits"`
	Errors      int `json:"errors"`
	Warnings    int `json:"warnings"`
}

type StructureCheck struct {
	Name     string   `json:"name"`
	OK       bool     `json:"ok"`
	Status   string   `json:"status"`
	Message  string   `json:"message,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type StructureKitResult struct {
	ID       string                   `json:"id"`
	Enabled  bool                     `json:"enabled"`
	Manifest string                   `json:"manifest"`
	OK       bool                     `json:"ok"`
	Commands []StructureCommandResult `json:"commands,omitempty"`
	Errors   []string                 `json:"errors,omitempty"`
	Warnings []string                 `json:"warnings,omitempty"`
}

type StructureCommandResult struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Path           string   `json:"path,omitempty"`
	Executable     string   `json:"executable,omitempty"`
	SupportsDryRun bool     `json:"supportsDryRun"`
	Status         string   `json:"status"`
	Errors         []string `json:"errors,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

type codexKitConfig struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	CodingKitRoot string            `json:"codingKitRoot"`
	Agents        codexKitAgents    `json:"agents"`
	Assets        map[string]string `json:"assets"`
	Rules         map[string]bool   `json:"rules"`
}

type codexKitAgents struct {
	SkillsSubmodule string `json:"skillsSubmodule"`
	PluginPath      string `json:"pluginPath"`
	MarketplacePath string `json:"marketplacePath"`
}

type marketplaceConfig struct {
	Plugins []marketplacePlugin `json:"plugins"`
}

type marketplacePlugin struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source struct {
		Path string `json:"path"`
	} `json:"source"`
}

type skillShape struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Role string `json:"role"`
}

type structureVerifier struct {
	repo             string
	report           *StructureReport
	manifests        map[string]Manifest
	catalog          []ManifestSnapshot
	registryResolved bool
}

var allowedManifestCommandNames = map[string]bool{
	"doctor":        true,
	"export":        true,
	"install":       true,
	"skills":        true,
	"status":        true,
	"test":          true,
	"uninstall":     true,
	"update":        true,
	"verify":        true,
	"verify-skills": true,
}

var allowedManifestCommandTypes = map[string]bool{
	"builtin-check":     true,
	"builtin-lifecycle": true,
	"builtin-package":   true,
	"external-command":  true,
	"go-composed":       true,
	"specialty-pwsh":    true,
	"unsupported":       true,
}

func VerifyStructure(repo string, entries []RegistryKit) StructureReport {
	return verifyStructure(repo, entries, nil)
}

func VerifyCatalogStructure(repo string, snapshots []ManifestSnapshot) StructureReport {
	entries := make([]RegistryKit, 0, len(snapshots))
	for _, snapshot := range snapshots {
		entry := snapshot.Entry()
		entries = append(entries, entry)
	}
	return verifyStructure(repo, entries, snapshots)
}

func verifyStructure(repo string, entries []RegistryKit, catalog []ManifestSnapshot) StructureReport {
	start := time.Now()
	report := StructureReport{
		SchemaVersion:  1,
		OK:             true,
		Checks:         []StructureCheck{},
		Kits:           []StructureKitResult{},
		LifecyclePlans: []LifecyclePlan{},
		Errors:         []string{},
		Warnings:       []string{},
	}
	manifests := map[string]Manifest{}
	for _, snapshot := range catalog {
		manifest, err := snapshot.Manifest()
		if err == nil {
			manifests[snapshot.Entry().ID] = manifest
		}
	}
	verifier := structureVerifier{
		repo:             repo,
		report:           &report,
		manifests:        manifests,
		catalog:          cloneManifestSnapshots(catalog),
		registryResolved: catalog != nil,
	}
	verifier.checkCodexKitConfig()
	verifier.checkRegistry(entries)
	verifier.checkLifecyclePlans(entries)
	verifier.finish(start)
	return report
}

func (v structureVerifier) checkCodexKitConfig() {
	checkName := "codex-kit config"
	cfgPath := platform.RepoPath(v.repo, "config/codex-kit.json")
	if !platform.IsFile(cfgPath) {
		v.addCheck(checkName, false, "missing", "config/codex-kit.json is missing", []string{"missing config/codex-kit.json"}, nil)
		return
	}

	var cfg codexKitConfig
	if err := readJSON(cfgPath, &cfg); err != nil {
		v.addCheck(checkName, false, "invalid", "config/codex-kit.json is not valid JSON", []string{err.Error()}, nil)
		return
	}

	errs := []string{}
	warnings := []string{}
	if cfg.Name == "" {
		errs = append(errs, "codex-kit name is empty")
	}
	if cfg.Version == "" {
		errs = append(errs, "codex-kit version is empty")
	}
	if cfg.Agents.SkillsSubmodule == "" {
		errs = append(errs, "agents.skillsSubmodule is empty")
	} else {
		submoduleRel := cleanRel(cfg.Agents.SkillsSubmodule)
		submodulePath := platform.RepoPath(v.repo, submoduleRel)
		if !platform.IsDir(submodulePath) {
			warnings = append(warnings, "skills submodule directory is missing: "+submoduleRel)
		} else if dirty, err := gitStatusShort(submodulePath); err == nil && dirty != "" {
			errs = append(errs, "skills submodule has uncommitted changes")
		} else if err != nil {
			warnings = append(warnings, "skills submodule is present but git status was not available: "+err.Error())
		}
	}
	if cfg.Agents.PluginPath == "" {
		errs = append(errs, "agents.pluginPath is empty")
	} else if !platform.Exists(platform.RepoPath(v.repo, cleanRel(cfg.Agents.PluginPath))) {
		warnings = append(warnings, "generated plugin package is missing: "+cleanRel(cfg.Agents.PluginPath))
	}
	if cfg.Agents.MarketplacePath == "" {
		errs = append(errs, "agents.marketplacePath is empty")
	} else {
		warnings = append(warnings, v.checkMarketplace(cleanRel(cfg.Agents.MarketplacePath), cleanRel(cfg.Agents.PluginPath), &errs)...)
	}

	for _, key := range []string{"examples", "modules", "platforms", "tests", "tools"} {
		rel := cleanRel(cfg.Assets[key])
		if rel == "" {
			errs = append(errs, "assets."+key+" is empty")
			continue
		}
		if !platform.IsDir(platform.RepoPath(v.repo, rel)) {
			warnings = append(warnings, "asset path missing: "+rel)
		}
	}
	if cfg.Rules["buildPluginInSubmodule"] {
		errs = append(errs, "rules.buildPluginInSubmodule must remain false")
	}
	if !cfg.Rules["pluginInstallUsesMarketplace"] {
		errs = append(errs, "rules.pluginInstallUsesMarketplace must remain true")
	}
	if !cfg.Rules["hooksAreAuxiliaryConstraints"] {
		warnings = append(warnings, "rules.hooksAreAuxiliaryConstraints is not enabled")
	}
	warnings = append(warnings, v.checkPackagedSkillNames(cleanRel(cfg.Agents.PluginPath))...)

	v.addCheck(checkName, len(errs) == 0, statusFromErrors(errs, "ok"), "codex-kit config parsed", errs, warnings)
}

func (v structureVerifier) checkMarketplace(rel, pluginRel string, errs *[]string) []string {
	path := platform.RepoPath(v.repo, rel)
	if !platform.IsFile(path) {
		*errs = append(*errs, "marketplace file missing: "+rel)
		return nil
	}
	var marketplace marketplaceConfig
	if err := readJSON(path, &marketplace); err != nil {
		*errs = append(*errs, "marketplace JSON invalid: "+err.Error())
		return nil
	}
	for _, plugin := range marketplace.Plugins {
		if plugin.Name != "aicoding" {
			continue
		}
		sourcePath := cleanRel(plugin.Source.Path)
		if sourcePath == "" {
			sourcePath = cleanRel(plugin.Path)
		}
		if sourcePath == "" {
			*errs = append(*errs, "marketplace aicoding plugin path is empty")
			return nil
		}
		if sourcePath != pluginRel {
			*errs = append(*errs, fmt.Sprintf("marketplace aicoding plugin path mismatch: %s != %s", sourcePath, pluginRel))
		}
		return nil
	}
	*errs = append(*errs, "marketplace does not contain aicoding plugin")
	return nil
}

func (v structureVerifier) checkPackagedSkillNames(pluginRel string) []string {
	if pluginRel == "" {
		return nil
	}
	skillsDir := platform.RepoPath(v.repo, filepath.ToSlash(filepath.Join(filepath.FromSlash(pluginRel), "skills")))
	if !platform.IsDir(skillsDir) {
		return nil
	}
	warnings := []string{}
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return []string{"cannot read plugin skills directory: " + err.Error()}
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "obsidian-") {
			warnings = append(warnings, "obsidian standalone skill is packaged in AiCoding plugin: "+entry.Name())
		}
	}
	return warnings
}

func (v structureVerifier) checkRegistry(entries []RegistryKit) {
	if !v.registryResolved {
		registryPath := platform.RepoPath(v.repo, "config/kit-registry.json")
		if !platform.IsFile(registryPath) {
			v.addCheck("kit registry", false, "missing", "config/kit-registry.json is missing", []string{"missing config/kit-registry.json"}, nil)
			return
		}
		if _, err := LoadRegistry(v.repo); err != nil {
			v.addCheck("kit registry", false, "invalid", "kit registry cannot be loaded", []string{err.Error()}, nil)
			return
		}
	}

	errs := []string{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.ID == "" {
			errs = append(errs, "registry entry has empty id")
		}
		if entry.Manifest == "" {
			errs = append(errs, "registry entry "+entry.ID+" has empty manifest")
		}
		if seen[entry.ID] {
			errs = append(errs, "duplicate registry id: "+entry.ID)
		}
		seen[entry.ID] = true
	}
	v.addCheck("kit registry", len(errs) == 0, statusFromErrors(errs, "ok"), "kit registry parsed", errs, nil)
	for _, entry := range entries {
		v.checkManifest(entry)
	}
}

func (v structureVerifier) checkManifest(entry RegistryKit) {
	result := StructureKitResult{ID: entry.ID, Enabled: entry.Enabled, Manifest: entry.Manifest, OK: true}
	manifestPath := platform.RepoPath(v.repo, entry.Manifest)
	manifest, resolved := v.manifests[entry.ID]
	if entry.Manifest == "" || (!resolved && !platform.IsFile(manifestPath)) {
		result.Errors = append(result.Errors, "manifest file missing: "+entry.Manifest)
		v.addKitResult(result)
		return
	}
	if !resolved {
		var err error
		manifest, err = LoadManifest(v.repo, entry.Manifest)
		if err != nil {
			result.Errors = append(result.Errors, "cannot parse manifest: "+err.Error())
			v.addKitResult(result)
			return
		}
	}
	if manifest.ID != entry.ID {
		result.Errors = append(result.Errors, fmt.Sprintf("manifest id mismatch: registry %s != manifest %s", entry.ID, manifest.ID))
	}
	if manifest.SchemaVersion <= 0 {
		result.Errors = append(result.Errors, "manifest schemaVersion must be positive")
	}
	if len(manifest.Kind) == 0 {
		result.Errors = append(result.Errors, "manifest kind is empty")
	}
	if manifest.Mode == "" {
		result.Errors = append(result.Errors, "manifest mode is empty")
	} else if !allowedManifestModes[manifest.Mode] {
		result.Errors = append(result.Errors, "unsupported manifest mode: "+manifest.Mode)
	}
	if len(manifest.Commands) == 0 {
		result.Errors = append(result.Errors, "manifest commands are empty")
	}
	v.checkManifestCommands(manifest, &result)
	v.checkManifestSkills(manifest, &result)
	v.addKitResult(result)
}

func (v structureVerifier) checkManifestCommands(manifest Manifest, result *StructureKitResult) {
	names := make([]string, 0, len(manifest.Commands))
	for name := range manifest.Commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		command := manifest.Commands[name]
		commandResult := StructureCommandResult{Name: name, Type: command.Type, Path: command.Path, Executable: command.Executable, SupportsDryRun: command.SupportsDryRun, Status: "ok"}
		if !allowedManifestCommandNames[name] {
			commandResult.Errors = append(commandResult.Errors, "unsupported command action: "+name)
		}
		if command.Type == "" {
			commandResult.Errors = append(commandResult.Errors, "command type is empty")
		} else if !allowedManifestCommandTypes[command.Type] {
			commandResult.Errors = append(commandResult.Errors, "unsupported command type: "+command.Type)
		}
		switch command.Type {
		case "specialty-pwsh":
			if command.Path == "" {
				commandResult.Errors = append(commandResult.Errors, "specialty-pwsh path is empty")
			} else {
				rel := cleanRel(command.Path)
				commandResult.Path = rel
				if !platform.IsFile(platform.RepoPath(v.repo, rel)) {
					commandResult.Errors = append(commandResult.Errors, "specialty PowerShell script missing: "+rel)
				}
				if warning := specialtyPowerShellPathWarning(rel); warning != "" {
					commandResult.Warnings = append(commandResult.Warnings, warning)
				}
			}
		case "external-command":
			if command.Executable == "" {
				commandResult.Errors = append(commandResult.Errors, "external-command executable is empty")
			}
		case "builtin-check", "builtin-lifecycle":
			for _, rel := range missingRequiredPaths(v.repo, command.RequiredPaths) {
				commandResult.Errors = append(commandResult.Errors, "missing required path: "+rel)
			}
		case "go-composed":
			if len(command.Steps) == 0 {
				commandResult.Errors = append(commandResult.Errors, "go-composed command has no steps")
			}
			for _, step := range command.Steps {
				if _, ok := manifest.Commands[step]; !ok {
					commandResult.Errors = append(commandResult.Errors, "go-composed step not defined: "+step)
				}
			}
		case "unsupported":
			if command.Reason == "" {
				commandResult.Warnings = append(commandResult.Warnings, "unsupported command has empty reason")
			}
		case "builtin-package":
			// Package creation remains a PowerShell/full verification concern; Go only validates the command envelope.
		}
		if len(commandResult.Errors) > 0 {
			commandResult.Status = "failed"
			for _, err := range commandResult.Errors {
				result.Errors = append(result.Errors, "command "+name+": "+err)
			}
		}
		for _, warning := range commandResult.Warnings {
			result.Warnings = append(result.Warnings, "command "+name+": "+warning)
		}
		result.Commands = append(result.Commands, commandResult)
	}
}

func (v structureVerifier) checkManifestSkills(manifest Manifest, result *StructureKitResult) {
	if len(manifest.Skills) == 0 {
		return
	}
	if raw, ok := manifest.Skills["umbrella"]; ok {
		if isJSONNull(raw) {
			result.Errors = append(result.Errors, "skills.umbrella must not be null")
		} else {
			var umbrella skillShape
			if err := json.Unmarshal(raw, &umbrella); err != nil {
				result.Errors = append(result.Errors, "skills.umbrella is invalid: "+err.Error())
			} else {
				if umbrella.ID == "" {
					result.Errors = append(result.Errors, "skills.umbrella.id is empty")
				}
				if umbrella.Role != "router" {
					result.Errors = append(result.Errors, "skills.umbrella.role must be router")
				}
			}
		}
	}
	if raw, ok := manifest.Skills["members"]; ok {
		if isJSONNull(raw) {
			result.Errors = append(result.Errors, "skills.members must not be null")
			return
		}
		var members []skillShape
		if err := json.Unmarshal(raw, &members); err != nil {
			result.Errors = append(result.Errors, "skills.members is invalid: "+err.Error())
			return
		}
		for _, member := range members {
			if member.ID == "" {
				result.Errors = append(result.Errors, "skills.members.id is empty")
			}
			if member.Role != "subskill" {
				result.Errors = append(result.Errors, "skills.members.role must be subskill for "+member.ID)
			}
		}
	}
}

func (v structureVerifier) checkLifecyclePlans(entries []RegistryKit) {
	for _, action := range []string{"install", "update", "uninstall", "status"} {
		dryRun := action != "status"
		var plan LifecyclePlan
		if len(v.catalog) > 0 {
			plan = PlanCatalogLifecycle(v.repo, v.catalog, LifecycleOptions{Action: action, Mode: "all", DryRun: dryRun})
		} else {
			plan = PlanLifecycle(v.repo, entries, LifecycleOptions{Action: action, Mode: "all", DryRun: dryRun})
		}
		v.report.LifecyclePlans = append(v.report.LifecyclePlans, plan)
		errs := []string{}
		warnings := []string{}
		for _, item := range plan.Kits {
			for _, warning := range item.Warnings {
				warnings = append(warnings, item.ID+": "+warning)
			}
			if !item.OK {
				reason := item.Reason
				if reason == "" {
					reason = item.Status
				}
				errs = append(errs, item.ID+": "+reason)
			}
		}
		v.addCheck("lifecycle "+action+" plan", len(errs) == 0, statusFromErrors(errs, "ok"), "Go lifecycle planner structural policy", errs, warnings)
	}
}

func (v structureVerifier) addKitResult(result StructureKitResult) {
	result.OK = len(result.Errors) == 0
	if result.OK {
		v.report.Checks = append(v.report.Checks, StructureCheck{Name: "kit manifest " + result.ID, OK: true, Status: "ok", Message: "manifest parsed"})
	} else {
		v.report.Checks = append(v.report.Checks, StructureCheck{Name: "kit manifest " + result.ID, OK: false, Status: "failed", Message: "manifest validation failed", Errors: append([]string{}, result.Errors...)})
	}
	for _, err := range result.Errors {
		v.addReportError(result.ID + ": " + err)
	}
	for _, warning := range result.Warnings {
		v.addReportWarning(result.ID + ": " + warning)
	}
	v.report.Kits = append(v.report.Kits, result)
}

func (v structureVerifier) addCheck(name string, ok bool, status, message string, errs, warnings []string) {
	check := StructureCheck{Name: name, OK: ok, Status: status, Message: message, Errors: append([]string{}, errs...), Warnings: append([]string{}, warnings...)}
	v.report.Checks = append(v.report.Checks, check)
	for _, err := range errs {
		v.addReportError(name + ": " + err)
	}
	for _, warning := range warnings {
		v.addReportWarning(name + ": " + warning)
	}
}

func (v structureVerifier) addReportError(err string) {
	if err == "" || containsString(v.report.Errors, err) {
		return
	}
	v.report.Errors = append(v.report.Errors, err)
}

func (v structureVerifier) addReportWarning(warning string) {
	if warning == "" || containsString(v.report.Warnings, warning) {
		return
	}
	v.report.Warnings = append(v.report.Warnings, warning)
}

func (v structureVerifier) finish(start time.Time) {
	for _, check := range v.report.Checks {
		v.report.Summary.Checks++
		if check.OK {
			v.report.Summary.Passed++
		} else {
			v.report.Summary.Failed++
		}
	}
	v.report.Summary.Kits = len(v.report.Kits)
	for _, kit := range v.report.Kits {
		if kit.Enabled {
			v.report.Summary.EnabledKits++
		}
	}
	v.report.Summary.Errors = len(v.report.Errors)
	v.report.Summary.Warnings = len(v.report.Warnings)
	v.report.OK = len(v.report.Errors) == 0
	v.report.ElapsedMS = time.Since(start).Milliseconds()
}

func readJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func gitStatusShort(path string) (string, error) {
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return "", fmt.Errorf("not a git checkout")
	}
	stdout, err := gitx.Run(path, "status", "--short")
	if err != nil {
		return "", fmt.Errorf("git status --short failed: %w", err)
	}
	return strings.TrimSpace(stdout), nil
}

func cleanRel(rel string) string {
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "./")
	rel = filepath.ToSlash(rel)
	return rel
}

func statusFromErrors(errs []string, ok string) string {
	if len(errs) == 0 {
		return ok
	}
	return "failed"
}

func specialtyPowerShellPathWarning(rel string) string {
	return ""
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
