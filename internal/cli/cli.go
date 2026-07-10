package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/bootstrap"
	"github.com/JiaxI2/AiCoding/internal/cache"
	"github.com/JiaxI2/AiCoding/internal/cstyle"
	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/governance"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/pwshregex"
	"github.com/JiaxI2/AiCoding/internal/releasegate"
	"github.com/JiaxI2/AiCoding/internal/repohealth"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/runner"
	"github.com/JiaxI2/AiCoding/internal/tagpolicy"
)

const version = "fast-path-v2"

func Main() {
	start := time.Now()
	if len(os.Args) < 2 {
		printUsageAndExit(2)
	}
	cmd := os.Args[1]
	var res report.Result
	var err error
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println(version)
		return
	case "hook":
		res, err = runHook(os.Args[2:], start)
	case "bootstrap":
		res, err = runBootstrap(os.Args[2:], start)
	case "smoke":
		res, err = runSmoke(os.Args[2:], start)
	case "ci":
		res, err = runCI(os.Args[2:], start)
	case "test":
		res, err = runTest(os.Args[2:], start)

	case "docsync":
		res, err = runDocSync(os.Args[2:], start)
	case "skill":
		res, err = runSkill(os.Args[2:], start)
	case "lifecycle":
		res, err = runLifecycle(os.Args[2:], start)
	case "export":
		res, err = runExport(os.Args[2:], start)
	case "fresh-clone":
		res, err = runFreshClone(os.Args[2:], start)
	case "full":
		res, err = runFull(os.Args[2:], start)
	case "cache":
		res, err = runCache(os.Args[2:], start)

	case "tag":
		res, err = runTag(os.Args[2:], start)
	case "release":
		res, err = runReleaseCommand(os.Args[2:], start)
	case "kit":
		res, err = runKit(os.Args[2:], start)
	case "doctor":
		res, err = runDoctor(os.Args[2:], start)
	case "verify":
		res, err = runVerify(os.Args[2:], start)
	case "status":
		res, err = runStatus(os.Args[2:], start)
	case "governance":
		res, err = runGovernance(os.Args[2:], start)
	case "powershell":
		res, err = runPowerShell(os.Args[2:], start)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsageAndExit(2)
	}
	if err != nil {
		if res.SchemaVersion == 0 {
			res = report.Result{SchemaVersion: 1, Command: cmd, OK: false, Message: err.Error(), ElapsedMS: report.Elapsed(start)}
		} else if res.Message == "" {
			res.Message = err.Error()
		}
	}
	if jsonRequested(os.Args[2:]) {
		report.WriteJSON(res)
	} else {
		report.WriteText(res)
	}
	if err != nil || !res.OK {
		os.Exit(1)
	}
}

func printUsageAndExit(code int) {
	fmt.Fprintf(os.Stderr, `AiCoding fast path CLI %s

Usage:
  aicoding hook pre-commit [--repo-root PATH] [--json]
  aicoding hook commit-msg --file COMMIT_MSG [--repo-root PATH] [--json]
  aicoding bootstrap [--repo-root PATH] [--json]

  aicoding smoke [--repo-root PATH] [--json]
  aicoding ci --profile Smoke|Full|Release [--repo-root PATH] [--json]
  aicoding test full|release [--repo-root PATH] [--timeout-sec N] [--long-timeout-sec N] [--concurrency N] [--json]
  aicoding test latest [--repo-root PATH] [--json]
  aicoding docsync staged|all|ci|release [--repo-root PATH] [--json]
  aicoding skill verify --all --profile Smoke|Full|Release [--repo-root PATH] [--json]
  aicoding skill c99-standard-c status [--repo-root PATH] [--json]
  aicoding skill c99-standard-c templates [--repo-root PATH] [--json]
  aicoding skill c99-standard-c fmt --scope changed|staged|all|paths [--path PATH ...] [--preview] [--repo-root PATH] [--json]
  aicoding skill c99-standard-c check --scope changed|staged|all|paths [--path PATH ...] [--repo-root PATH] [--json]
  aicoding lifecycle plan --action install|update|uninstall --all [--repo-root PATH] [--json]
  aicoding lifecycle install|update|uninstall --all [--repo-root PATH] [--json]
  aicoding lifecycle rollback --last [--repo-root PATH] [--json]
  aicoding export --all --zip [--repo-root PATH] [--json]
  aicoding fresh-clone --profile Smoke|Full|Release [--repo-root PATH] [--json]
  aicoding full [--repo-root PATH] [--json]
  aicoding cache status [--repo-root PATH] [--json]
  aicoding cache clean [--repo-root PATH] [--json]
  aicoding tag audit [--repo-root PATH] [--json]
  aicoding release verify [--repo-root PATH] [--json]
  aicoding release gate [--repo-root PATH] [--json]
  aicoding governance lint [--repo-root PATH] [--json]
  aicoding governance layout [--repo-root PATH] [--json]

  aicoding kit list [--repo-root PATH] [--json]
  aicoding kit verify --all --profile Smoke|Lifecycle [--repo-root PATH] [--json]
  aicoding kit doctor [--repo-root PATH] [--json]
  aicoding kit lifecycle --action install|update|uninstall --all --dry-run [--repo-root PATH] [--json]
  aicoding kit lifecycle --action status --all [--repo-root PATH] [--json]
  aicoding doctor perf [--repo-root PATH] [--json]
  aicoding doctor pwsh [--repo-root PATH] [--json]
  aicoding doctor pwsh-budget [--repo-root PATH] [--json]
  aicoding verify hooks [--repo-root PATH] [--json]
  aicoding verify repo-text [--repo-root PATH] [--json]
  aicoding verify release-notes [--repo-root PATH] [--json]
  aicoding status --all [--repo-root PATH] [--json]
  aicoding powershell regex-lint --staged [--repo-root PATH] [--json]
  aicoding powershell regex-lint --path PATH [--repo-root PATH] [--json]

This CLI owns Go-native fast, lifecycle, export, DocSync, fresh-clone, Full, and Release control paths.
`, version)
	os.Exit(code)
}

func jsonRequested(args []string) bool {
	for _, a := range args {
		if a == "--json" || a == "-json" || a == "-Json" {
			return true
		}
	}
	return false
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func runCStyleCommand(commandPrefix string, skillID string, args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, fmt.Errorf("%s requires subcommand: status, templates, fmt, or check", commandPrefix)
	}

	sub := args[0]
	fs := flag.NewFlagSet(commandPrefix+" "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	scopeArg := fs.String("scope", "changed", "changed, staged, all, or paths")
	previewArg := fs.Bool("preview", false, "preview formatting changes without writing files")
	_ = fs.Bool("json", false, "json output")

	var pathArgs multiFlag
	fs.Var(&pathArgs, "path", "explicit path for --scope paths; can be repeated")
	_ = fs.Parse(args[1:])

	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail(commandPrefix+" "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}

	switch sub {
	case "status":
		status, statusErr := cstyle.SkillStatus(repo, skillID)
		errs := []string{}
		if statusErr != nil {
			errs = append(errs, statusErr.Error())
		}
		if !status.ClangFormat.Found {
			errs = append(errs, "clang-format not found on PATH")
		}
		if status.SkillID != "" && !status.FormatterConfigExists {
			errs = append(errs, "formatter config not found: "+status.FormatterConfig)
		}
		elapsed := report.Elapsed(start)
		data := standardReport(commandPrefix+" status", skillID, elapsed, map[string]interface{}{
			"skill_id":                status.SkillID,
			"language":                status.Language,
			"standard":                status.Standard,
			"formatter":               status.Formatter,
			"formatter_config_exists": status.FormatterConfigExists,
			"templates_exists":        status.CommentTemplatesExists,
			"rules_exists":            status.RulesExists,
			"errors":                  len(errs),
		}, nil, errs, status)
		return report.Result{
			SchemaVersion: 1,
			Command:       commandPrefix + " status",
			OK:            len(errs) == 0,
			Message:       "C99 Standard C skill formatter status",
			RepoRoot:      repo,
			Data:          data,
			Errors:        errs,
			ElapsedMS:     elapsed,
		}, report.BoolErr(errs)

	case "templates":
		data, validationErr := cstyle.ValidateCommentTemplates(repo, skillID)
		elapsed := report.Elapsed(start)
		errs := append([]string{}, data.Errors...)
		standard := standardReport(commandPrefix+" templates", skillID, elapsed, map[string]interface{}{
			"path":      data.Path,
			"valid":     data.Valid,
			"templates": data.Count,
			"errors":    len(errs),
		}, nil, errs, data)
		return report.Result{
			SchemaVersion: 1,
			Command:       commandPrefix + " templates",
			OK:            validationErr == nil,
			Message:       "C99 Standard C skill comment templates validation",
			RepoRoot:      repo,
			Data:          standard,
			Errors:        data.Errors,
			ElapsedMS:     elapsed,
		}, validationErr

	case "fmt", "check":
		data, runErr := cstyle.RunBySkill(skillID, cstyle.Options{
			RepoRoot: repo,
			Scope:    cstyle.Scope(*scopeArg),
			Paths:    pathArgs,
			Check:    sub == "check",
			Preview:  *previewArg,
		})
		message := "C99 Standard C skill format completed"
		if sub == "check" {
			message = "C99 Standard C skill check completed"
		}
		elapsed := report.Elapsed(start)
		standard := standardReport(commandPrefix+" "+sub, skillID, elapsed, map[string]interface{}{
			"skill_id": data.SkillID,
			"scope":    string(data.Scope),
			"files":    len(data.Files),
			"changed":  len(data.Changed),
			"errors":   len(data.Errors),
		}, nil, data.Errors, data)
		return report.Result{
			SchemaVersion: 1,
			Command:       commandPrefix + " " + sub,
			OK:            runErr == nil,
			Message:       message,
			RepoRoot:      repo,
			Checked:       data.Files,
			Data:          standard,
			Errors:        data.Errors,
			ElapsedMS:     elapsed,
		}, runErr

	default:
		return report.Result{}, fmt.Errorf("unsupported %s subcommand: %s", commandPrefix, sub)
	}
}

func runHook(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("hook requires subcommand: pre-commit or commit-msg")
	}
	sub := args[0]
	switch sub {
	case "pre-commit":
		fs := flag.NewFlagSet("hook pre-commit", flag.ContinueOnError)
		repoArg := fs.String("repo-root", "", "repository root")
		_ = fs.Bool("json", false, "json output")
		_ = fs.Parse(args[1:])
		repo, err := platform.ResolveRepoRoot(*repoArg)
		if err != nil {
			return report.Fail("hook pre-commit", start, "cannot resolve repo root", nil, err.Error()), err
		}

		plan := runner.NewPlan(
			runner.Task{ID: "governance pre-commit", Group: "hook", Run: func(context.Context) runner.TaskResult {
				errs := governance.Lint(repo, "pre-commit", "")
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs}
			}},
			runner.Task{ID: "docsync staged", Group: "hook", Run: func(context.Context) runner.TaskResult {
				errs := docsync.LintStaged(repo)
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs}
			}},
			runner.Task{ID: "powershell regex staged", Group: "hook", Run: func(context.Context) runner.TaskResult {
				issues, scanErr := pwshregex.LintStaged(repo)
				errs := []string{}
				if scanErr != nil {
					errs = append(errs, scanErr.Error())
				}
				errs = append(errs, pwshregex.BlockingMessages(issues)...)
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: issues}
			}},
			runner.Task{ID: "c99-standard-c staged check", Group: "hook", Run: func(context.Context) runner.TaskResult {
				data, runErr := cstyle.CheckBySkill(cstyle.DefaultSkillID, cstyle.Options{RepoRoot: repo, Scope: cstyle.ScopeStaged})
				errs := append([]string{}, data.Errors...)
				if runErr != nil && len(errs) == 0 {
					errs = append(errs, runErr.Error())
				}
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: data}
			}},
		)
		results := plan.Run(context.Background(), runner.Options{MaxParallel: 4})
		errs := []string{}
		for _, result := range results {
			for _, e := range result.Errors {
				if e != "" {
					errs = append(errs, result.ID+": "+e)
				}
			}
		}
		return report.Result{SchemaVersion: 1, Command: "hook pre-commit", OK: len(errs) == 0, Message: "pre-commit Go gate", RepoRoot: repo, Data: results, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "commit-msg":
		fs := flag.NewFlagSet("hook commit-msg", flag.ContinueOnError)
		repoArg := fs.String("repo-root", "", "repository root")
		fileArg := fs.String("file", "", "commit message file")
		_ = fs.Bool("json", false, "json output")
		_ = fs.Parse(args[1:])
		if *fileArg == "" && fs.NArg() > 0 {
			*fileArg = fs.Arg(0)
		}
		repo, err := platform.ResolveRepoRoot(*repoArg)
		if err != nil {
			return report.Fail("hook commit-msg", start, "cannot resolve repo root", nil, err.Error()), err
		}
		errs := governance.Lint(repo, "commit-msg", *fileArg)
		return report.Result{SchemaVersion: 1, Command: "hook commit-msg", OK: len(errs) == 0, Message: "commit-msg fast gate", RepoRoot: repo, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	default:
		return report.Result{}, fmt.Errorf("unsupported hook: %s", sub)
	}
}
func runGovernance(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("governance requires subcommand: lint or layout")
	}
	sub := args[0]
	fs := flag.NewFlagSet("governance "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	mode := fs.String("mode", "all", "all or pre-commit; lint only")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("governance "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	switch sub {
	case "lint":
		errs := governance.Lint(repo, *mode, "")
		return report.Result{SchemaVersion: 1, Command: "governance lint", OK: len(errs) == 0, Message: "governance fast lint", RepoRoot: repo, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "layout":
		layout := governance.CheckLayout(repo)
		return report.Result{SchemaVersion: 1, Command: "governance layout", OK: len(layout.Errors) == 0, Message: "repository layout gate", RepoRoot: repo, Data: layout, Errors: layout.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(layout.Errors)
	default:
		return report.Result{}, fmt.Errorf("unsupported governance subcommand: %s", sub)
	}
}

func runPowerShell(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "regex-lint" {
		return report.Result{}, errors.New("powershell requires subcommand: regex-lint")
	}
	fs := flag.NewFlagSet("powershell regex-lint", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	pathArg := fs.String("path", "", "file or directory to scan")
	stagedArg := fs.Bool("staged", false, "scan staged PowerShell files")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])

	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("powershell regex-lint", start, "cannot resolve repo root", nil, err.Error()), err
	}

	var issues []pwshregex.Issue
	if *stagedArg {
		issues, err = pwshregex.LintStaged(repo)
	} else {
		target := *pathArg
		if target == "" && fs.NArg() > 0 {
			target = fs.Arg(0)
		}
		if target == "" {
			return report.Fail("powershell regex-lint", start, "path is required unless --staged is used", nil, "missing --path"), errors.New("missing --path")
		}
		if !filepath.IsAbs(target) {
			target = filepath.ToSlash(target)
		}
		issues, err = pwshregex.LintPath(repo, target)
	}
	if err != nil {
		return report.Fail("powershell regex-lint", start, "regex lint failed", issues, err.Error()), err
	}

	errs := pwshregex.BlockingMessages(issues)
	return report.Result{
		SchemaVersion: 1,
		Command:       "powershell regex-lint",
		OK:            len(errs) == 0,
		Message:       "PowerShell regex optimization fast lint",
		RepoRoot:      repo,
		Data:          issues,
		Errors:        errs,
		ElapsedMS:     report.Elapsed(start),
	}, report.BoolErr(errs)
}

func runKit(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("kit requires subcommand")
	}
	sub := args[0]
	fs := flag.NewFlagSet("kit "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	actionArg := fs.String("action", "", "lifecycle action: install, update, uninstall, or status")
	dryRunArg := fs.Bool("dry-run", false, "plan lifecycle action without executing adapters")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("kit "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		return report.Fail("kit "+sub, start, "cannot load registry", nil, err.Error()), err
	}
	withManifests := kit.LoadKitViews(repo, entries)
	switch sub {
	case "list":
		return report.Result{SchemaVersion: 1, Command: "kit list", OK: true, Message: "kit registry", RepoRoot: repo, Data: withManifests, ElapsedMS: report.Elapsed(start)}, nil
	case "doctor":
		errs := kit.DoctorKits(repo, entries)
		return report.Result{SchemaVersion: 1, Command: "kit doctor", OK: len(errs) == 0, Message: "kit registry doctor", RepoRoot: repo, Data: withManifests, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "lifecycle":
		action := strings.ToLower(*actionArg)
		switch action {
		case "install", "update", "uninstall", "status":
			// supported planner actions
		case "":
			return report.Fail("kit lifecycle", start, "missing lifecycle action", nil, "use --action install|update|uninstall|status"), errors.New("missing lifecycle action")
		default:
			return report.Fail("kit lifecycle", start, "unsupported lifecycle action", nil, action), fmt.Errorf("unsupported lifecycle action: %s", action)
		}
		if action != "status" && !*dryRunArg {
			return report.Fail("kit lifecycle", start, "use top-level lifecycle install/update/uninstall for real Go lifecycle actions", nil, "use aicoding lifecycle "+action+" --all --json"), errors.New("use top-level lifecycle command")
		}
		selected, err := kit.SelectKits(entries, *kitArg, *allArg)
		if err != nil {
			return report.Fail("kit lifecycle", start, "kit selection failed", nil, err.Error()), err
		}
		mode := "kit"
		if *allArg {
			mode = "all"
		}
		plan := kit.PlanLifecycle(repo, selected, kit.LifecycleOptions{Action: action, Mode: mode, DryRun: *dryRunArg})
		errs := []string{}
		for _, item := range plan.Kits {
			if !item.OK {
				reason := item.Reason
				if reason == "" {
					reason = item.Status
				}
				errs = append(errs, item.ID+": "+reason)
			}
		}
		message := "kit lifecycle planner"
		if *dryRunArg {
			message = "kit lifecycle dry-run planner"
		}
		return report.Result{SchemaVersion: 1, Command: "kit lifecycle", OK: len(errs) == 0, Message: message, RepoRoot: repo, Data: plan, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "verify", "test":
		selected, err := kit.SelectKits(entries, *kitArg, *allArg)
		if err != nil {
			return report.Fail("kit "+sub, start, "kit selection failed", nil, err.Error()), err
		}
		if strings.EqualFold(*profile, "Lifecycle") {
			if sub != "verify" {
				return report.Fail("kit "+sub, start, "Lifecycle profile only supports kit verify", nil, "use kit verify --profile Lifecycle"), errors.New("Lifecycle profile only supports kit verify")
			}
			structure := kit.VerifyStructure(repo, selected)
			return report.Result{SchemaVersion: 1, Command: "kit verify", OK: structure.OK, Message: "kit lifecycle structure verify", RepoRoot: repo, Data: structure, Errors: structure.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(structure.Errors)
		}
		if !strings.EqualFold(*profile, "Smoke") {
			return report.Fail("kit "+sub, start, "kit command handles Smoke/Lifecycle only; use skill verify or full/release for broader Go gates", nil, "use aicoding skill verify --all --profile "+*profile+" --json"), errors.New("non-Smoke kit profile is not handled by kit subcommand")
		}
		results := kit.SmokeKits(repo, selected)
		errs := []string{}
		for _, r := range results {
			if !r.OK {
				for _, e := range r.Errors {
					errs = append(errs, r.ID+": "+e)
				}
			}
		}
		return report.Result{SchemaVersion: 1, Command: "kit " + sub, OK: len(errs) == 0, Message: "kit smoke " + sub, RepoRoot: repo, Data: results, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	default:
		return report.Result{}, fmt.Errorf("unsupported kit subcommand: %s", sub)
	}
}

func runVerify(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("verify requires subcommand: hooks, repo-text, or release-notes")
	}
	sub := args[0]
	fs := flag.NewFlagSet("verify "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("verify "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	switch sub {
	case "hooks":
		checks, errs := repohealth.VerifyHooks(repo)
		return report.Result{SchemaVersion: 1, Command: "verify hooks", OK: len(errs) == 0, Message: "hook fast path verification", RepoRoot: repo, Data: checks, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "repo-text":
		checks, errs := repohealth.VerifyRepoText(repo)
		return report.Result{SchemaVersion: 1, Command: "verify repo-text", OK: len(errs) == 0, Message: "repository text verification", RepoRoot: repo, Data: checks, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "release-notes":
		checks, errs := repohealth.VerifyReleaseNotes(repo)
		return report.Result{SchemaVersion: 1, Command: "verify release-notes", OK: len(errs) == 0, Message: "release notes and tag policy verification", RepoRoot: repo, Data: checks, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	default:
		return report.Result{}, fmt.Errorf("unsupported verify subcommand: %s", sub)
	}
}

func runStatus(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	allArg := fs.Bool("all", false, "summarize all local fast-path state")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	if !*allArg {
		return report.Result{}, errors.New("status requires --all")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("status", start, "cannot resolve repo root", nil, err.Error()), err
	}
	status, errs := repohealth.StatusAll(repo)
	return report.Result{SchemaVersion: 1, Command: "status --all", OK: len(errs) == 0, Message: "fast path repository status", RepoRoot: repo, Data: status, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}
func runDoctor(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("doctor requires subcommand: perf, pwsh, or pwsh-budget")
	}
	sub := args[0]
	fs := flag.NewFlagSet("doctor "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("doctor "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	if sub == "pwsh" {
		calls, errs := repohealth.ScanPwsh(repo)
		return report.Result{SchemaVersion: 1, Command: "doctor pwsh", OK: len(errs) == 0, Message: "PowerShell invocation inventory", RepoRoot: repo, Data: calls, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	}
	if sub == "pwsh-budget" {
		budget, errs := repohealth.ScanPwshBudget(repo)
		return report.Result{SchemaVersion: 1, Command: "doctor pwsh-budget", OK: len(errs) == 0, Message: "PowerShell budget inventory", RepoRoot: repo, Data: budget, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	}
	if sub != "perf" {
		return report.Result{}, fmt.Errorf("unsupported doctor subcommand: %s", sub)
	}
	checks := []map[string]interface{}{}
	measure := func(name string, fn func() error) {
		t0 := time.Now()
		err := fn()
		item := map[string]interface{}{"name": name, "elapsedMs": time.Since(t0).Milliseconds(), "ok": err == nil}
		if err != nil {
			item["error"] = err.Error()
		}
		checks = append(checks, item)
	}
	measure("git rev-parse", func() error { _, e := gitx.Run(repo, "rev-parse", "--show-toplevel"); return e })
	measure("git diff cached names", func() error { _, e := gitx.StagedFiles(repo); return e })
	measure("load kit registry", func() error { _, e := kit.LoadRegistry(repo); return e })
	measure("governance lint", func() error { return report.BoolErr(governance.Lint(repo, "pre-commit", "")) })
	measure("staged docsync lint", func() error { return report.BoolErr(docsync.LintStaged(repo)) })
	return report.Result{SchemaVersion: 1, Command: "doctor perf", OK: true, Message: "performance probes", RepoRoot: repo, Data: checks, ElapsedMS: report.Elapsed(start)}, nil
}

func runBootstrap(args []string, start time.Time) (report.Result, error) {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	noBuild := fs.Bool("no-build", false, "check and create bin directory without building")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args)
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("bootstrap", start, "cannot resolve repo root", nil, err.Error()), err
	}
	status, errs := bootstrap.Bootstrap(repo, bootstrap.Options{Build: !*noBuild})
	return report.Result{SchemaVersion: 1, Command: "bootstrap", OK: len(errs) == 0, Message: "bootstrap fast path binary", RepoRoot: repo, Data: status, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runCache(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, errors.New("cache requires subcommand: status or clean")
	}
	sub := args[0]
	fs := flag.NewFlagSet("cache "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("cache "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	switch sub {
	case "status":
		status, err := cache.Status(repo)
		if err != nil {
			return report.Fail("cache status", start, "cache status failed", status, err.Error()), err
		}
		return report.Result{SchemaVersion: 1, Command: "cache status", OK: true, Message: "fast path cache status", RepoRoot: repo, Data: status, ElapsedMS: report.Elapsed(start)}, nil
	case "clean":
		result, err := cache.Clean(repo)
		if err != nil {
			return report.Fail("cache clean", start, "cache clean failed", result, err.Error()), err
		}
		return report.Result{SchemaVersion: 1, Command: "cache clean", OK: true, Message: "fast path cache clean", RepoRoot: repo, Data: result, ElapsedMS: report.Elapsed(start)}, nil
	default:
		return report.Result{}, fmt.Errorf("unsupported cache subcommand: %s", sub)
	}
}

func runTag(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "audit" {
		return report.Result{}, errors.New("tag requires subcommand: audit")
	}
	fs := flag.NewFlagSet("tag audit", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("tag audit", start, "cannot resolve repo root", nil, err.Error()), err
	}
	audit, errs := tagpolicy.AuditRepo(repo)
	return report.Result{SchemaVersion: 1, Command: "tag audit", OK: len(errs) == 0, Message: "tag namespace structural audit", RepoRoot: repo, Data: audit, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runRelease(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "verify" {
		return report.Result{}, errors.New("release requires subcommand: verify")
	}
	fs := flag.NewFlagSet("release verify", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("release verify", start, "cannot resolve repo root", nil, err.Error()), err
	}
	result, errs := releasegate.Verify(repo)
	return report.Result{SchemaVersion: 1, Command: "release verify", OK: len(errs) == 0, Message: "release structural fast verification", RepoRoot: repo, Data: result, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}
