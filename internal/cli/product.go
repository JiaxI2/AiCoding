package cli

import (
	"context"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/repohealth"
	"github.com/JiaxI2/AiCoding/internal/report"
)

var productDoctorChecks = repohealth.DoctorAll
var productVerifyChecks = repohealth.VerifyAll

func runProductDoctor(args []string, start time.Time, outerCommand string) (report.Result, error) {
	fs := newFlagSet("doctor")
	repoArg := fs.String("repo-root", "", "repository root")
	allArg := fs.Bool("all", false, "run all product diagnostics")
	codexConfigArg := fs.String("codex-config", "", "Codex config.toml path")
	runtimeProfileArg := fs.String("runtime-profile", "", "expected runtime Skill profile: runtime, full, or skill-development")
	runtimeSkillArg := fs.String("runtime-skill", "", "selected canonical Skill for skill-development")
	sourceRepositoryArg := fs.String("source-repository", "", "Codex-Skills source repository")
	standaloneRootArg := fs.String("standalone-root", "agents", "agents or codex")
	timeoutSecArg := fs.Int("timeout-sec", 180, "overall product diagnostic timeout seconds")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if !*allArg {
		return report.Result{}, usageErrorf("doctor requires --all")
	}
	if *timeoutSecArg < 1 {
		return report.Result{}, usageErrorf("doctor --timeout-sec must be positive")
	}
	runtimeProfile, standaloneRoot, err := productRuntimeSelection(*runtimeProfileArg, *runtimeSkillArg, *standaloneRootArg)
	if err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail(outerCommand, start, "cannot resolve repo root", nil, err.Error()), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSecArg)*time.Second)
	defer cancel()
	checks := productDoctorChecks(ctx, repo, repohealth.ProductOptions{
		CodexConfig:      *codexConfigArg,
		RuntimeProfile:   runtimeProfile,
		RuntimeSkill:     *runtimeSkillArg,
		SourceRepository: *sourceRepositoryArg,
		StandaloneRoot:   standaloneRoot,
	})
	summary, warnings, errorsFound := report.AggregateChecks(checks)
	summary["scope"] = "all"
	elapsed := report.Elapsed(start)
	data := standardReport("doctor --all", "doctor", elapsed, summary, warnings, errorsFound, checks)
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       outerCommand,
		OK:            len(errorsFound) == 0,
		Message:       "AiCoding product diagnostics",
		RepoRoot:      repo,
		Data:          data,
		Warnings:      warnings,
		Errors:        errorsFound,
		ElapsedMS:     elapsed,
	}, report.BoolErr(errorsFound)
}

func runProductVerify(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("verify")
	repoArg := fs.String("repo-root", "", "repository root")
	profileArg := fs.String("profile", "", productProfileHelp())
	codexConfigArg := fs.String("codex-config", "", "Codex config.toml path")
	configuredArg := fs.Bool("configured", false, "include configured Codex MCP inventory")
	runtimeProfileArg := fs.String("runtime-profile", "", "expected runtime Skill profile: runtime, full, or skill-development")
	runtimeSkillArg := fs.String("runtime-skill", "", "selected canonical Skill for skill-development")
	sourceRepositoryArg := fs.String("source-repository", "", "Codex-Skills source repository")
	standaloneRootArg := fs.String("standalone-root", "agents", "agents or codex")
	timeoutSecArg := fs.Int("timeout-sec", 180, "overall product verification timeout seconds")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	_, profile, err := normalizeTestProfile(*profileArg)
	if err != nil {
		return report.Result{}, usageErrorf("verify requires --profile Smoke|Full|Release")
	}
	runtimeProfile, standaloneRoot, err := productRuntimeSelection(*runtimeProfileArg, *runtimeSkillArg, *standaloneRootArg)
	if err != nil {
		return report.Result{}, err
	}
	if *timeoutSecArg < 1 {
		return report.Result{}, usageErrorf("verify --timeout-sec must be positive")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("verify --profile "+profile, start, "cannot resolve repo root", nil, err.Error()), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSecArg)*time.Second)
	defer cancel()
	checks := productVerifyChecks(ctx, repo, repohealth.ProductOptions{
		Profile:           profile,
		CodexConfig:       *codexConfigArg,
		IncludeConfigured: *configuredArg,
		RuntimeProfile:    runtimeProfile,
		RuntimeSkill:      *runtimeSkillArg,
		SourceRepository:  *sourceRepositoryArg,
		StandaloneRoot:    standaloneRoot,
	})
	summary, warnings, errorsFound := report.AggregateChecks(checks)
	summary["profile"] = profile
	elapsed := report.Elapsed(start)
	command := "verify --profile " + profile
	data := standardReport(command, profile, elapsed, summary, warnings, errorsFound, checks)
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       command,
		OK:            len(errorsFound) == 0,
		Message:       "AiCoding product verification",
		RepoRoot:      repo,
		Data:          data,
		Warnings:      warnings,
		Errors:        errorsFound,
		ElapsedMS:     elapsed,
	}, report.BoolErr(errorsFound)
}

func productRuntimeSelection(profile, skill, standaloneRoot string) (string, string, error) {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile != "" && !validChoice(profile, "runtime", "full", "skill-development") {
		return "", "", usageErrorf("unsupported runtime profile: %s", profile)
	}
	if profile == "skill-development" && strings.TrimSpace(skill) == "" {
		return "", "", usageErrorf("skill-development runtime profile requires --runtime-skill")
	}
	standaloneRoot = strings.ToLower(strings.TrimSpace(standaloneRoot))
	if !validChoice(standaloneRoot, "agents", "codex") {
		return "", "", usageErrorf("unsupported standalone root: %s", standaloneRoot)
	}
	return profile, standaloneRoot, nil
}
