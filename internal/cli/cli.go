package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/governance"
	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

const version = "fast-path-v1"

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
	case "kit":
		res, err = runKit(os.Args[2:], start)
	case "doctor":
		res, err = runDoctor(os.Args[2:], start)
	case "governance":
		res, err = runGovernance(os.Args[2:], start)
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
  aicoding governance lint [--repo-root PATH] [--json]
  aicoding kit list [--repo-root PATH] [--json]
  aicoding kit verify --all --profile Smoke [--repo-root PATH] [--json]
  aicoding kit doctor [--repo-root PATH] [--json]
  aicoding doctor perf [--repo-root PATH] [--json]

This v1 CLI intentionally accelerates hot-path governance, staged DocSync and Smoke checks.
Full/Release gates remain in PowerShell/Python and CI.
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
		errs := append(governance.Lint(repo, "pre-commit", ""), docsync.LintStaged(repo)...)
		return report.Result{SchemaVersion: 1, Command: "hook pre-commit", OK: len(errs) == 0, Message: "pre-commit fast gate", RepoRoot: repo, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
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
	if len(args) < 1 || args[0] != "lint" {
		return report.Result{}, errors.New("governance requires subcommand: lint")
	}
	fs := flag.NewFlagSet("governance lint", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	mode := fs.String("mode", "all", "all or pre-commit")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("governance lint", start, "cannot resolve repo root", nil, err.Error()), err
	}
	errs := governance.Lint(repo, *mode, "")
	return report.Result{SchemaVersion: 1, Command: "governance lint", OK: len(errs) == 0, Message: "governance fast lint", RepoRoot: repo, Errors: errs, ElapsedMS: report.Elapsed(start)}, report.BoolErr(errs)
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
	case "verify", "test":
		if !strings.EqualFold(*profile, "Smoke") {
			return report.Fail("kit "+sub, start, "fast CLI only handles Smoke profile", nil, "use scripts/aicoding-kit.ps1 for Full/Release"), errors.New("non-Smoke profile is not handled by fast CLI")
		}
		selected, err := kit.SelectKits(entries, *kitArg, *allArg)
		if err != nil {
			return report.Fail("kit "+sub, start, "kit selection failed", nil, err.Error()), err
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

func runDoctor(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 || args[0] != "perf" {
		return report.Result{}, errors.New("doctor requires subcommand: perf")
	}
	fs := flag.NewFlagSet("doctor perf", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("doctor perf", start, "cannot resolve repo root", nil, err.Error()), err
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
