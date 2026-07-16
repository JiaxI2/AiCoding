package kit

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type ActionOptions struct {
	Action string
	Mode   string
	DryRun bool
}

type ActionReport struct {
	SchemaVersion int            `json:"schemaVersion"`
	Action        string         `json:"action"`
	Mode          string         `json:"mode"`
	DryRun        bool           `json:"dryRun"`
	OK            bool           `json:"ok"`
	Summary       ActionSummary  `json:"summary"`
	Kits          []ActionResult `json:"kits"`
	Warnings      []string       `json:"warnings,omitempty"`
	Errors        []string       `json:"errors,omitempty"`
	RollbackFile  string         `json:"rollbackFile,omitempty"`
}

type ActionSummary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
	Warnings int `json:"warnings"`
}

type ActionResult struct {
	ID       string      `json:"id"`
	Action   string      `json:"action"`
	OK       bool        `json:"ok"`
	Status   string      `json:"status"`
	Message  string      `json:"message,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Warnings []string    `json:"warnings,omitempty"`
	Errors   []string    `json:"errors,omitempty"`
}

type installState struct {
	SchemaVersion      int       `json:"schemaVersion"`
	KitID              string    `json:"kitId"`
	Version            string    `json:"version"`
	Manifest           string    `json:"manifest"`
	Action             string    `json:"action"`
	InstalledAt        time.Time `json:"installedAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	PluginSourceCommit string    `json:"pluginSourceCommit,omitempty"`
	PluginSkillsDigest string    `json:"pluginSkillsDigest,omitempty"`
	PluginCachePath    string    `json:"pluginCachePath,omitempty"`
}

type rollbackSnapshot struct {
	SchemaVersion int                      `json:"schemaVersion"`
	CreatedAt     time.Time                `json:"createdAt"`
	Action        string                   `json:"action"`
	States        map[string]*installState `json:"states"`
}

func RunAction(repo string, entries []RegistryKit, opts ActionOptions) ActionReport {
	action := strings.ToLower(strings.TrimSpace(opts.Action))
	report := ActionReport{SchemaVersion: 1, Action: action, Mode: opts.Mode, DryRun: opts.DryRun, OK: true, Kits: []ActionResult{}}
	if action == "" {
		report.OK = false
		report.Errors = []string{"missing action"}
		return report
	}
	if !opts.DryRun && (action == "install" || action == "update" || action == "uninstall") {
		file, err := saveRollbackSnapshot(repo, entries, action)
		if err != nil {
			report.OK = false
			report.Errors = append(report.Errors, "cannot save rollback snapshot: "+err.Error())
			return report
		}
		report.RollbackFile = file
	}
	for _, entry := range entries {
		result := runActionForKit(repo, entry, action, opts.DryRun)
		report.Kits = append(report.Kits, result)
		report.Summary.Total++
		if result.OK {
			report.Summary.OK++
		} else {
			report.Summary.Failed++
			report.OK = false
		}
		if result.Status == "skipped" || result.Status == "unsupported" {
			report.Summary.Skipped++
		}
		report.Summary.Warnings += len(result.Warnings)
		for _, w := range result.Warnings {
			report.Warnings = append(report.Warnings, entry.ID+": "+w)
		}
		for _, e := range result.Errors {
			report.Errors = append(report.Errors, entry.ID+": "+e)
		}
	}
	return report
}

func runActionForKit(repo string, entry RegistryKit, action string, dryRun bool) ActionResult {
	manifest, err := LoadManifest(repo, entry.Manifest)
	if err != nil {
		return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Errors: []string{"cannot load manifest: " + err.Error()}}
	}
	command, ok := manifest.Commands[action]
	if !ok {
		return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "skipped", Message: "action not defined"}
	}
	switch command.Type {
	case "builtin-lifecycle":
		return runBuiltinLifecycle(repo, entry, manifest, command, action, dryRun)
	case "builtin-check":
		missing := missingRequiredPaths(repo, command.RequiredPaths)
		return ActionResult{ID: entry.ID, Action: action, OK: len(missing) == 0, Status: statusFromMissing(missing), Data: map[string]interface{}{"requiredPaths": command.RequiredPaths, "missing": missing}, Errors: prefixMissing(missing)}
	case "builtin-package":
		if action != "export" {
			return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "skipped", Message: "package command is export-only"}
		}
		pkg, err := ExportKit(repo, entry, manifest, command, ExportOptions{Zip: true, DryRun: dryRun})
		if err != nil {
			return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Errors: []string{err.Error()}, Data: pkg}
		}
		return ActionResult{ID: entry.ID, Action: action, OK: true, Status: pkg.Status, Data: pkg}
	case "go-composed":
		results := []ActionResult{}
		ok := true
		for _, step := range command.Steps {
			stepResult := runActionForKit(repo, entry, step, dryRun)
			results = append(results, stepResult)
			if !stepResult.OK {
				ok = false
			}
		}
		return ActionResult{ID: entry.ID, Action: action, OK: ok, Status: statusFromOK(ok), Data: map[string]interface{}{"steps": results}, Errors: actionErrors(results)}
	case "external-command":
		if dryRun {
			return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "planned", Message: "external command dry-run", Data: map[string]interface{}{"executable": command.Executable, "args": command.Args}}
		}
		return runExternal(repo, entry.ID, action, command)
	case "unsupported":
		return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "skipped", Message: command.Reason, Warnings: []string{command.Reason}}
	case "specialty-pwsh":
		return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "skipped", Message: "specialty PowerShell command is explicit and not executed by Go default", Warnings: []string{command.Path}}
	default:
		return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Errors: []string{"unsupported command type: " + command.Type}}
	}
}

func runBuiltinLifecycle(repo string, entry RegistryKit, manifest Manifest, command CommandDef, action string, dryRun bool) ActionResult {
	missing := missingRequiredPaths(repo, command.RequiredPaths)
	if len(missing) > 0 {
		return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "missing", Data: map[string]interface{}{"missing": missing}, Errors: prefixMissing(missing)}
	}
	if dryRun {
		result := ActionResult{ID: entry.ID, Action: action, OK: true, Status: "planned", Message: "builtin lifecycle dry-run"}
		if entry.ID == "aicoding-platform" && (action == "install" || action == "update") {
			pluginSync, err := inspectPlatformPlugin(repo, manifest)
			if err != nil {
				result.OK = false
				result.Status = "failed"
				result.Errors = []string{err.Error()}
			} else {
				result.Data = pluginSync
				if pluginSync.Drift {
					result.Warnings = append(result.Warnings, "installed plugin cache drift: "+strings.Join(pluginSync.DriftReasons, "; "))
				}
			}
		}
		return result
	}
	switch action {
	case "install", "update":
		var data interface{}
		var syncedPlugin *PlatformPluginSync
		warnings := []string{}
		if entry.ID == "aicoding-platform" {
			pluginSync, err := syncPlatformPlugin(repo, manifest)
			data = pluginSync
			if err != nil {
				status := "failed"
				if pluginSync.ManualRequired {
					status = "manual-required"
				}
				return ActionResult{ID: entry.ID, Action: action, OK: false, Status: status, Data: pluginSync, Errors: []string{err.Error()}}
			}
			repoWarnings, err := configurePlatformRepository(repo, &pluginSync)
			data = pluginSync
			syncedPlugin = &pluginSync
			warnings = append(warnings, repoWarnings...)
			if err != nil {
				return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Data: pluginSync, Warnings: warnings, Errors: []string{err.Error()}}
			}
		}
		if err := writeInstallState(repo, entry, manifest, action, syncedPlugin); err != nil {
			return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Errors: []string{err.Error()}}
		}
		return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "ok", Message: "builtin lifecycle state updated after runtime synchronization", Data: data, Warnings: warnings}
	case "uninstall":
		if err := removeInstallState(repo, manifest); err != nil {
			return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Errors: []string{err.Error()}}
		}
		return ActionResult{ID: entry.ID, Action: action, OK: true, Status: "ok", Message: "builtin lifecycle state removed"}
	default:
		return ActionResult{ID: entry.ID, Action: action, OK: false, Status: "failed", Errors: []string{"unsupported builtin lifecycle action: " + action}}
	}
}

func runExternal(repo, id, action string, command CommandDef) ActionResult {
	cmd := exec.Command(command.Executable, command.Args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ActionResult{ID: id, Action: action, OK: false, Status: "failed", Data: map[string]string{"output": strings.TrimSpace(string(out))}, Errors: []string{err.Error()}}
	}
	return ActionResult{ID: id, Action: action, OK: true, Status: "ok", Data: map[string]string{"output": strings.TrimSpace(string(out))}}
}

func writeInstallState(repo string, entry RegistryKit, manifest Manifest, action string, pluginSync *PlatformPluginSync) error {
	path := statePath(repo, manifest, entry.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	now := time.Now().UTC()
	state := installState{SchemaVersion: 1, KitID: entry.ID, Version: manifest.Version, Manifest: entry.Manifest, Action: action, InstalledAt: now, UpdatedAt: now}
	if previousState, err := readInstallState(path); err == nil && previousState != nil {
		state.InstalledAt = previousState.InstalledAt
	}
	if pluginSync != nil {
		state.PluginSourceCommit = pluginSync.SourceBuildInfo.SourceCommit
		state.PluginSkillsDigest = pluginSync.SourceBuildInfo.SkillsDigest
		state.PluginCachePath = pluginSync.InstalledPackage
	}
	return writeJSONFile(path, state)
}

func removeInstallState(repo string, manifest Manifest) error {
	path := statePath(repo, manifest, manifest.ID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func statePath(repo string, manifest Manifest, fallbackID string) string {
	if rel := manifest.State["installState"]; rel != "" {
		return platform.RepoPath(repo, rel)
	}
	id := manifest.ID
	if id == "" {
		id = fallbackID
	}
	return platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "state", "kits", id, "install-state.json")))
}

func readInstallState(path string) (*installState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state installState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveRollbackSnapshot(repo string, entries []RegistryKit, action string) (string, error) {
	snap := rollbackSnapshot{SchemaVersion: 1, CreatedAt: time.Now().UTC(), Action: action, States: map[string]*installState{}}
	for _, entry := range entries {
		manifest, err := LoadManifest(repo, entry.Manifest)
		if err != nil {
			continue
		}
		path := statePath(repo, manifest, entry.ID)
		state, err := readInstallState(path)
		if err == nil {
			snap.States[entry.ID] = state
		} else {
			snap.States[entry.ID] = nil
		}
	}
	file := platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "state", "rollback", "last.json")))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	return file, writeJSONFile(file, snap)
}

func RollbackLast(repo string) ActionReport {
	file := platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "state", "rollback", "last.json")))
	report := ActionReport{SchemaVersion: 1, Action: "rollback", Mode: "last", OK: true, RollbackFile: file}
	b, err := os.ReadFile(file)
	if err != nil {
		report.OK = false
		report.Errors = []string{err.Error()}
		return report
	}
	var snap rollbackSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		report.OK = false
		report.Errors = []string{err.Error()}
		return report
	}
	entries, err := LoadRegistry(repo)
	if err != nil {
		report.OK = false
		report.Errors = []string{err.Error()}
		return report
	}
	byID := map[string]RegistryKit{}
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	for id, state := range snap.States {
		entry := byID[id]
		manifest, err := LoadManifest(repo, entry.Manifest)
		if err != nil {
			report.Kits = append(report.Kits, ActionResult{ID: id, Action: "rollback", OK: false, Status: "failed", Errors: []string{err.Error()}})
			report.OK = false
			continue
		}
		path := statePath(repo, manifest, id)
		if state == nil {
			_ = os.Remove(path)
			report.Kits = append(report.Kits, ActionResult{ID: id, Action: "rollback", OK: true, Status: "ok", Message: "state removed"})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			report.Kits = append(report.Kits, ActionResult{ID: id, Action: "rollback", OK: false, Status: "failed", Errors: []string{err.Error()}})
			report.OK = false
			continue
		}
		if err := writeJSONFile(path, state); err != nil {
			report.Kits = append(report.Kits, ActionResult{ID: id, Action: "rollback", OK: false, Status: "failed", Errors: []string{err.Error()}})
			report.OK = false
			continue
		}
		report.Kits = append(report.Kits, ActionResult{ID: id, Action: "rollback", OK: true, Status: "ok", Message: "state restored"})
	}
	for _, r := range report.Kits {
		report.Summary.Total++
		if r.OK {
			report.Summary.OK++
		} else {
			report.Summary.Failed++
		}
	}
	return report
}

func writeJSONFile(path string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func statusFromMissing(missing []string) string {
	if len(missing) == 0 {
		return "ok"
	}
	return "missing"
}
func statusFromOK(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}
func prefixMissing(missing []string) []string {
	errs := []string{}
	for _, m := range missing {
		errs = append(errs, "missing required path: "+m)
	}
	return errs
}
func actionErrors(results []ActionResult) []string {
	errs := []string{}
	for _, r := range results {
		if !r.OK {
			errs = append(errs, r.Errors...)
		}
	}
	return errs
}
