package lifecycle

import (
	"context"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
)

func Run(ctx context.Context, repo string, opts Options) Report {
	return run(ctx, repo, normalizeOptions(opts), defaultCommandExecutor)
}

func run(ctx context.Context, repo string, opts Options, execute commandExecutor) Report {
	result := Report{
		SchemaVersion: 1,
		Action:        opts.Action,
		Mode:          lifecycleMode(opts),
		Scope:         opts.Scope,
		DryRun:        opts.DryRun,
		OK:            true,
		Adapters:      []AdapterResult{},
	}
	for _, scope := range selectedScopes(opts.Scope) {
		var adapter AdapterResult
		switch scope {
		case ScopeKit:
			adapter = runKitAdapter(repo, opts)
		case ScopeMCP:
			adapter = runMCPAdapter(ctx, repo, opts)
		case ScopeRuntimeSkill:
			adapter = runRuntimeSkillAdapter(ctx, repo, opts, execute)
		default:
			adapter = failedAdapter(scope, opts, "unsupported lifecycle scope: "+scope)
		}
		appendAdapter(&result, adapter)
	}
	return result
}

func normalizeOptions(opts Options) Options {
	opts.Action = strings.ToLower(strings.TrimSpace(opts.Action))
	opts.Scope = strings.ToLower(strings.TrimSpace(opts.Scope))
	if opts.Scope == "" {
		opts.Scope = ScopeKit
	}
	if opts.VerifyProfile == "" {
		opts.VerifyProfile = "Smoke"
	}
	if opts.StandaloneRoot == "" {
		opts.StandaloneRoot = "agents"
	}
	return opts
}

func lifecycleMode(opts Options) string {
	if opts.DryRun {
		return "plan"
	}
	switch opts.Action {
	case "install", "update", "uninstall":
		return "apply"
	default:
		return opts.Action
	}
}

func selectedScopes(scope string) []string {
	if scope == ScopeAll {
		return []string{ScopeKit, ScopeMCP, ScopeRuntimeSkill}
	}
	return []string{scope}
}

func appendAdapter(report *Report, adapter AdapterResult) {
	report.Adapters = append(report.Adapters, adapter)
	report.Summary.Total++
	report.Summary.Warnings += len(adapter.Warnings)
	if adapter.OK {
		report.Summary.OK++
	} else {
		report.Summary.Failed++
		report.OK = false
	}
	for _, warning := range adapter.Warnings {
		report.Warnings = append(report.Warnings, adapter.ID+": "+warning)
	}
	for _, issue := range adapter.Errors {
		report.Errors = append(report.Errors, adapter.ID+": "+issue)
	}
	if !adapter.OK && len(adapter.Errors) == 0 {
		report.Errors = append(report.Errors, adapter.ID+": lifecycle adapter failed")
	}
}

func runKitAdapter(repo string, opts Options) AdapterResult {
	result := AdapterResult{ID: ScopeKit, Action: opts.Action, DryRun: opts.DryRun, OK: false, Status: "failed"}
	if opts.Action == "rollback" {
		actionReport := kit.RollbackLast(repo)
		result.OK = actionReport.OK
		result.Status = statusFromOK(actionReport.OK)
		result.Data = actionReport
		result.Warnings = actionReport.Warnings
		result.Errors = actionReport.Errors
		return result
	}

	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		result.Errors = []string{"cannot load kit registry: " + err.Error()}
		return result
	}
	selected, err := kit.SelectKits(entries, opts.KitID, opts.All || opts.Scope == ScopeAll)
	if err != nil {
		result.Errors = []string{err.Error()}
		return result
	}

	switch opts.Action {
	case "install", "update", "uninstall", "status":
		if opts.DryRun || opts.Action == "status" {
			plan := kit.PlanLifecycle(repo, selected, kit.LifecycleOptions{
				Action: opts.Action,
				Mode:   selectionMode(opts),
				DryRun: opts.DryRun,
			})
			result.OK = plan.OK
			result.Status = statusFromPlan(plan.OK, opts.DryRun)
			result.Data = plan
			result.Warnings = kitPlanWarnings(plan)
			result.Errors = kitPlanErrors(plan)
			return result
		}
		actionReport := kit.RunAction(repo, selected, kit.ActionOptions{
			Action: opts.Action,
			Mode:   selectionMode(opts),
			DryRun: false,
		})
		result.OK = actionReport.OK
		result.Status = statusFromOK(actionReport.OK)
		result.Data = actionReport
		result.Warnings = actionReport.Warnings
		result.Errors = actionReport.Errors
		return result
	case "doctor":
		errorsFound := kit.DoctorKits(repo, selected)
		result.OK = len(errorsFound) == 0
		result.Status = statusFromOK(result.OK)
		result.Data = kit.LoadKitViews(repo, selected)
		result.Errors = errorsFound
		return result
	case "verify":
		verification := kit.VerifyStructure(repo, selected)
		result.OK = verification.OK
		result.Status = statusFromOK(verification.OK)
		result.Data = verification
		result.Warnings = verification.Warnings
		result.Errors = verification.Errors
		return result
	default:
		result.Errors = []string{"unsupported kit lifecycle action: " + opts.Action}
		return result
	}
}

func runMCPAdapter(ctx context.Context, repo string, opts Options) AdapterResult {
	result := AdapterResult{ID: ScopeMCP, Action: opts.Action, DryRun: opts.DryRun, OK: false, Status: "failed"}
	entries, err := mcpcontrol.SelectComponents(repo, opts.ComponentID, opts.All || opts.Scope == ScopeAll)
	if err != nil {
		result.Errors = []string{err.Error()}
		return result
	}
	switch opts.Action {
	case "install", "update", "uninstall":
		results := mcpcontrol.RunLifecycle(repo, opts.CodexConfig, entries, opts.Action, opts.DryRun)
		errorsFound, warnings := mcpLifecycleMessages(results)
		result.OK = len(errorsFound) == 0
		result.Status = statusFromPlan(result.OK, opts.DryRun)
		result.Data = results
		result.Warnings = warnings
		result.Errors = errorsFound
		return result
	case "status":
		status := mcpcontrol.Status(repo, opts.CodexConfig, entries)
		errorsFound, warnings := mcpStatusMessages(status)
		result.OK = len(errorsFound) == 0
		result.Status = statusFromOK(result.OK)
		result.Data = status
		result.Warnings = warnings
		result.Errors = errorsFound
		return result
	case "doctor":
		doctor := mcpcontrol.DoctorComponents(repo, entries)
		errorsFound := mcpCommandErrors(entries, doctor)
		result.OK = len(errorsFound) == 0
		result.Status = statusFromOK(result.OK)
		result.Data = doctor
		result.Errors = errorsFound
		return result
	case "verify":
		verification := mcpcontrol.Verify(
			ctx,
			repo,
			opts.CodexConfig,
			entries,
			opts.VerifyProfile,
			opts.IncludeConfigured,
		)
		result.OK = verification.OK
		result.Status = statusFromOK(verification.OK)
		result.Data = verification
		result.Warnings = verification.Warnings
		result.Errors = verification.Errors
		return result
	default:
		result.Errors = []string{"unsupported MCP lifecycle action: " + opts.Action}
		return result
	}
}

func selectionMode(opts Options) string {
	if opts.All || opts.Scope == ScopeAll {
		return "all"
	}
	return "selected"
}

func kitPlanErrors(plan kit.LifecyclePlan) []string {
	errorsFound := []string{}
	for _, item := range plan.Kits {
		if item.OK {
			continue
		}
		reason := item.Reason
		if reason == "" {
			reason = item.Status
		}
		errorsFound = append(errorsFound, item.ID+": "+reason)
	}
	return errorsFound
}

func kitPlanWarnings(plan kit.LifecyclePlan) []string {
	warnings := []string{}
	for _, item := range plan.Kits {
		for _, warning := range item.Warnings {
			warnings = append(warnings, item.ID+": "+warning)
		}
	}
	return warnings
}

func mcpStatusMessages(results []mcpcontrol.StatusResult) ([]string, []string) {
	errorsFound := []string{}
	warnings := []string{}
	for _, result := range results {
		for _, issue := range result.Errors {
			errorsFound = append(errorsFound, result.ID+": "+issue)
		}
		for _, issue := range result.Warnings {
			warnings = append(warnings, result.ID+": "+issue)
		}
	}
	return errorsFound, warnings
}

func mcpCommandErrors(entries []mcpcontrol.RegistryEntry, results []mcpcontrol.CommandResult) []string {
	errorsFound := []string{}
	for index, result := range results {
		id := "component"
		if index < len(entries) {
			id = entries[index].ID
		}
		for _, issue := range result.Errors {
			errorsFound = append(errorsFound, id+": "+issue)
		}
	}
	return errorsFound
}

func mcpLifecycleMessages(results []mcpcontrol.LifecycleResult) ([]string, []string) {
	errorsFound := []string{}
	warnings := []string{}
	for _, result := range results {
		for _, issue := range result.Errors {
			errorsFound = append(errorsFound, result.ID+": "+issue)
		}
		for _, issue := range result.Warnings {
			warnings = append(warnings, result.ID+": "+issue)
		}
	}
	return errorsFound, warnings
}

func statusFromPlan(ok, dryRun bool) string {
	if !ok {
		return "failed"
	}
	if dryRun {
		return "planned"
	}
	return "ok"
}

func statusFromOK(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}

func failedAdapter(id string, opts Options, message string) AdapterResult {
	return AdapterResult{
		ID:     id,
		Action: opts.Action,
		DryRun: opts.DryRun,
		OK:     false,
		Status: "failed",
		Errors: []string{message},
	}
}
