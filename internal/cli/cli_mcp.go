package cli

import (
	"context"
	"strings"
	"time"

	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func runMCP(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("mcp requires subcommand: list, status, doctor, verify, install, update, or uninstall")
	}
	sub := strings.ToLower(args[0])
	if !validChoice(sub, "list", "status", "doctor", "verify", "install", "update", "uninstall") {
		return report.Result{}, usageErrorf("unsupported mcp subcommand: %s", sub)
	}
	component := ""
	flagArgs := args[1:]
	if len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "-") {
		component = flagArgs[0]
		flagArgs = flagArgs[1:]
	}
	fs := newFlagSet("mcp " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	componentArg := fs.String("component", component, "MCP component id")
	allArg := fs.Bool("all", false, "all enabled managed MCP components")
	profileArg := fs.String("profile", "Smoke", "Smoke, Full or Release")
	codexConfigArg := fs.String("codex-config", "", "Codex config.toml path")
	configuredArg := fs.Bool("configured", false, "include currently configured Codex MCP compatibility probes")
	dryRunArg := fs.Bool("dry-run", false, "plan lifecycle changes without writing")
	_ = fs.Bool("json", false, "JSON output")
	if err := parseNoPositionals(fs, flagArgs); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("mcp "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	if sub == "list" {
		inventory, listErr := mcpcontrol.ListInventory(repo, *codexConfigArg)
		if listErr != nil {
			return report.Fail("mcp list", start, "cannot load MCP inventory", nil, listErr.Error()), listErr
		}
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp list",
			OK:            true,
			Message:       "MCP component and Codex configuration inventory",
			RepoRoot:      repo,
			InputDigest:   inventory.CatalogDigest,
			Data:          inventory,
			Warnings:      inventory.Warnings,
			ElapsedMS:     report.Elapsed(start),
		}, nil
	}

	catalog, err := mcpcontrol.LoadCatalogSnapshot(repo)
	if err != nil {
		return report.Fail("mcp "+sub, start, "cannot load MCP catalog", nil, err.Error()), err
	}
	components, err := catalog.Select(*componentArg, *allArg)
	if err != nil {
		return report.Fail("mcp "+sub, start, "MCP component selection failed", nil, err.Error()), err
	}
	entries := componentSnapshotEntries(components)
	switch sub {
	case "status":
		status := mcpcontrol.StatusCatalog(repo, *codexConfigArg, components)
		errorsFound, warnings := statusMessages(status)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp status",
			OK:            len(errorsFound) == 0,
			Message:       "MCP component status",
			RepoRoot:      repo,
			InputDigest:   catalog.Digest(),
			Data:          status,
			Warnings:      warnings,
			Errors:        errorsFound,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(errorsFound)
	case "doctor":
		doctor := mcpcontrol.DoctorCatalogComponentsContext(context.Background(), repo, components)
		errorsFound := commandErrors(entries, doctor)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp doctor",
			OK:            len(errorsFound) == 0,
			Message:       "MCP managed component doctor",
			RepoRoot:      repo,
			InputDigest:   catalog.Digest(),
			Data:          doctor,
			Errors:        errorsFound,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(errorsFound)
	case "verify":
		includeConfigured := *configuredArg || *allArg
		verification := mcpcontrol.VerifyCatalog(
			context.Background(),
			repo,
			*codexConfigArg,
			components,
			*profileArg,
			includeConfigured,
		)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp verify",
			OK:            verification.OK,
			Message:       "MCP managed and configured compatibility verification",
			RepoRoot:      repo,
			InputDigest:   catalog.Digest(),
			Data:          verification,
			Warnings:      verification.Warnings,
			Errors:        verification.Errors,
			ElapsedMS:     report.Elapsed(start),
		}, mcpcontrol.VerifyErrors(verification)
	case "install", "update", "uninstall":
		unified := lifecyclecontrol.Run(context.Background(), repo, lifecyclecontrol.Options{
			Action:      sub,
			Scope:       lifecyclecontrol.ScopeMCP,
			All:         *allArg,
			ComponentID: *componentArg,
			CodexConfig: *codexConfigArg,
			DryRun:      *dryRunArg,
		})
		data := lifecycleAdapterData(unified, lifecyclecontrol.ScopeMCP)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp " + sub,
			OK:            unified.OK,
			Message:       "MCP managed lifecycle",
			RepoRoot:      repo,
			InputDigest:   lifecycleAdapterInputDigest(unified, lifecyclecontrol.ScopeMCP),
			PlanDigest:    unified.PlanDigest,
			Data:          data,
			Warnings:      unified.Warnings,
			Errors:        unified.Errors,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(unified.Errors)
	default:
		return report.Result{}, usageErrorf("unsupported mcp subcommand: %s", sub)
	}
}

func componentSnapshotEntries(snapshots []mcpcontrol.ComponentSnapshot) []mcpcontrol.RegistryEntry {
	entries := make([]mcpcontrol.RegistryEntry, 0, len(snapshots))
	for _, snapshot := range snapshots {
		entries = append(entries, snapshot.Entry())
	}
	return entries
}

func statusMessages(results []mcpcontrol.StatusResult) ([]string, []string) {
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

func commandErrors(entries []mcpcontrol.RegistryEntry, results []mcpcontrol.CommandResult) []string {
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

func lifecycleAdapterData(result lifecyclecontrol.Report, id string) interface{} {
	for _, adapter := range result.Adapters {
		if adapter.ID == id {
			return adapter.Data
		}
	}
	return nil
}

func lifecycleAdapterInputDigest(result lifecyclecontrol.Report, id string) string {
	for _, adapter := range result.Adapters {
		if adapter.ID == id {
			return adapter.InputDigest
		}
	}
	return ""
}
