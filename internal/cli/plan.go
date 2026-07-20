package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	plancheck "github.com/JiaxI2/AiCoding/internal/plan"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

const planRequiredAction = `pwsh tools/specialty/new-agent-plan-mode-session.ps1 -Feature "<功能名>" -Description "<需求描述>" -NeedsDecision -Json`

func runPlan(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || strings.ToLower(args[0]) != "check" {
		return report.Result{}, usageErrorf("plan requires subcommand: check")
	}
	fs := newFlagSet("plan check")
	repoArg := fs.String("repo-root", "", "repository root")
	stagedArg := fs.Bool("staged", false, "check staged paths")
	_ = fs.Bool("json", false, "json output")
	var pathArgs multiFlag
	fs.Var(&pathArgs, "paths", "repository-relative path; can be repeated")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	if *stagedArg == (len(pathArgs) > 0) {
		return report.Result{}, usageErrorf("plan check requires exactly one of --staged or --paths")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("plan check", start, "cannot resolve repo root", nil, err.Error()), err
	}
	policy, err := plancheck.LoadPolicy(repo)
	if err != nil {
		return planFailure(repo, start, "cannot load plan trigger policy", err)
	}
	paths := []string(pathArgs)
	if *stagedArg {
		paths, err = gitx.StagedFiles(repo)
		if err != nil {
			return planFailure(repo, start, "cannot read staged paths", err)
		}
	}
	check, err := plancheck.CheckPaths(policy, paths)
	if err != nil {
		return planFailure(repo, start, "cannot check plan trigger paths", err)
	}
	if len(check.Sensitive) > 0 {
		check.RequiredAction = planRequiredAction
		errors := []string{fmt.Sprintf("%d architecture-sensitive path(s) require Plan Mode artifacts", len(check.Sensitive))}
		return report.Result{
			SchemaVersion: report.SchemaVersion,
			Command:       "plan check",
			OK:            false,
			ErrorKind:     report.ErrorKindValidation,
			Message:       "architecture-sensitive changes require Plan Mode",
			RepoRoot:      repo,
			Checked:       check.Paths,
			Data:          check,
			Errors:        errors,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(errors)
	}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "plan check",
		OK:            true,
		Message:       "no unapproved Plan Mode trigger detected",
		RepoRoot:      repo,
		Checked:       check.Paths,
		Data:          check,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func planFailure(repo string, start time.Time, message string, err error) (report.Result, error) {
	result := report.Fail("plan check", start, message, nil, err.Error())
	result.RepoRoot = repo
	result.ErrorKind = report.ErrorKindValidation
	return result, report.BoolErr(result.Errors)
}
