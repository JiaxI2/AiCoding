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
	if len(args) < 1 {
		return report.Result{}, usageErrorf("plan requires subcommand: check, verify, or status")
	}
	switch strings.ToLower(args[0]) {
	case "check":
		return runPlanCheck(args[1:], start)
	case "verify":
		return runPlanVerify(args[1:], start)
	case "status":
		return runPlanStatus(args[1:], start)
	default:
		return report.Result{}, usageErrorf("unsupported plan subcommand: %s", args[0])
	}
}

func runPlanCheck(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("plan check")
	repoArg := fs.String("repo-root", "", "repository root")
	stagedArg := fs.Bool("staged", false, "check staged paths")
	_ = fs.Bool("json", false, "json output")
	var pathArgs multiFlag
	fs.Var(&pathArgs, "paths", "repository-relative path; can be repeated")
	if err := parseNoPositionals(fs, args); err != nil {
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

func runPlanVerify(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("plan verify")
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("plan verify", start, "cannot resolve repo root", nil, err.Error()), err
	}
	verification, err := plancheck.VerifySpecs(repo)
	if err != nil {
		return planFailureFor("plan verify", repo, start, "cannot verify plan specs", err)
	}
	result := report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "plan verify",
		OK:            verification.OK,
		Message:       "Plan Mode artifact verification",
		RepoRoot:      repo,
		Data:          verification,
		Warnings:      verification.Warnings,
		Errors:        verification.Errors,
		ElapsedMS:     report.Elapsed(start),
	}
	if !verification.OK {
		result.ErrorKind = report.ErrorKindValidation
	}
	return result, report.BoolErr(verification.Errors)
}

func runPlanStatus(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("plan status")
	repoArg := fs.String("repo-root", "", "repository root")
	idArg := fs.String("id", "", "plan id")
	allArg := fs.Bool("all", false, "list all plans")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if *allArg && strings.TrimSpace(*idArg) != "" {
		return report.Result{}, usageErrorf("plan status accepts either --id or --all")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("plan status", start, "cannot resolve repo root", nil, err.Error()), err
	}
	specs, err := plancheck.ListSpecs(repo)
	if err != nil {
		return planFailureFor("plan status", repo, start, "cannot list plan specs", err)
	}
	selected := specs
	if id := strings.TrimSpace(*idArg); id != "" {
		selected = []plancheck.Spec{}
		for _, spec := range specs {
			if spec.ID == id {
				selected = append(selected, spec)
				break
			}
		}
		if len(selected) == 0 {
			return planFailureFor("plan status", repo, start, "plan id was not found", fmt.Errorf("unknown plan id: %s", id))
		}
	}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "plan status",
		OK:            true,
		Message:       "Plan Mode artifact status",
		RepoRoot:      repo,
		Data:          selected,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func planFailure(repo string, start time.Time, message string, err error) (report.Result, error) {
	return planFailureFor("plan check", repo, start, message, err)
}

func planFailureFor(command, repo string, start time.Time, message string, err error) (report.Result, error) {
	result := report.Fail(command, start, message, nil, err.Error())
	result.RepoRoot = repo
	result.ErrorKind = report.ErrorKindValidation
	return result, report.BoolErr(result.Errors)
}
