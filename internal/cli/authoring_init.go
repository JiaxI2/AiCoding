package cli

import (
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func runSkillInit(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("skill init")
	repoArg := fs.String("repo-root", "", "repository root")
	outArg := fs.String("out", "", "writable output directory outside the AiCoding repository")
	dryRunArg := fs.Bool("dry-run", false, "report the scaffold without writing")
	_ = fs.Bool("json", false, "json output")
	id, flagArgs := authoringInitID(args)
	if err := parseNoPositionals(fs, flagArgs); err != nil {
		return report.Result{}, err
	}
	if id == "" {
		return report.Result{}, usageErrorf("skill init requires an id before flags")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("skill init", start, "cannot resolve repo root", nil, err.Error()), err
	}
	initReport, initErr := kit.InitSkill(repo, id, kit.SkillInitOptions{Out: *outArg, DryRun: *dryRunArg})
	result := report.Result{
		SchemaVersion: report.SchemaVersion, Command: "skill init", OK: initReport.OK,
		Message: "external Skill authoring scaffold", RepoRoot: repo, Data: initReport,
		Errors: append([]string(nil), initReport.Errors...), ElapsedMS: report.Elapsed(start),
	}
	if initErr != nil {
		result.ErrorKind = report.ErrorKindValidation
		result = report.WithDecision(result, report.CategoryValidation,
			"aicoding skill init "+id+" --out <writable-Codex-Skills-worktree> --json")
		return result, report.BoolErr(result.Errors)
	}
	return result, nil
}

func runMCPInit(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("mcp init")
	repoArg := fs.String("repo-root", "", "repository root")
	outArg := fs.String("out", "", "output directory for the component manifest")
	dryRunArg := fs.Bool("dry-run", false, "report the scaffold without writing")
	_ = fs.Bool("json", false, "json output")
	id, flagArgs := authoringInitID(args)
	if err := parseNoPositionals(fs, flagArgs); err != nil {
		return report.Result{}, err
	}
	if id == "" {
		return report.Result{}, usageErrorf("mcp init requires an id before flags")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("mcp init", start, "cannot resolve repo root", nil, err.Error()), err
	}
	initReport, initErr := mcpcontrol.InitComponentScaffold(repo, id, mcpcontrol.InitOptions{Out: *outArg, DryRun: *dryRunArg})
	result := report.Result{
		SchemaVersion: report.SchemaVersion, Command: "mcp init", OK: initReport.OK,
		Message: "MCP component manifest scaffold", RepoRoot: repo, Data: initReport,
		Errors: append([]string(nil), initReport.Errors...), ElapsedMS: report.Elapsed(start),
	}
	if initErr != nil {
		result.ErrorKind = report.ErrorKindValidation
		result = report.WithDecision(result, report.CategoryValidation, "aicoding mcp init "+id+" --dry-run --json")
		return result, report.BoolErr(result.Errors)
	}
	return result, nil
}

func authoringInitID(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}
