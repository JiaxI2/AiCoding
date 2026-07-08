package kit

import (
	"fmt"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type LifecycleOptions struct {
	Action string
	Mode   string
	DryRun bool
}

type LifecyclePlan struct {
	SchemaVersion int                `json:"schemaVersion"`
	Action        string             `json:"action"`
	Mode          string             `json:"mode"`
	DryRun        bool               `json:"dryRun"`
	OK            bool               `json:"ok"`
	Summary       LifecycleSummary   `json:"summary"`
	Kits          []LifecycleKitPlan `json:"kits"`
	ElapsedMS     int64              `json:"elapsedMs"`
}

type LifecycleSummary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
	Warnings int `json:"warnings"`
}

type LifecycleKitPlan struct {
	ID                   string   `json:"id"`
	Manifest             string   `json:"manifest"`
	Action               string   `json:"action"`
	DryRun               bool     `json:"dryRun"`
	Status               string   `json:"status"`
	OK                   bool     `json:"ok"`
	Skipped              bool     `json:"skipped"`
	Reason               string   `json:"reason,omitempty"`
	SupportsDryRun       bool     `json:"supportsDryRun"`
	CommandType          string   `json:"commandType,omitempty"`
	CommandPath          string   `json:"commandPath,omitempty"`
	RequiredPaths        []string `json:"requiredPaths"`
	MissingRequiredPaths []string `json:"missingRequiredPaths,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
	ElapsedMS            int64    `json:"elapsedMs"`
}

func PlanLifecycle(repo string, entries []RegistryKit, opts LifecycleOptions) LifecyclePlan {
	start := time.Now()
	action := strings.ToLower(opts.Action)
	plan := LifecyclePlan{
		SchemaVersion: 1,
		Action:        action,
		Mode:          opts.Mode,
		DryRun:        opts.DryRun,
		OK:            true,
		Kits:          []LifecycleKitPlan{},
	}
	for _, entry := range entries {
		result := planLifecycleKit(repo, entry, action, opts.DryRun)
		plan.Kits = append(plan.Kits, result)
		plan.Summary.Total++
		if result.OK {
			plan.Summary.OK++
		} else {
			plan.Summary.Failed++
			plan.OK = false
		}
		if result.Skipped {
			plan.Summary.Skipped++
		}
		plan.Summary.Warnings += len(result.Warnings)
	}
	plan.ElapsedMS = time.Since(start).Milliseconds()
	return plan
}

func planLifecycleKit(repo string, entry RegistryKit, action string, dryRun bool) LifecycleKitPlan {
	start := time.Now()
	result := LifecycleKitPlan{
		ID:       entry.ID,
		Manifest: entry.Manifest,
		Action:   action,
		DryRun:   dryRun,
		OK:       true,
		Status:   "planned",
	}
	manifest, err := LoadManifest(repo, entry.Manifest)
	if err != nil {
		result.OK = false
		result.Status = "failed"
		result.Reason = "cannot load manifest: " + err.Error()
		return finishLifecycleKitPlan(result, start)
	}
	command, ok := manifest.Commands[action]
	if !ok {
		result.Status = "skipped"
		result.Skipped = true
		result.Reason = "action not defined in manifest"
		result.Warnings = appendPluginPackageWarning(repo, manifest, result.Warnings)
		return finishLifecycleKitPlan(result, start)
	}

	result.SupportsDryRun = command.SupportsDryRun
	result.CommandType = command.Type
	result.CommandPath = command.Path
	if result.CommandPath == "" {
		result.CommandPath = command.Executable
	}
	result.RequiredPaths = append([]string{}, command.RequiredPaths...)

	if command.Type == "unsupported" {
		result.Status = "skipped"
		result.Skipped = true
		result.Reason = command.Reason
		if result.Reason == "" {
			result.Reason = "unsupported lifecycle action"
		}
		result.Warnings = appendPluginPackageWarning(repo, manifest, result.Warnings)
		return finishLifecycleKitPlan(result, start)
	}

	if dryRunLifecycleAction(action) && dryRun && !command.SupportsDryRun {
		result.Status = "skipped"
		result.Skipped = true
		result.Reason = "dry-run skipped command without supportsDryRun"
		result.Warnings = appendPluginPackageWarning(repo, manifest, result.Warnings)
		return finishLifecycleKitPlan(result, start)
	}

	switch command.Type {
	case "builtin-lifecycle":
		result.MissingRequiredPaths = missingRequiredPaths(repo, command.RequiredPaths)
		if len(result.MissingRequiredPaths) > 0 {
			result.OK = false
			result.Status = "missing"
			result.Reason = "missing required paths"
		} else if dryRun {
			result.Status = "planned"
			result.Reason = "Go builtin lifecycle dry-run"
		} else {
			result.Status = "static"
			result.Reason = "Go builtin lifecycle action"
		}
	case "builtin-check":
		result.MissingRequiredPaths = missingRequiredPaths(repo, command.RequiredPaths)
		if len(result.MissingRequiredPaths) > 0 {
			result.OK = false
			result.Status = "missing"
			result.Reason = "missing required paths"
		} else {
			result.Status = "ok"
			result.Reason = "required paths are present"
		}
	case "powershell-script":
		if command.Path == "" {
			result.OK = false
			result.Status = "failed"
			result.Reason = "powershell-script path is empty"
			break
		}
		if !platform.IsFile(platform.RepoPath(repo, command.Path)) {
			result.OK = false
			result.Status = "missing"
			result.Reason = "command script missing: " + command.Path
			break
		}
		if dryRun {
			result.Status = "planned"
			result.Reason = "dry-run plan; PowerShell script not executed"
		} else {
			result.Status = "static"
			result.Reason = "PowerShell status script present; not executed by Go planner"
		}
	case "external-command":
		result.Status = "static"
		result.Reason = "external command not executed by Go planner"
	case "composed":
		result.Status = "static"
		result.Reason = "composed command not executed by Go planner"
	default:
		result.OK = false
		result.Status = "failed"
		result.Reason = fmt.Sprintf("unsupported command type: %s", command.Type)
	}

	result.Warnings = appendPluginPackageWarning(repo, manifest, result.Warnings)
	return finishLifecycleKitPlan(result, start)
}

func finishLifecycleKitPlan(result LifecycleKitPlan, start time.Time) LifecycleKitPlan {
	result.ElapsedMS = time.Since(start).Milliseconds()
	return result
}

func dryRunLifecycleAction(action string) bool {
	switch action {
	case "install", "update", "uninstall":
		return true
	default:
		return false
	}
}

func missingRequiredPaths(repo string, paths []string) []string {
	missing := []string{}
	for _, rel := range paths {
		if !platform.Exists(platform.RepoPath(repo, rel)) {
			missing = append(missing, rel)
		}
	}
	return missing
}

func appendPluginPackageWarning(repo string, manifest Manifest, warnings []string) []string {
	if manifest.ID != "aicoding-platform" {
		return warnings
	}
	pluginRoot := manifest.Paths["pluginRoot"]
	if pluginRoot == "" {
		return warnings
	}
	if !platform.Exists(platform.RepoPath(repo, pluginRoot)) {
		warnings = append(warnings, "missing generated plugin package: "+pluginRoot)
	}
	return warnings
}
