package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/loopkit/gateref"
	plancheck "github.com/JiaxI2/AiCoding/internal/plan"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

const planRequiredAction = `pwsh tools/specialty/new-agent-plan-mode-session.ps1 -Feature "<功能名>" -Description "<需求描述>" -NeedsDecision -Json`

type planGateStatus struct {
	Ref       gateref.GateRef `json:"ref"`
	Satisfied bool            `json:"satisfied"`
}

type planStatusView struct {
	Spec                plancheck.Spec           `json:"spec"`
	Binding             *plancheck.BindingStatus `json:"binding,omitempty"`
	Gates               []planGateStatus         `json:"gates"`
	CompletionSuggested bool                     `json:"completionSuggested"`
	RequiredAction      string                   `json:"requiredAction,omitempty"`
}

func runPlan(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("plan requires subcommand: check, verify, status, or approve")
	}
	sub, err := resolveCatalogSubcommandID(CommandPlan, args[0])
	if err != nil {
		return report.Result{}, err
	}
	switch sub {
	case SubPlanCheck:
		return runPlanCheck(args[1:], start)
	case SubPlanVerify:
		return runPlanVerify(args[1:], start)
	case SubPlanStatus:
		return runPlanStatus(args[1:], start)
	case SubPlanApprove:
		return runPlanApprove(args[1:], start)
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
		verification, verifyErr := plancheck.VerifySpecs(repo)
		if verifyErr != nil {
			return planFailure(repo, start, "cannot verify plan specs", verifyErr)
		}
		if !verification.OK {
			return planFailure(repo, start, "cannot use invalid plan specs", fmt.Errorf("%s", strings.Join(verification.Errors, "; ")))
		}
		check.ApprovedPlans, check.Uncovered, err = plancheck.ApprovedCoverage(verification.Specs, check.Sensitive)
		if err != nil {
			return planFailure(repo, start, "cannot evaluate approved plan coverage", err)
		}
	}
	if len(check.Uncovered) > 0 {
		check.RequiredAction = planRequiredAction
		errors := []string{fmt.Sprintf("%d architecture-sensitive path(s) are not covered by an approved plan", len(check.Uncovered))}
		return report.Result{
			SchemaVersion: report.SchemaVersion,
			Command:       "plan check",
			OK:            false,
			ErrorKind:     report.ErrorKindValidation,
			Message:       "architecture-sensitive changes require approved Plan Mode scope",
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
		Message:       "all architecture-sensitive paths are covered or exempt",
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
	views := make([]planStatusView, 0, len(selected))
	warnings := []string{}
	var policy plancheck.Policy
	policyLoaded := false
	for _, spec := range selected {
		view := planStatusView{Spec: spec, Gates: []planGateStatus{}}
		if spec.Status == plancheck.StatusApproved || spec.Status == plancheck.StatusImplemented {
			if !policyLoaded {
				policy, err = plancheck.LoadPolicy(repo)
				if err != nil {
					return planFailureFor("plan status", repo, start, "cannot load plan policy", err)
				}
				policyLoaded = true
			}
			currentTree, treeErr := gitx.TreeOID(repo, "HEAD")
			if treeErr != nil {
				return planFailureFor("plan status", repo, start, "cannot resolve current tree", treeErr)
			}
			changed, diffErr := gitx.DiffTreeFiles(repo, spec.ApprovedTree, currentTree)
			if diffErr != nil {
				return planFailureFor("plan status", repo, start, "cannot compare approved tree", diffErr)
			}
			binding, bindingErr := plancheck.EvaluateBinding(policy, spec, currentTree, changed)
			if bindingErr != nil {
				return planFailureFor("plan status", repo, start, "cannot evaluate plan drift", bindingErr)
			}
			view.Binding = &binding
			view.Gates, err = planValidationGates(repo, currentTree, spec.Gates)
			if err != nil {
				return planFailureFor("plan status", repo, start, "cannot check validation evidence", err)
			}
			allGates := len(view.Gates) > 0
			for _, gate := range view.Gates {
				allGates = allGates && gate.Satisfied
			}
			view.CompletionSuggested = binding.ScopeCovered && allGates && len(binding.OutOfScope) == 0
			switch {
			case len(binding.OutOfScope) > 0:
				view.RequiredAction = "review out-of-scope changes before continuing"
				warnings = append(warnings, fmt.Sprintf("plan %s has %d out-of-scope change(s)", spec.ID, len(binding.OutOfScope)))
			case len(binding.Drift) > 0:
				view.RequiredAction = "implementation is in progress; re-review the plan if scope changed"
				warnings = append(warnings, fmt.Sprintf("plan %s has %d in-scope drift path(s)", spec.ID, len(binding.Drift)))
			case !allGates:
				view.RequiredAction = "run the declared validation profiles for the current tree"
			}
		}
		views = append(views, view)
	}
	sort.Strings(warnings)
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "plan status",
		OK:            true,
		Message:       "Plan Mode artifact status",
		RepoRoot:      repo,
		Data:          views,
		Warnings:      warnings,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func runPlanApprove(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("plan approve")
	repoArg := fs.String("repo-root", "", "repository root")
	idArg := fs.String("id", "", "plan id")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	id := strings.TrimSpace(*idArg)
	if id == "" {
		return report.Result{}, usageErrorf("plan approve requires --id")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("plan approve", start, "cannot resolve repo root", nil, err.Error()), err
	}
	status, err := gitx.StatusSnapshot(repo)
	if err != nil {
		return planFailureFor("plan approve", repo, start, "cannot read worktree status", err)
	}
	if status.TrackedModified || status.Staged || status.Untracked || status.SubmoduleDirty || status.Unmerged {
		return planFailureFor("plan approve", repo, start, "cannot approve plan", fmt.Errorf("plan approval requires a clean worktree"))
	}
	tree, err := gitx.TreeOID(repo, "HEAD")
	if err != nil {
		return planFailureFor("plan approve", repo, start, "cannot resolve approval tree", err)
	}
	spec, err := plancheck.Approve(repo, id, tree)
	if err != nil {
		return planFailureFor("plan approve", repo, start, "cannot approve plan", err)
	}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "plan approve",
		OK:            true,
		Message:       "plan approved and bound to the clean HEAD tree",
		RepoRoot:      repo,
		Data:          spec,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func planValidationGates(repo, tree string, gates []plancheck.Gate) ([]planGateStatus, error) {
	store, err := validationevidence.Open(repo)
	if err != nil {
		return nil, err
	}
	subject := validationevidence.Subject{
		TreeOID: tree, Mode: validationevidence.SubjectHead, Reusable: true,
		Scope: validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}
	statuses := make([]planGateStatus, 0, len(gates))
	for _, gate := range gates {
		profile, _, normalizeErr := normalizeTestProfile(gate.Profile)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		evidenceSpec, evidenceErr := testengine.EvidenceSpec(validationTestConfig(repo, profile))
		if evidenceErr != nil {
			return nil, evidenceErr
		}
		fingerprint, fingerprintErr := store.Fingerprint(subject, evidenceSpec)
		if fingerprintErr != nil {
			return nil, fingerprintErr
		}
		decision := store.Check(subject, fingerprint)
		status := planGateStatus{
			Ref:       gateref.GateRef{Profile: profile, ValidationIdentity: fingerprint.Identity},
			Satisfied: decision.Hit,
		}
		if decision.Hit && decision.Receipt != nil {
			status.Ref.ValidationIdentity = decision.Receipt.ValidationIdentity
			status.Ref.ReceiptID = decision.Receipt.ReceiptID
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
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
