package cli

import (
	"context"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/cstyle"
	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/kit"
	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
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
		return report.Result{}, usageErrorf("lifecycle requires subcommand: plan, install, update, status, doctor, verify, uninstall, or rollback")
	}
	sub := strings.ToLower(args[0])
	if !validChoice(sub, "plan", "install", "update", "status", "doctor", "verify", "uninstall", "rollback") {
		return report.Result{}, usageErrorf("unsupported lifecycle action: %s", sub)
	}
	fs := newFlagSet("lifecycle " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	scopeArg := fs.String("scope", lifecyclecontrol.ScopeKit, "kit, mcp, runtime-skill, or all")
	kitArg := fs.String("kit", "", "kit id")
	componentArg := fs.String("component", "", "MCP component id")
	allArg := fs.Bool("all", false, "all enabled entries in the selected adapter")
	actionArg := fs.String("action", "", "lifecycle action for plan")
	lastArg := fs.Bool("last", false, "rollback last snapshot")
	profileArg := fs.String("profile", "Smoke", "verification profile: Smoke, Full or Release")
	codexConfigArg := fs.String("codex-config", "", "Codex config.toml path")
	configuredArg := fs.Bool("configured", false, "include configured Codex MCP compatibility probes")
	runtimeProfileArg := fs.String("runtime-profile", "", "runtime, full, or skill-development")
	runtimeSkillArg := fs.String("runtime-skill", "", "selected canonical Skill for skill-development or targeted removal")
	sourceRepositoryArg := fs.String("source-repository", "", "Codex-Skills source repository")
	standaloneRootArg := fs.String("standalone-root", "agents", "agents or codex")
	migrateUnmanagedArg := fs.Bool("migrate-unmanaged", false, "back up and replace registered unmanaged runtime Skill paths")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("lifecycle "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}

	scope := strings.ToLower(strings.TrimSpace(*scopeArg))
	if !validChoice(scope, lifecyclecontrol.ScopeKit, lifecyclecontrol.ScopeMCP, lifecyclecontrol.ScopeRuntimeSkill, lifecyclecontrol.ScopeAll) {
		return report.Result{}, usageErrorf("unsupported lifecycle scope: %s", scope)
	}
	standaloneRoot := strings.ToLower(strings.TrimSpace(*standaloneRootArg))
	if !validChoice(standaloneRoot, "agents", "codex") {
		return report.Result{}, usageErrorf("unsupported standalone root: %s", *standaloneRootArg)
	}
	runtimeProfile := strings.ToLower(strings.TrimSpace(*runtimeProfileArg))
	if runtimeProfile != "" && !validChoice(runtimeProfile, "runtime", "full", "skill-development") {
		return report.Result{}, usageErrorf("unsupported runtime profile: %s", *runtimeProfileArg)
	}
	if scope == lifecyclecontrol.ScopeAll && (*kitArg != "" || *componentArg != "") {
		return report.Result{}, usageErrorf("lifecycle --scope all does not accept --kit or --component")
	}
	if scope == lifecyclecontrol.ScopeKit && *componentArg != "" {
		return report.Result{}, usageErrorf("lifecycle --scope kit does not accept --component")
	}
	if scope == lifecyclecontrol.ScopeMCP && *kitArg != "" {
		return report.Result{}, usageErrorf("lifecycle --scope mcp does not accept --kit")
	}
	hasRuntimeFlags := runtimeProfile != "" || *runtimeSkillArg != "" || *sourceRepositoryArg != "" || *migrateUnmanagedArg
	if scope != lifecyclecontrol.ScopeRuntimeSkill && scope != lifecyclecontrol.ScopeAll && hasRuntimeFlags {
		return report.Result{}, usageErrorf("runtime Skill flags require --scope runtime-skill or --scope all")
	}

	if sub == "rollback" {
		if !*lastArg {
			return report.Result{}, usageErrorf("lifecycle rollback requires --last")
		}
		if scope != lifecyclecontrol.ScopeKit {
			return report.Result{}, usageErrorf("lifecycle rollback currently supports --scope kit only")
		}
	}
	action := sub
	if sub == "plan" {
		action = strings.ToLower(*actionArg)
		if action == "" {
			action = "install"
		}
		if !validChoice(action, "install", "update", "uninstall") {
			return report.Result{}, usageErrorf("lifecycle plan requires --action install|update|uninstall")
		}
	}
	if !validChoice(action, "install", "update", "status", "doctor", "verify", "uninstall", "rollback") {
		return report.Result{}, usageErrorf("unsupported lifecycle action: %s", action)
	}
	if scope == lifecyclecontrol.ScopeKit && action != "rollback" && !*allArg && *kitArg == "" {
		return report.Result{}, usageErrorf("lifecycle --scope kit requires --all or --kit")
	}
	if scope == lifecyclecontrol.ScopeMCP && !*allArg && *componentArg == "" {
		return report.Result{}, usageErrorf("lifecycle --scope mcp requires --all or --component")
	}
	if (scope == lifecyclecontrol.ScopeRuntimeSkill || scope == lifecyclecontrol.ScopeAll) &&
		validChoice(action, "install", "update") && runtimeProfile == "" {
		return report.Result{}, usageErrorf("runtime Skill apply requires --runtime-profile")
	}
	if runtimeProfile == "skill-development" && *runtimeSkillArg == "" {
		return report.Result{}, usageErrorf("skill-development runtime profile requires --runtime-skill")
	}

	lifecycleReport := lifecyclecontrol.Run(context.Background(), repo, lifecyclecontrol.Options{
		Action:            action,
		Scope:             scope,
		All:               *allArg || scope == lifecyclecontrol.ScopeAll,
		KitID:             *kitArg,
		ComponentID:       *componentArg,
		CodexConfig:       *codexConfigArg,
		VerifyProfile:     *profileArg,
		IncludeConfigured: *configuredArg,
		DryRun:            sub == "plan",
		RuntimeProfile:    runtimeProfile,
		RuntimeSkill:      *runtimeSkillArg,
		SourceRepository:  *sourceRepositoryArg,
		StandaloneRoot:    standaloneRoot,
		MigrateUnmanaged:  *migrateUnmanagedArg,
	})
	return report.Result{
		SchemaVersion: 1,
		Command:       "lifecycle " + sub,
		OK:            lifecycleReport.OK,
		Message:       "unified lifecycle control plane",
		RepoRoot:      repo,
		Data:          lifecycleReport,
		Warnings:      lifecycleReport.Warnings,
		Errors:        lifecycleReport.Errors,
		ElapsedMS:     report.Elapsed(start),
	}, report.BoolErr(lifecycleReport.Errors)
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
