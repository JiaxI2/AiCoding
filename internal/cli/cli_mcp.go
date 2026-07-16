package cli

import (
	"context"
	"errors"
	"flag"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func runMCP(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("mcp requires subcommand: list, status, doctor, verify, install, update, or uninstall")
	}
	sub := strings.ToLower(args[0])
	component := ""
	flagArgs := args[1:]
	if len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "-") {
		component = flagArgs[0]
		flagArgs = flagArgs[1:]
	}
	fs := flag.NewFlagSet("mcp "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	componentArg := fs.String("component", component, "MCP component id")
	allArg := fs.Bool("all", false, "all enabled managed MCP components")
	profileArg := fs.String("profile", "Smoke", "Smoke, Full or Release")
	codexConfigArg := fs.String("codex-config", "", "Codex config.toml path")
	configuredArg := fs.Bool("configured", false, "include currently configured Codex MCP compatibility probes")
	dryRunArg := fs.Bool("dry-run", false, "plan lifecycle changes without writing")
	_ = fs.Bool("json", false, "JSON output")
	if err := fs.Parse(flagArgs); err != nil {
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
			Data:          inventory,
			Warnings:      inventory.Warnings,
			ElapsedMS:     report.Elapsed(start),
		}, nil
	}

	entries, err := mcpcontrol.SelectComponents(repo, *componentArg, *allArg)
	if err != nil {
		return report.Fail("mcp "+sub, start, "MCP component selection failed", nil, err.Error()), err
	}
	switch sub {
	case "status":
		status := mcpcontrol.Status(repo, *codexConfigArg, entries)
		errorsFound, warnings := statusMessages(status)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp status",
			OK:            len(errorsFound) == 0,
			Message:       "MCP component status",
			RepoRoot:      repo,
			Data:          status,
			Warnings:      warnings,
			Errors:        errorsFound,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(errorsFound)
	case "doctor":
		doctor := mcpcontrol.DoctorComponents(repo, entries)
		errorsFound := commandErrors(entries, doctor)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp doctor",
			OK:            len(errorsFound) == 0,
			Message:       "MCP managed component doctor",
			RepoRoot:      repo,
			Data:          doctor,
			Errors:        errorsFound,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(errorsFound)
	case "verify":
		includeConfigured := *configuredArg || *allArg
		verification := mcpcontrol.Verify(
			context.Background(),
			repo,
			*codexConfigArg,
			entries,
			*profileArg,
			includeConfigured,
		)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp verify",
			OK:            verification.OK,
			Message:       "MCP managed and configured compatibility verification",
			RepoRoot:      repo,
			Data:          verification,
			Warnings:      verification.Warnings,
			Errors:        verification.Errors,
			ElapsedMS:     report.Elapsed(start),
		}, mcpcontrol.VerifyErrors(verification)
	case "install", "update", "uninstall":
		results := mcpcontrol.RunLifecycle(repo, *codexConfigArg, entries, sub, *dryRunArg)
		errorsFound, warnings := lifecycleMessages(results)
		return report.Result{
			SchemaVersion: 1,
			Command:       "mcp " + sub,
			OK:            len(errorsFound) == 0,
			Message:       "MCP managed lifecycle",
			RepoRoot:      repo,
			Data:          results,
			Warnings:      warnings,
			Errors:        errorsFound,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(errorsFound)
	default:
		return report.Result{}, errors.New("unsupported mcp subcommand: " + sub)
	}
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

func lifecycleMessages(results []mcpcontrol.LifecycleResult) ([]string, []string) {
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
