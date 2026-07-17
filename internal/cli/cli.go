package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/pwshregex"
	"github.com/JiaxI2/AiCoding/internal/releasegate"
	"github.com/JiaxI2/AiCoding/internal/repohealth"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/reuse"
	"github.com/JiaxI2/AiCoding/internal/runner"
	"github.com/JiaxI2/AiCoding/internal/tagpolicy"
)

var buildVersion string

func Main() {
	if code := Execute(os.Args[1:], os.Stdout, os.Stderr); code != ExitSuccess {
		os.Exit(code)
	}
}

func Execute(args []string, stdout io.Writer, stderr io.Writer) int {
	start := time.Now()
	if len(args) < 1 {
		writeUsage(stderr)
		return ExitUsage
	}
	requestedCommand := args[0]
	commandArgs := args[1:]
	route, known := commands.lookup(requestedCommand)
	if !known {
		if jsonRequested(commandArgs) {
			message := "unknown command: " + requestedCommand
			res := report.Fail(requestedCommand, start, message, nil, message)
			res.ErrorKind = report.ErrorKindUsage
			_ = report.WriteJSONTo(stdout, res)
			return ExitUsage
		}
		fmt.Fprintf(stderr, "unknown command: %s\n", requestedCommand)
		writeUsage(stderr)
		return ExitUsage
	}
	cmd := route.descriptor.Name
	if len(commandArgs) == 1 && isHelpArg(commandArgs[0]) && route.descriptor.RequiresSubcommand {
		writeUsage(stdout)
		return ExitSuccess
	}
	if route.direct == directHelp {
		writeUsage(stdout)
		return ExitSuccess
	}
	if route.direct == directVersion {
		fmt.Fprintln(stdout, productVersion())
		return ExitSuccess
	}
	var res report.Result
	var err error
	if route.handler == nil {
		err = usageErrorf("command route is unavailable: %s", cmd)
	} else {
		res, err = route.handler(commandArgs, start)
	}
	if help, ok := requestedHelp(err); ok {
		fmt.Fprint(stdout, help)
		return ExitSuccess
	}
	if err != nil {
		if res.SchemaVersion == 0 {
			res = report.Fail(cmd, start, err.Error(), nil, err.Error())
		} else if res.Message == "" {
			res.Message = err.Error()
		}
		if len(res.Errors) == 0 {
			res.Errors = []string{err.Error()}
		}
	}
	switch {
	case isUsageError(err):
		res.ErrorKind = report.ErrorKindUsage
	case report.IsValidationError(err):
		res.ErrorKind = report.ErrorKindValidation
	case res.ErrorKind == "" && !res.OK:
		res.ErrorKind = report.ErrorKindValidation
	case res.ErrorKind == "" && err != nil:
		res.ErrorKind = report.ErrorKindExecution
	}
	if canonical, ok := deprecatedCommand(args); ok {
		res = addDeprecation(res, canonical)
	}
	if jsonRequested(commandArgs) {
		_ = report.WriteJSONTo(stdout, res)
	} else {
		report.WriteTextTo(stdout, res)
		if cmd == "codex" {
			writeCodexUsageText(stdout, res)
		}
	}
	return exitCodeFor(res, err)
}

func productVersion() string {
	if buildVersion != "" {
		return buildVersion
	}
	candidates := []string{filepath.Join("config", "codex-kit.json")}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "..", "config", "codex-kit.json"))
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var metadata struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &metadata); err == nil && metadata.Version != "" {
			return metadata.Version
		}
	}
	return "development"
}

func writeUsage(w io.Writer) {
	writeCatalogHelp(w)
}

func jsonRequested(args []string) bool {
	for _, a := range args {
		if a == "--" {
			break
		}
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
		return report.Result{}, usageErrorf("%s requires subcommand: status, templates, fmt, check, or verify", commandPrefix)
	}

	sub := args[0]
	if !validChoice(sub, "status", "templates", "fmt", "check", "verify") {
		return report.Result{}, usageErrorf("unsupported %s subcommand: %s", commandPrefix, sub)
	}
	fs := newFlagSet(commandPrefix + " " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	scopeArg := fs.String("scope", "changed", "changed, staged, all, or paths")
	previewArg := fs.Bool("preview", false, "preview formatting changes without writing files")
	profileArg := fs.String("profile", "fast", "C Kit verification profile: fast or full")
	targetArg := fs.String("target", "", "C Kit verification target manifest")
	timingsArg := fs.Bool("timings", false, "include C Kit per-step timings")
	_ = fs.Bool("json", false, "json output")

	var pathArgs multiFlag
	var overlayArgs multiFlag
	fs.Var(&pathArgs, "path", "explicit path for --scope paths; can be repeated")
	fs.Var(&overlayArgs, "overlay", "C Kit partial configuration overlay; can be repeated")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}

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
			"kit_id":                  status.KitID,
			"kit_version":             status.KitVersion,
			"kit_root_exists":         status.KitRootExists,
			"kit_config_exists":       status.KitConfigExists,
			"kit_snippets_exists":     status.KitSnippetsExists,
			"kit_quick_target_exists": status.KitQuickTargetExists,
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

	case "verify":
		data, verifyErr := cstyle.VerifyBySkill(skillID, cstyle.VerifyOptions{
			RepoRoot: repo,
			Profile:  *profileArg,
			Target:   *targetArg,
			Overlays: overlayArgs,
			Timings:  *timingsArg,
		})
		elapsed := report.Elapsed(start)
		errs := []string{}
		if verifyErr != nil {
			errs = append(errs, verifyErr.Error())
		}
		standard := standardReport(commandPrefix+" verify", data.Profile, elapsed, map[string]interface{}{
			"skill_id": data.SkillID,
			"kit_id":   data.KitID,
			"profile":  data.Profile,
			"target":   data.Target,
			"overlays": len(data.Overlays),
			"errors":   len(errs),
		}, nil, errs, data)
		return report.Result{
			SchemaVersion: 1,
			Command:       commandPrefix + " verify",
			OK:            verifyErr == nil,
			Message:       "C99 Standard C skill C Kit verification",
			RepoRoot:      repo,
			Data:          standard,
			Errors:        errs,
			ElapsedMS:     elapsed,
		}, verifyErr

	default:
		return report.Result{}, usageErrorf("unsupported %s subcommand: %s", commandPrefix, sub)
	}
}

func runHook(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("hook requires subcommand: pre-commit or commit-msg")
	}
	sub := args[0]
	switch sub {
	case "pre-commit":
		fs := newFlagSet("hook pre-commit")
		repoArg := fs.String("repo-root", "", "repository root")
		_ = fs.Bool("json", false, "json output")
		if err := parseNoPositionals(fs, args[1:]); err != nil {
			return report.Result{}, err
		}
		repo, err := platform.ResolveRepoRoot(*repoArg)
		if err != nil {
			return report.Fail("hook pre-commit", start, "cannot resolve repo root", nil, err.Error()), err
		}

		plan, planErr := runner.NewExecutionPlan(
			runner.Task{ID: "governance pre-commit", Action: "governance.lint", Group: "hook", Run: func(context.Context) runner.TaskResult {
				errs := governance.Lint(repo, "pre-commit", "")
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs}
			}},
			runner.Task{ID: "docsync staged", Action: "docsync.lint-staged", Group: "hook", Run: func(context.Context) runner.TaskResult {
				errs := docsync.LintStaged(repo)
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs}
			}},
			runner.Task{ID: "reuse governance", Action: "reuse.verify", Group: "hook", Run: func(context.Context) runner.TaskResult {
				check := reuse.Verify(repo)
				return runner.TaskResult{OK: check.OK, Errors: check.Errors, Warnings: check.Warnings, Data: check}
			}},
			runner.Task{ID: "powershell regex staged", Action: "powershell.regex-lint-staged", Group: "hook", Run: func(context.Context) runner.TaskResult {
				issues, scanErr := pwshregex.LintStaged(repo)
				errs := []string{}
				if scanErr != nil {
					errs = append(errs, scanErr.Error())
				}
				errs = append(errs, pwshregex.BlockingMessages(issues)...)
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: issues}
			}},
			runner.Task{ID: "c99-standard-c staged check", Action: "c99-standard-c.check-staged", Group: "hook", Run: func(context.Context) runner.TaskResult {
				data, runErr := cstyle.CheckBySkill(cstyle.DefaultSkillID, cstyle.Options{RepoRoot: repo, Scope: cstyle.ScopeStaged})
				errs := append([]string{}, data.Errors...)
				if runErr != nil && len(errs) == 0 {
					errs = append(errs, runErr.Error())
				}
				return runner.TaskResult{OK: len(errs) == 0, Errors: errs, Data: data}
			}},
		)
		if planErr != nil {
			return report.Fail("hook pre-commit", start, "cannot build execution plan", nil, planErr.Error()), planErr
		}
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
		fs := newFlagSet("hook commit-msg")
		repoArg := fs.String("repo-root", "", "repository root")
		fileArg := fs.String("file", "", "commit message file")
		_ = fs.Bool("json", false, "json output")
		if err := parseFlags(fs, args[1:]); err != nil {
			return report.Result{}, err
		}
		if fs.NArg() > 1 || (*fileArg != "" && fs.NArg() != 0) {
			return report.Result{}, usageErrorf("hook commit-msg accepts one message file through --file or a positional argument")
		}
		if *fileArg == "" && fs.NArg() > 0 {
			*fileArg = fs.Arg(0)
		}
		if *fileArg == "" {
			return report.Result{}, usageErrorf("hook commit-msg requires --file COMMIT_MSG")
		}
		repo, err := platform.ResolveRepoRoot(*repoArg)
		if err != nil {
			return report.Fail("hook commit-msg", start, "cannot resolve repo root", nil, err.Error()), err
		}
		errs := governance.Lint(repo, "commit-msg", *fileArg)
		return report.Result{SchemaVersion: 1, Command: "hook commit-msg", OK: len(errs) == 0, Message: "commit-msg fast gate", RepoRoot: repo, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	default:
		return report.Result{}, usageErrorf("unsupported hook: %s", sub)
	}
}
func runGovernance(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("governance requires subcommand: lint, dependencies, layout, or reuse")
	}
	sub := args[0]
	if !validChoice(sub, "lint", "dependencies", "layout", "reuse") {
		return report.Result{}, usageErrorf("unsupported governance subcommand: %s", sub)
	}
	fs := newFlagSet("governance " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	mode := fs.String("mode", "all", "all or pre-commit; lint only")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("governance "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	switch sub {
	case "lint":
		errs := governance.Lint(repo, *mode, "")
		return report.Result{SchemaVersion: 1, Command: "governance lint", OK: len(errs) == 0, Message: "governance fast lint", RepoRoot: repo, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "dependencies":
		dependencies := governance.CheckDependencies(repo)
		return report.Result{SchemaVersion: 1, Command: "governance dependencies", OK: len(dependencies.Errors) == 0, Message: "dependency direction governance gate", RepoRoot: repo, Data: dependencies, Warnings: dependencies.Warnings, Errors: dependencies.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(dependencies.Errors)
	case "layout":
		layout := governance.CheckLayout(repo)
		return report.Result{SchemaVersion: 1, Command: "governance layout", OK: len(layout.Errors) == 0, Message: "repository layout gate", RepoRoot: repo, Data: layout, Errors: layout.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(layout.Errors)
	case "reuse":
		check := reuse.Verify(repo)
		return report.Result{SchemaVersion: 1, Command: "governance reuse", OK: check.OK, Message: "reuse governance evidence gate", RepoRoot: repo, Data: check, Warnings: check.Warnings, Errors: check.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(check.Errors)
	default:
		return report.Result{}, usageErrorf("unsupported governance subcommand: %s", sub)
	}
}

func runPowerShell(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "regex-lint" {
		return report.Result{}, usageErrorf("powershell requires subcommand: regex-lint")
	}
	fs := newFlagSet("powershell regex-lint")
	repoArg := fs.String("repo-root", "", "repository root")
	pathArg := fs.String("path", "", "file or directory to scan")
	stagedArg := fs.Bool("staged", false, "scan staged PowerShell files")
	_ = fs.Bool("json", false, "json output")
	if err := parseFlags(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	if fs.NArg() > 1 || (*pathArg != "" && fs.NArg() != 0) {
		return report.Result{}, usageErrorf("powershell regex-lint accepts one path through --path or a positional argument")
	}

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
			return report.Result{}, usageErrorf("path is required unless --staged is used")
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
		return report.Result{}, usageErrorf("kit requires subcommand")
	}
	sub := args[0]
	if !validChoice(sub, "list", "doctor", "lifecycle", "verify", "test") {
		return report.Result{}, usageErrorf("unsupported kit subcommand: %s", sub)
	}
	fs := newFlagSet("kit " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	actionArg := fs.String("action", "", "lifecycle action: install, update, uninstall, or status")
	dryRunArg := fs.Bool("dry-run", false, "plan lifecycle action without executing adapters")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("kit "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	if sub == "list" {
		catalog, loadErr := kit.LoadCatalogSnapshot(repo)
		if loadErr != nil {
			return report.Fail("kit list", start, "cannot load kit catalog", nil, loadErr.Error()), loadErr
		}
		return report.Result{
			SchemaVersion: 1,
			Command:       "kit list",
			OK:            true,
			Message:       "kit catalog",
			RepoRoot:      repo,
			InputDigest:   catalog.Digest(),
			Data:          kit.CatalogKitViews(catalog.Kits()),
			ElapsedMS:     report.Elapsed(start),
		}, nil
	}
	entries, err := kit.LoadRegistry(repo)
	if err != nil {
		return report.Fail("kit "+sub, start, "cannot load registry", nil, err.Error()), err
	}
	withManifests := kit.LoadKitViews(repo, entries)
	switch sub {
	case "doctor":
		errs := kit.DoctorKits(repo, entries)
		return report.Result{SchemaVersion: 1, Command: "kit doctor", OK: len(errs) == 0, Message: "kit registry doctor", RepoRoot: repo, Data: withManifests, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	case "lifecycle":
		action := strings.ToLower(*actionArg)
		switch action {
		case "install", "update", "uninstall", "status":
			// supported planner actions
		case "":
			return report.Result{}, usageErrorf("kit lifecycle requires --action install|update|uninstall|status")
		default:
			return report.Result{}, usageErrorf("unsupported lifecycle action: %s", action)
		}
		if action != "status" && !*dryRunArg {
			return report.Result{}, usageErrorf("use aicoding lifecycle %s --all for real lifecycle actions", action)
		}
		_, err := kit.SelectKits(entries, *kitArg, *allArg)
		if err != nil {
			return report.Fail("kit lifecycle", start, "kit selection failed", nil, err.Error()), err
		}
		unified := lifecyclecontrol.Run(context.Background(), repo, lifecyclecontrol.Options{
			Action: action,
			Scope:  lifecyclecontrol.ScopeKit,
			All:    *allArg,
			KitID:  *kitArg,
			DryRun: *dryRunArg,
		})
		message := "kit lifecycle planner"
		if *dryRunArg {
			message = "kit lifecycle dry-run planner"
		}
		return report.Result{
			SchemaVersion: 1,
			Command:       "kit lifecycle",
			OK:            unified.OK,
			Message:       message,
			RepoRoot:      repo,
			InputDigest:   lifecycleAdapterInputDigest(unified, lifecyclecontrol.ScopeKit),
			PlanDigest:    unified.PlanDigest,
			Data:          lifecycleAdapterData(unified, lifecyclecontrol.ScopeKit),
			Warnings:      unified.Warnings,
			Errors:        unified.Errors,
			ElapsedMS:     report.Elapsed(start),
		}, report.BoolErr(unified.Errors)
	case "verify", "test":
		catalog, loadErr := kit.LoadCatalogSnapshot(repo)
		if loadErr != nil {
			return report.Fail("kit "+sub, start, "cannot load kit catalog", nil, loadErr.Error()), loadErr
		}
		selected, err := catalog.Select(*kitArg, *allArg)
		if err != nil {
			return report.Fail("kit "+sub, start, "kit selection failed", nil, err.Error()), err
		}
		if strings.EqualFold(*profile, "Lifecycle") {
			if sub != "verify" {
				return report.Result{}, usageErrorf("Lifecycle profile only supports kit verify")
			}
			structure := kit.VerifyCatalogStructure(repo, selected)
			return report.Result{SchemaVersion: 1, Command: "kit verify", OK: structure.OK, Message: "kit lifecycle structure verify", RepoRoot: repo, InputDigest: catalog.Digest(), Data: structure, Errors: structure.Errors, ElapsedMS: report.Elapsed(start)}, report.BoolErr(structure.Errors)
		}
		if !strings.EqualFold(*profile, "Smoke") {
			return report.Result{}, usageErrorf("kit %s handles Smoke/Lifecycle only; use aicoding skill verify --all --profile %s", sub, *profile)
		}
		results := kit.SmokeCatalogKits(repo, selected)
		errs := []string{}
		for _, r := range results {
			if !r.OK {
				for _, e := range r.Errors {
					errs = append(errs, r.ID+": "+e)
				}
			}
		}
		return report.Result{SchemaVersion: 1, Command: "kit " + sub, OK: len(errs) == 0, Message: "kit smoke " + sub, RepoRoot: repo, InputDigest: catalog.Digest(), Data: results, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
	default:
		return report.Result{}, usageErrorf("unsupported kit subcommand: %s", sub)
	}
}

func runVerify(args []string, start time.Time) (report.Result, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runProductVerify(args, start)
	}
	sub := args[0]
	if !validChoice(sub, "hooks", "repo-text", "release-notes") {
		return report.Result{}, usageErrorf("unsupported verify subcommand: %s", sub)
	}
	fs := newFlagSet("verify " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
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
		return report.Result{}, usageErrorf("unsupported verify subcommand: %s", sub)
	}
}

func runStatus(args []string, start time.Time) (report.Result, error) {
	return runProductDoctor(args, start, "status --all")
}
func runDoctor(args []string, start time.Time) (report.Result, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runProductDoctor(args, start, "doctor --all")
	}
	sub := args[0]
	if !validChoice(sub, "perf", "pwsh", "pwsh-budget") {
		return report.Result{}, usageErrorf("unsupported doctor subcommand: %s", sub)
	}
	fs := newFlagSet("doctor " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
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
		return report.Result{}, usageErrorf("unsupported doctor subcommand: %s", sub)
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
	fs := newFlagSet("bootstrap")
	repoArg := fs.String("repo-root", "", "repository root")
	noBuild := fs.Bool("no-build", false, "check and create bin directory without building")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("bootstrap", start, "cannot resolve repo root", nil, err.Error()), err
	}
	status, errs := bootstrap.Bootstrap(repo, bootstrap.Options{Build: !*noBuild})
	return report.Result{SchemaVersion: 1, Command: "bootstrap", OK: len(errs) == 0, Message: "bootstrap fast path binary", RepoRoot: repo, Data: status, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runCache(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("cache requires subcommand: status or clean")
	}
	sub := args[0]
	if !validChoice(sub, "status", "clean") {
		return report.Result{}, usageErrorf("unsupported cache subcommand: %s", sub)
	}
	fs := newFlagSet("cache " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
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
		return report.Result{}, usageErrorf("unsupported cache subcommand: %s", sub)
	}
}

func runTag(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "audit" {
		return report.Result{}, usageErrorf("tag requires subcommand: audit")
	}
	fs := newFlagSet("tag audit")
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("tag audit", start, "cannot resolve repo root", nil, err.Error()), err
	}
	audit, errs := tagpolicy.AuditRepo(repo)
	return report.Result{SchemaVersion: 1, Command: "tag audit", OK: len(errs) == 0, Message: "tag namespace structural audit", RepoRoot: repo, Data: audit, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}

func runRelease(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "verify" {
		return report.Result{}, usageErrorf("release requires subcommand: verify")
	}
	fs := newFlagSet("release verify")
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("release verify", start, "cannot resolve repo root", nil, err.Error()), err
	}
	result, errs := releasegate.Verify(repo)
	return report.Result{SchemaVersion: 1, Command: "release verify", OK: len(errs) == 0, Message: "release structural fast verification", RepoRoot: repo, Data: result, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
}
