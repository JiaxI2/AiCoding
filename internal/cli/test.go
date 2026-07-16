package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

type globalTestSummary struct {
	Repo       string `json:"repo"`
	Profile    string `json:"profile"`
	StartedAt  string `json:"started_at"`
	EndedAt    string `json:"ended_at"`
	DurationMS int64  `json:"duration_ms"`
	Total      int    `json:"total"`
	Pass       int    `json:"pass"`
	Fail       int    `json:"fail"`
	Warn       int    `json:"warn"`
	Skip       int    `json:"skip"`
	Conclusion string `json:"conclusion"`
}

type globalTestCaseResult struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Severity   string `json:"severity"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	TimedOut   bool   `json:"timed_out"`
	JSONValid  bool   `json:"json_valid"`
	Command    string `json:"command"`
	StdoutFile string `json:"stdout_file,omitempty"`
	StderrFile string `json:"stderr_file,omitempty"`
	MetaFile   string `json:"meta_file,omitempty"`
	Reason     string `json:"reason"`
	Profile    string `json:"profile"`
}

type globalTestFileReport struct {
	Summary globalTestSummary      `json:"summary"`
	Results []globalTestCaseResult `json:"results"`
}

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

	cmdArgs := []string{
		"run", "./tools/aicoding-global-tester",
		"--repo", repo,
		"--profile", profile,
		"--out", outDir,
		"--timeout-sec", fmt.Sprint(*timeoutSec),
		"--long-timeout-sec", fmt.Sprint(*longTimeoutSec),
		"--concurrency", fmt.Sprint(*concurrency),
	}
	if *strictArg {
		cmdArgs = append(cmdArgs, "--strict")
	}
	if *noJSONCheckArg {
		cmdArgs = append(cmdArgs, "--no-json-check")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*runnerTimeoutSec)*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = repo
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	timedOut := ctx.Err() == context.DeadlineExceeded

	fileReport, loadErr := loadGlobalTestReport(outDir)
	errs := []string{}
	if timedOut {
		errs = append(errs, "global tester process timed out")
	}
	if runErr != nil {
		errs = append(errs, runErr.Error())
	}
	if loadErr != nil {
		errs = append(errs, loadErr.Error())
	}

	data := globalTestStandardReport(command, profile, outDir, report.Elapsed(start), fileReport)
	if timedOut || runErr != nil || loadErr != nil {
		data.Findings = append(data.Findings, report.Finding{Level: "ERROR", Message: strings.Join(compactStrings(errs), "; ")})
	}
	data.Summary["runner_timed_out"] = timedOut
	data.Summary["runner_stdout"] = truncateForReport(stdout.String(), 4000)
	data.Summary["runner_stderr"] = truncateForReport(stderr.String(), 4000)

	ok := len(errs) == 0 && fileReport.Summary.Conclusion != "FAIL"
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

	outDir, err := latestGlobalTestDir(repo)
	if err != nil {
		return report.Fail("test latest", start, "cannot locate latest test report", nil, err.Error()), err
	}
	fileReport, err := loadGlobalTestReport(outDir)
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

func loadGlobalTestReport(outDir string) (globalTestFileReport, error) {
	var fileReport globalTestFileReport
	resultsPath := filepath.Join(outDir, "results.json")
	raw, err := os.ReadFile(resultsPath)
	if err != nil {
		return fileReport, fmt.Errorf("read results.json: %w", err)
	}
	if err := json.Unmarshal(raw, &fileReport); err != nil {
		return fileReport, fmt.Errorf("parse results.json: %w", err)
	}
	if fileReport.Summary.Profile == "" {
		summaryRaw, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
		if err != nil {
			return fileReport, fmt.Errorf("read summary.json: %w", err)
		}
		if err := json.Unmarshal(summaryRaw, &fileReport.Summary); err != nil {
			return fileReport, fmt.Errorf("parse summary.json: %w", err)
		}
	}
	return fileReport, nil
}

func latestGlobalTestDir(repo string) (string, error) {
	pattern := filepath.Join(repo, "test-results", "aicoding-global-test-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	type candidate struct {
		path    string
		modTime time.Time
	}
	candidates := []candidate{}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(match, "summary.json")); err != nil {
			continue
		}
		candidates = append(candidates, candidate{path: match, modTime: info.ModTime()})
	}
	if len(candidates) == 0 {
		return "", errors.New("no test-results/aicoding-global-test-* report found")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].path > candidates[j].path
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	return candidates[0].path, nil
}

func globalTestStandardReport(command string, profile string, outDir string, durationMS int64, fileReport globalTestFileReport) report.StandardReport {
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

func globalTestFindings(results []globalTestCaseResult) []report.Finding {
	findings := []report.Finding{}
	for _, result := range results {
		switch result.Status {
		case "FAIL", "WARN":
			findings = append(findings, report.Finding{
				Level:    result.Status,
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

func truncateForReport(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
