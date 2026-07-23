package cli

import (
	"context"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func runMCP(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("mcp requires subcommand: init, list, status, doctor, or verify")
	}
	subID, err := resolveCatalogSubcommandID(CommandMCP, args[0])
	if err != nil {
		return report.Result{}, err
	}
	sub := args[0]
	if subID == SubMCPInit {
		return runMCPInit(args[1:], start)
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
	profileArg := fs.String("profile", "Smoke", productProfileHelp())
	codexConfigArg := fs.String("codex-config", "", "Codex config.toml path")
	configuredArg := fs.Bool("configured", false, "include currently configured Codex MCP compatibility probes")
	_ = fs.Bool("json", false, "JSON output")
	if err := parseNoPositionals(fs, flagArgs); err != nil {
		return report.Result{}, err
	}
	_, profileDisplay, err := normalizeTestProfile(*profileArg)
	if err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("mcp "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	if subID == SubMCPList {
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
	switch subID {
	case SubMCPStatus:
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
	case SubMCPDoctor:
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
	case SubMCPVerify:
		includeConfigured := *configuredArg || *allArg
		verification := mcpcontrol.VerifyCatalog(
			context.Background(),
			repo,
			*codexConfigArg,
			components,
			profileDisplay,
			includeConfigured,
		)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp verify",
			OK:            verification.OK,
			Message:       "MCP managed and configured verification",
			RepoRoot:      repo,
			InputDigest:   catalog.Digest(),
			Data:          verification,
			Warnings:      verification.Warnings,
			Errors:        verification.Errors,
			ElapsedMS:     report.Elapsed(start),
		}, mcpcontrol.VerifyErrors(verification)
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
