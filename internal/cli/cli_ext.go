package cli

import (
	"errors"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/cstyle"
	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/governance"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/repohealth"
	"github.com/JiaxI2/AiCoding/internal/report"
)

type aggregateCheck struct {
	Name     string      `json:"name"`
	OK       bool        `json:"ok"`
	Errors   []string    `json:"errors,omitempty"`
	Warnings []string    `json:"warnings,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

func runDocSync(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("docsync requires mode: staged, all, ci, or release")
	}
	mode := strings.ToLower(args[0])
	fs := flag.NewFlagSet("docsync "+mode, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("docsync "+mode, start, "cannot resolve repo root", nil, err.Error()), err
	}
	res := docsync.Check(repo, mode)
	return report.Result{SchemaVersion: 1, Command: "docsync " + mode, OK: res.OK, Message: "Go DocSync gate", RepoRoot: repo, Checked: res.Checked, Data: res, Warnings: res.Warnings, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
}

func runSkill(args []string, start time.Time) (report.Result, error) {
	if len(args) >= 1 && args[0] == cstyle.DefaultSkillID {
		return runCStyleCommand("skill "+cstyle.DefaultSkillID, cstyle.DefaultSkillID, args[1:], start)
	}
	if len(args) < 1 || args[0] != "verify" {
		return report.Result{}, errors.New("skill requires subcommand: verify or c99-standard-c")
	}
	fs := flag.NewFlagSet("skill verify", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, entries, err := selectedKits(*repoArg, *kitArg, *allArg)
	if err != nil {
		return report.Fail("skill verify", start, "kit selection failed", nil, err.Error()), err
	}
	res := kit.VerifySkills(repo, entries, *profile)
	return report.Result{SchemaVersion: 1, Command: "skill verify", OK: res.OK, Message: "Go skill structure verification", RepoRoot: repo, Data: res, Warnings: res.Warnings, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
}

func runLifecycle(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("lifecycle requires subcommand: plan, install, update, uninstall, rollback")
	}
	sub := strings.ToLower(args[0])
	fs := flag.NewFlagSet("lifecycle "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	actionArg := fs.String("action", "", "lifecycle action for plan")
	lastArg := fs.Bool("last", false, "rollback last snapshot")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("lifecycle "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	if sub == "rollback" {
		if !*lastArg {
			return report.Fail("lifecycle rollback", start, "rollback requires --last", nil, "missing --last"), errors.New("missing --last")
		}
		res := kit.RollbackLast(repo)
		return report.Result{SchemaVersion: 1, Command: "lifecycle rollback", OK: res.OK, Message: "Go lifecycle rollback", RepoRoot: repo, Data: res, Warnings: res.Warnings, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
	}
	action := sub
	if sub == "plan" {
		action = strings.ToLower(*actionArg)
		if action == "" {
			action = "install"
		}
	}
	repo, entries, err := selectedKits(*repoArg, *kitArg, *allArg)
	if err != nil {
		return report.Fail("lifecycle "+sub, start, "kit selection failed", nil, err.Error()), err
	}
	if sub == "plan" {
		plan := kit.PlanLifecycle(repo, entries, kit.LifecycleOptions{Action: action, Mode: selectionMode(*allArg), DryRun: true})
		errs := lifecyclePlanErrors(plan)
		return report.Result{SchemaVersion: 1, Command: "lifecycle plan", OK: len(errs) == 0, Message: "Go lifecycle plan", RepoRoot: repo, Data: plan, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	}
	if action != "install" && action != "update" && action != "uninstall" {
		return report.Result{}, errors.New("unsupported lifecycle action: " + action)
	}
	res := kit.RunAction(repo, entries, kit.ActionOptions{Action: action, Mode: selectionMode(*allArg), DryRun: false})
	return report.Result{SchemaVersion: 1, Command: "lifecycle " + action, OK: res.OK, Message: "Go lifecycle action", RepoRoot: repo, Data: res, Warnings: res.Warnings, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
}

func runExport(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	allArg := fs.Bool("all", false, "export all")
	zipArg := fs.Bool("zip", false, "write zip")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	if !*allArg || !*zipArg {
		return report.Fail("export", start, "export requires --all --zip", nil, "missing --all or --zip"), errors.New("missing --all or --zip")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("export", start, "cannot resolve repo root", nil, err.Error()), err
	}
	res, err := kit.ExportBundle(repo, "")
	errs := []string{}
	if err != nil {
		errs = append(errs, err.Error())
	}
	return report.Result{SchemaVersion: 1, Command: "export --all --zip", OK: len(errs) == 0, Message: "Go native export", RepoRoot: repo, Data: res, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runFreshClone(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("fresh-clone", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	keepTemp := fs.Bool("keep-temp", false, "keep temp clone")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("fresh-clone", start, "cannot resolve repo root", nil, err.Error()), err
	}
	res := kit.FreshClone(repo, *profile, *keepTemp)
	return report.Result{SchemaVersion: 1, Command: "fresh-clone", OK: res.OK, Message: "Go fresh clone gate", RepoRoot: repo, Data: res, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
}

func runFull(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("full", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("full", start, "cannot resolve repo root", nil, err.Error()), err
	}
	checks := runAggregate(repo, "Full", false)
	errs, warnings := aggregateErrors(checks)
	return report.Result{SchemaVersion: 1, Command: "full", OK: len(errs) == 0, Message: "Go Full aggregate", RepoRoot: repo, Data: checks, Warnings: warnings, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runReleaseCommand(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("release requires subcommand: verify or gate")
	}
	if args[0] == "verify" {
		return runRelease(args, start)
	}
	if args[0] != "gate" {
		return report.Result{}, errors.New("unsupported release subcommand: " + args[0])
	}
	fs := flag.NewFlagSet("release gate", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("release gate", start, "cannot resolve repo root", nil, err.Error()), err
	}
	checks := runAggregate(repo, "Release", true)
	errs, warnings := aggregateErrors(checks)
	return report.Result{SchemaVersion: 1, Command: "release gate", OK: len(errs) == 0, Message: "Go Release aggregate", RepoRoot: repo, Data: checks, Warnings: warnings, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runAggregate(repo, profile string, release bool) []aggregateCheck {
	checks := []aggregateCheck{}
	dsMode := "ci"
	if release {
		dsMode = "release"
	}
	ds := docsync.Check(repo, dsMode)
	checks = append(checks, aggregateCheck{Name: "docsync " + dsMode, OK: ds.OK, Errors: ds.Errors, Warnings: ds.Warnings, Data: ds})
	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		checks = append(checks, aggregateCheck{Name: "load registry", OK: false, Errors: []string{err.Error()}})
		return checks
	}
	selected, err := kit.SelectKits(entries, "", true)
	if err != nil {
		checks = append(checks, aggregateCheck{Name: "select kits", OK: false, Errors: []string{err.Error()}})
		return checks
	}
	skills := kit.VerifySkills(repo, selected, profile)
	checks = append(checks, aggregateCheck{Name: "skill verify " + profile, OK: skills.OK, Errors: skills.Errors, Warnings: skills.Warnings, Data: skills})
	structure := kit.VerifyStructure(repo, selected)
	checks = append(checks, aggregateCheck{Name: "kit structure", OK: structure.OK, Errors: structure.Errors, Warnings: structure.Warnings, Data: structure})
	checks = append(checks, aggregateCheck{Name: "governance lint", OK: len(governance.Lint(repo, "all", "")) == 0, Errors: governance.Lint(repo, "all", "")})
	_, hookErrs := repohealth.VerifyHooks(repo)
	checks = append(checks, aggregateCheck{Name: "verify hooks", OK: len(hookErrs) == 0, Errors: hookErrs})
	_, textErrs := repohealth.VerifyRepoText(repo)
	checks = append(checks, aggregateCheck{Name: "verify repo-text", OK: len(textErrs) == 0, Errors: textErrs})
	_, releaseErrs := repohealth.VerifyReleaseNotes(repo)
	checks = append(checks, aggregateCheck{Name: "verify release-notes", OK: len(releaseErrs) == 0, Errors: releaseErrs})
	if release {
		exp, err := kit.ExportBundle(repo, "")
		errs := []string{}
		if err != nil {
			errs = append(errs, err.Error())
		}
		checks = append(checks, aggregateCheck{Name: "export", OK: err == nil, Errors: errs, Data: exp})
		if os.Getenv("AICODING_SKIP_FRESH_CLONE") != "1" {
			fc := kit.FreshClone(repo, "Release", false)
			checks = append(checks, aggregateCheck{Name: "fresh-clone Release", OK: fc.OK, Errors: fc.Errors, Data: fc})
		}
	}
	return checks
}

func selectedKits(repoArg, kitArg string, allArg bool) (string, []kit.RegistryKit, error) {
	repo, err := platform.ResolveRepoRoot(repoArg)
	if err != nil {
		return "", nil, err
	}
	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		return "", nil, err
	}
	selected, err := kit.SelectKits(entries, kitArg, allArg)
	if err != nil {
		return "", nil, err
	}
	return repo, selected, nil
}
func selectionMode(all bool) string {
	if all {
		return "all"
	}
	return "kit"
}
func lifecyclePlanErrors(plan kit.LifecyclePlan) []string {
	errs := []string{}
	for _, item := range plan.Kits {
		if !item.OK {
			errs = append(errs, item.ID+": "+item.Reason)
		}
	}
	return errs
}
func aggregateErrors(checks []aggregateCheck) ([]string, []string) {
	errs := []string{}
	warnings := []string{}
	for _, c := range checks {
		for _, e := range c.Errors {
			if e != "" {
				errs = append(errs, c.Name+": "+e)
			}
		}
		for _, w := range c.Warnings {
			if w != "" {
				warnings = append(warnings, c.Name+": "+w)
			}
		}
	}
	return errs, warnings
}
