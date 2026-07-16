package cli

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
)

var runTestEngine = testengine.Run

func runTest(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("test requires --profile Smoke|Full|Release or subcommand latest")
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "full", "release":
		return runTestProfile(sub, args[1:], "test "+sub, start)
	case "latest":
		return runTestLatest(args[1:], start)
	default:
		if strings.HasPrefix(args[0], "-") {
			return runTestProfile("", args, "", start)
		}
		return report.Result{}, usageErrorf("unsupported test subcommand: %s", sub)
	}
}

func runTestProfile(profile string, args []string, command string, start time.Time) (report.Result, error) {
	name := "test"
	if profile != "" {
		name += " " + profile
	}
	fs := newFlagSet(name)
	repoArg := fs.String("repo-root", "", "repository root")
	outArg := fs.String("out", "", "output directory")
	profileValue := profile
	if profile == "" {
		fs.StringVar(&profileValue, "profile", "", "Smoke, Full or Release")
	}
	timeoutSec := fs.Int("timeout-sec", 180, "per-command timeout seconds")
	longTimeoutSec := fs.Int("long-timeout-sec", 600, "long-command timeout seconds")
	concurrency := fs.Int("concurrency", 4, "concurrent read-only CLI calls")
	runnerTimeoutSec := fs.Int("runner-timeout-sec", 3600, "overall tester process timeout seconds")
	strictArg := fs.Bool("strict", false, "treat WARN severity command failures as FAIL")
	noJSONCheckArg := fs.Bool("no-json-check", false, "disable JSON output validation")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	profile, displayProfile, err := normalizeTestProfile(profileValue)
	if err != nil {
		return report.Result{}, err
	}
	if command == "" {
		command = "test --profile " + displayProfile
	}

	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("test "+profile, start, "cannot resolve repo root", nil, err.Error()), err
	}

	outDir := strings.TrimSpace(*outArg)
	if outDir == "" {
		outDir = filepath.Join(repo, "test-results", "aicoding-global-test-"+time.Now().Format("20060102-150405"))
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(repo, outDir)
	}

	cfg := testengine.Config{
		Repo:        repo,
		Out:         outDir,
		Profile:     profile,
		Timeout:     time.Duration(*timeoutSec) * time.Second,
		LongTimeout: time.Duration(*longTimeoutSec) * time.Second,
		Concurrency: *concurrency,
		Strict:      *strictArg,
		NoJSONCheck: *noJSONCheckArg,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*runnerTimeoutSec)*time.Second)
	defer cancel()

	fileReport, runErr := runTestEngine(ctx, cfg)
	timedOut := ctx.Err() == context.DeadlineExceeded

	errs := []string{}
	if timedOut {
		errs = append(errs, "test engine timed out")
	}
	if runErr != nil {
		errs = append(errs, runErr.Error())
	}

	data := globalTestStandardReport(command, profile, outDir, report.Elapsed(start), fileReport)
	if timedOut || runErr != nil {
		data.Findings = append(data.Findings, report.Finding{Level: "ERROR", Message: strings.Join(compactStrings(errs), "; ")})
	}
	data.Summary["runner_timed_out"] = timedOut
	data.Summary["runner_mode"] = "in-process"
	data.Summary["runner_stdout"] = ""
	data.Summary["runner_stderr"] = ""

	ok := len(errs) == 0 && testengine.ExitCode(fileReport, nil) == 0
	res := report.Result{
		SchemaVersion: 1,
		Command:       command,
		OK:            ok,
		Message:       "AiCoding global " + displayProfile + " test",
		RepoRoot:      repo,
		Data:          data,
		Errors:        compactStrings(errs),
		ElapsedMS:     report.Elapsed(start),
	}
	if !ok && len(res.Errors) == 0 {
		res.Errors = []string{"global tester conclusion: " + fileReport.Summary.Conclusion}
	}
	return res, report.BoolErr(res.Errors)
}

func runTestLatest(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("test latest")
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}

	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("test latest", start, "cannot resolve repo root", nil, err.Error()), err
	}

	outDir, err := testengine.LatestDir(repo)
	if err != nil {
		return report.Fail("test latest", start, "cannot locate latest test report", nil, err.Error()), err
	}
	fileReport, err := testengine.Load(outDir)
	if err != nil {
		return report.Fail("test latest", start, "cannot read latest test report", nil, err.Error()), err
	}
	data := globalTestStandardReport("test latest", fileReport.Summary.Profile, outDir, report.Elapsed(start), fileReport)
	return report.Result{SchemaVersion: 1, Command: "test latest", OK: true, Message: "latest AiCoding global test report", RepoRoot: repo, Data: data, ElapsedMS: report.Elapsed(start)}, nil
}

func normalizeTestProfile(value string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "smoke":
		return "smoke", "Smoke", nil
	case "full":
		return "full", "Full", nil
	case "release":
		return "release", "Release", nil
	case "":
		return "", "", usageErrorf("test requires --profile Smoke|Full|Release")
	default:
		return "", "", usageErrorf("unsupported test profile: %s", value)
	}
}

func globalTestStandardReport(command string, profile string, outDir string, durationMS int64, fileReport testengine.Report) report.StandardReport {
	s := fileReport.Summary
	status := s.Conclusion
	if status == "" {
		status = "FAIL"
	}
	return report.StandardReport{
		Status: status,
		Summary: map[string]interface{}{
			"repo":        s.Repo,
			"profile":     s.Profile,
			"started_at":  s.StartedAt,
			"ended_at":    s.EndedAt,
			"duration_ms": s.DurationMS,
			"total":       s.Total,
			"pass":        s.Pass,
			"fail":        s.Fail,
			"warn":        s.Warn,
			"skip":        s.Skip,
			"conclusion":  s.Conclusion,
			"output_dir":  filepath.ToSlash(outDir),
		},
		Findings:   globalTestFindings(fileReport.Results),
		Command:    command,
		Profile:    profile,
		DurationMS: durationMS,
		Logs: []report.LogRef{
			{Label: "report", Path: filepath.ToSlash(filepath.Join(outDir, "report.md"))},
			{Label: "summary", Path: filepath.ToSlash(filepath.Join(outDir, "summary.json"))},
			{Label: "results", Path: filepath.ToSlash(filepath.Join(outDir, "results.json"))},
		},
		Details: fileReport,
	}
}

func globalTestFindings(results []testengine.Result) []report.Finding {
	findings := []report.Finding{}
	for _, result := range results {
		switch result.Status {
		case testengine.Fail, testengine.Warn:
			findings = append(findings, report.Finding{
				Level:    string(result.Status),
				Message:  result.Reason,
				Category: result.Category,
				ID:       result.ID,
			})
		}
	}
	return findings
}

func compactStrings(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
