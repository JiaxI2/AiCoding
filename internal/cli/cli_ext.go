package cli

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/cstyle"
	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/governance"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/repohealth"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/reuse"
	"github.com/JiaxI2/AiCoding/internal/runner"
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
	elapsed := report.Elapsed(start)
	data := standardReport("docsync "+mode, mode, elapsed, map[string]interface{}{
		"mode":       res.Mode,
		"checked":    len(res.Checked),
		"risk_files": len(res.RiskFiles),
		"doc_files":  len(res.DocFiles),
		"warnings":   len(res.Warnings),
		"errors":     len(res.Errors),
	}, res.Warnings, res.Errors, res)
	return report.Result{SchemaVersion: 1, Command: "docsync " + mode, OK: res.OK, Message: "Go DocSync gate", RepoRoot: repo, Checked: res.Checked, Data: data, Warnings: res.Warnings, Errors: res.Errors, ElapsedMS: elapsed}, report.BoolErr(res.Errors)
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
	reuseCheck := reuse.Verify(repo)
	for _, issue := range reuseCheck.Errors {
		res.Errors = append(res.Errors, "reuse governance: "+issue)
	}
	for _, warning := range reuseCheck.Warnings {
		res.Warnings = append(res.Warnings, "reuse governance: "+warning)
	}
	res.OK = res.OK && reuseCheck.OK
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

func runSmoke(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("smoke", start, "cannot resolve repo root", nil, err.Error()), err
	}
	checks := runAggregate(repo, "Smoke", false)
	errs, warnings := aggregateErrors(checks)
	return report.Result{SchemaVersion: 1, Command: "smoke", OK: len(errs) == 0, Message: "Go Smoke aggregate", RepoRoot: repo, Data: checks, Warnings: warnings, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runCI(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("ci", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("ci", start, "cannot resolve repo root", nil, err.Error()), err
	}
	plan := runner.NewPlan(goTestTask(repo))
	aggregatePlan := buildAggregatePlan(repo, *profile, strings.EqualFold(*profile, "Release"))
	for _, task := range aggregatePlan.Tasks() {
		plan.Add(task)
	}
	checks := aggregateChecks(plan.Run(context.Background(), runner.Options{MaxParallel: 4}))
	errs, warnings := aggregateErrors(checks)
	return report.Result{SchemaVersion: 1, Command: "ci", OK: len(errs) == 0, Message: "Go CI aggregate", RepoRoot: repo, Data: checks, Warnings: warnings, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
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
	plan := buildAggregatePlan(repo, profile, release)
	return aggregateChecks(plan.Run(context.Background(), runner.Options{MaxParallel: 4}))
}

func buildAggregatePlan(repo, profile string, release bool) runner.Plan {
	plan := runner.NewPlan()
	dsMode := "ci"
	if release {
		dsMode = "release"
	}
	plan.Add(runner.Task{ID: "docsync " + dsMode, Group: "docsync", Run: func(context.Context) runner.TaskResult {
		ds := docsync.Check(repo, dsMode)
		return runner.TaskResult{OK: ds.OK, Errors: ds.Errors, Warnings: ds.Warnings, Data: ds}
	}})
	plan.Add(runner.Task{ID: "reuse governance", Group: "governance", Run: func(context.Context) runner.TaskResult {
		check := reuse.Verify(repo)
		return runner.TaskResult{OK: check.OK, Errors: check.Errors, Warnings: check.Warnings, Data: check}
	}})

	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		plan.Add(staticFailureTask("load registry", err.Error()))
		return plan
	}
	selected, err := kit.SelectKits(entries, "", true)
	if err != nil {
		plan.Add(staticFailureTask("select kits", err.Error()))
		return plan
	}

	plan.Add(runner.Task{ID: "skill verify " + profile, Group: "kit", Run: func(context.Context) runner.TaskResult {
		skills := kit.VerifySkills(repo, selected, profile)
		return runner.TaskResult{OK: skills.OK, Errors: skills.Errors, Warnings: skills.Warnings, Data: skills}
	}})
	plan.Add(runner.Task{ID: "kit structure", Group: "kit", Run: func(context.Context) runner.TaskResult {
		structure := kit.VerifyStructure(repo, selected)
		return runner.TaskResult{OK: structure.OK, Errors: structure.Errors, Warnings: structure.Warnings, Data: structure}
	}})
	if platform.IsFile(platform.RepoPath(repo, "config/mcp-registry.json")) {
		plan.Add(runner.Task{ID: "mcp registry", Group: "mcp", Run: func(context.Context) runner.TaskResult {
			errs := mcpcontrol.DoctorRegistry(repo)
			return runner.TaskResult{OK: len(errs) == 0, Errors: errs}
		}})
	}
	plan.Add(runner.Task{ID: "governance lint", Group: "governance", Run: func(context.Context) runner.TaskResult {
		errs := governance.Lint(repo, "all", "")
		return runner.TaskResult{OK: len(errs) == 0, Errors: errs}
	}})
	plan.Add(runner.Task{ID: "verify hooks", Group: "repohealth", Run: func(context.Context) runner.TaskResult {
		checks, errs := repohealth.VerifyHooks(repo)
		return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: checks}
	}})
	plan.Add(runner.Task{ID: "verify repo-text", Group: "repohealth", Run: func(context.Context) runner.TaskResult {
		checks, errs := repohealth.VerifyRepoText(repo)
		return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: checks}
	}})
	plan.Add(runner.Task{ID: "verify release-notes", Group: "repohealth", Run: func(context.Context) runner.TaskResult {
		checks, errs := repohealth.VerifyReleaseNotes(repo)
		return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: checks}
	}})
	plan.Add(runner.Task{ID: "doctor perf", Group: "repohealth", Run: func(context.Context) runner.TaskResult {
		res, err := runDoctor([]string{"perf", "--repo-root", repo}, time.Now())
		return taskResultFromReport(res, err)
	}})
	if release {
		plan.Add(runner.Task{ID: "export", Group: "release", Run: func(context.Context) runner.TaskResult {
			exp, err := kit.ExportBundle(repo, "")
			errs := []string{}
			if err != nil {
				errs = append(errs, err.Error())
			}
			return runner.TaskResult{OK: err == nil, Errors: errs, Data: exp}
		}})
		if os.Getenv("AICODING_SKIP_FRESH_CLONE") != "1" {
			plan.Add(runner.Task{ID: "fresh-clone Release", Group: "release", Run: func(context.Context) runner.TaskResult {
				fc := kit.FreshClone(repo, "Release", false)
				return runner.TaskResult{OK: fc.OK, Errors: fc.Errors, Data: fc}
			}})
		}
	}
	return plan
}

func goTestTask(repo string) runner.Task {
	return runner.Task{ID: "go test ./...", Group: "go", Critical: true, Timeout: 5 * time.Minute, Run: func(ctx context.Context) runner.TaskResult {
		cmd := exec.CommandContext(ctx, "go", "test", "./...")
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		data := map[string]string{"output": strings.TrimSpace(string(out))}
		if err != nil {
			return runner.TaskResult{OK: false, Errors: []string{err.Error()}, Data: data}
		}
		return runner.TaskResult{OK: true, Data: data}
	}}
}

func staticFailureTask(id, errText string) runner.Task {
	return runner.Task{ID: id, Group: "setup", Run: func(context.Context) runner.TaskResult {
		return runner.TaskResult{OK: false, Errors: []string{errText}}
	}}
}

func taskResultFromReport(res report.Result, err error) runner.TaskResult {
	errs := append([]string{}, res.Errors...)
	if err != nil && len(errs) == 0 {
		errs = append(errs, err.Error())
	}
	return runner.TaskResult{OK: err == nil && res.OK, Errors: errs, Warnings: res.Warnings, Data: res.Data}
}

func aggregateChecks(results []runner.TaskResult) []aggregateCheck {
	checks := make([]aggregateCheck, 0, len(results))
	for _, result := range results {
		checks = append(checks, aggregateCheck{Name: result.ID, OK: result.OK, Errors: result.Errors, Warnings: result.Warnings, Data: result.Data})
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
