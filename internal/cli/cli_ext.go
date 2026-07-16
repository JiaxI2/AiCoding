package cli

import (
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/cstyle"
	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/reuse"
)

func runDocSync(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("docsync requires mode: staged, all, ci, or release")
	}
	mode := strings.ToLower(args[0])
	switch mode {
	case "staged", "all", "ci", "release":
	default:
		return report.Result{}, usageErrorf("unsupported docsync mode: %s", mode)
	}
	fs := newFlagSet("docsync " + mode)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
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
		return report.Result{}, usageErrorf("skill requires subcommand: verify or c99-standard-c")
	}
	fs := newFlagSet("skill verify")
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
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
		return report.Result{}, usageErrorf("lifecycle requires subcommand: plan, install, update, uninstall, rollback")
	}
	sub := strings.ToLower(args[0])
	if !validChoice(sub, "plan", "install", "update", "uninstall", "rollback") {
		return report.Result{}, usageErrorf("unsupported lifecycle action: %s", sub)
	}
	fs := newFlagSet("lifecycle " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	actionArg := fs.String("action", "", "lifecycle action for plan")
	lastArg := fs.Bool("last", false, "rollback last snapshot")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("lifecycle "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	if sub == "rollback" {
		if !*lastArg {
			return report.Result{}, usageErrorf("lifecycle rollback requires --last")
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
		return report.Result{}, usageErrorf("unsupported lifecycle action: %s", action)
	}
	res := kit.RunAction(repo, entries, kit.ActionOptions{Action: action, Mode: selectionMode(*allArg), DryRun: false})
	return report.Result{SchemaVersion: 1, Command: "lifecycle " + action, OK: res.OK, Message: "Go lifecycle action", RepoRoot: repo, Data: res, Warnings: res.Warnings, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
}

func runExport(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("export")
	repoArg := fs.String("repo-root", "", "repository root")
	allArg := fs.Bool("all", false, "export all")
	zipArg := fs.Bool("zip", false, "write zip")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if !*allArg || !*zipArg {
		return report.Result{}, usageErrorf("export requires --all --zip")
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
	fs := newFlagSet("fresh-clone")
	repoArg := fs.String("repo-root", "", "repository root")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	keepTemp := fs.Bool("keep-temp", false, "keep temp clone")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("fresh-clone", start, "cannot resolve repo root", nil, err.Error()), err
	}
	res := kit.FreshClone(repo, *profile, *keepTemp)
	return report.Result{SchemaVersion: 1, Command: "fresh-clone", OK: res.OK, Message: "Go fresh clone gate", RepoRoot: repo, Data: res, Errors: res.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(res.Errors)
}

func runSmoke(args []string, start time.Time) (report.Result, error) {
	return runTestProfile("smoke", args, "smoke", start)
}

func runCI(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("ci")
	repoArg := fs.String("repo-root", "", "repository root")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	jsonArg := fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	testArgs := []string{"--profile", *profile}
	if *repoArg != "" {
		testArgs = append(testArgs, "--repo-root", *repoArg)
	}
	if *jsonArg {
		testArgs = append(testArgs, "--json")
	}
	return runTestProfile("", testArgs, "ci", start)
}

func runFull(args []string, start time.Time) (report.Result, error) {
	return runTestProfile("full", args, "full", start)
}

func runReleaseCommand(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("release requires subcommand: verify or gate")
	}
	if args[0] == "verify" {
		return runRelease(args, start)
	}
	if args[0] != "gate" {
		return report.Result{}, usageErrorf("unsupported release subcommand: %s", args[0])
	}
	return runTestProfile("release", args[1:], "release gate", start)
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
