package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Severity string
type Status string

const (
	Required Severity = "REQUIRED"
	WarnOnly Severity = "WARN"
	Optional Severity = "OPTIONAL"

	Pass Status = "PASS"
	Fail Status = "FAIL"
	Warn Status = "WARN"
	Skip Status = "SKIP"
)

type Config struct {
	Repo          string
	Out           string
	Profile       string
	Timeout       time.Duration
	LongTimeout   time.Duration
	Concurrency   int
	Strict        bool
	IncludeMutate bool
	NoJSONCheck   bool
}

type TestCase struct {
	ID           string
	Category     string
	Title        string
	Severity     Severity
	Profiles     []string
	Kind         string // command, static, concurrent
	Command      []string
	TimeoutKind  string // normal, long
	ExpectJSON   bool
	OptionalPath string
	Note         string
}

type Result struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Title      string   `json:"title"`
	Status     Status   `json:"status"`
	Severity   Severity `json:"severity"`
	DurationMS int64    `json:"duration_ms"`
	ExitCode   int      `json:"exit_code"`
	TimedOut   bool     `json:"timed_out"`
	JSONValid  bool     `json:"json_valid"`
	Command    string   `json:"command"`
	StdoutFile string   `json:"stdout_file,omitempty"`
	StderrFile string   `json:"stderr_file,omitempty"`
	MetaFile   string   `json:"meta_file,omitempty"`
	Reason     string   `json:"reason"`
	Profile    string   `json:"profile"`
}

type Summary struct {
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

type Report struct {
	Summary Summary  `json:"summary"`
	Results []Result `json:"results"`
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(2)
	}

	start := time.Now()
	if err := os.MkdirAll(filepath.Join(cfg.Out, "logs"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create out dir error:", err)
		os.Exit(2)
	}

	results := runAll(cfg)
	summary := summarize(cfg, start, time.Now(), results)
	report := Report{Summary: summary, Results: results}

	writeJSON(filepath.Join(cfg.Out, "results.json"), report)
	writeJSON(filepath.Join(cfg.Out, "summary.json"), summary)
	writeMarkdown(filepath.Join(cfg.Out, "report.md"), report)

	fmt.Println("AiCoding global test completed.")
	fmt.Println("Report:", filepath.Join(cfg.Out, "report.md"))
	fmt.Println("Conclusion:", summary.Conclusion)

	if summary.Conclusion == "FAIL" {
		os.Exit(1)
	}
}

func parseFlags() (Config, error) {
	var cfg Config
	var timeoutSec int
	var longTimeoutSec int

	flag.StringVar(&cfg.Repo, "repo", ".", "AiCoding repository root")
	flag.StringVar(&cfg.Out, "out", "", "output directory; default: <repo>/test-results/aicoding-global-test-YYYYMMDD-HHMMSS")
	flag.StringVar(&cfg.Profile, "profile", "full", "smoke|full|release|manual")
	flag.IntVar(&timeoutSec, "timeout-sec", 180, "per-command timeout seconds")
	flag.IntVar(&longTimeoutSec, "long-timeout-sec", 600, "long-command timeout seconds")
	flag.IntVar(&cfg.Concurrency, "concurrency", 4, "concurrent read-only CLI calls")
	flag.BoolVar(&cfg.Strict, "strict", false, "treat WARN severity command failures as FAIL")
	flag.BoolVar(&cfg.IncludeMutate, "include-mutating", false, "reserved: include isolated mutating lifecycle tests")
	flag.BoolVar(&cfg.NoJSONCheck, "no-json-check", false, "disable JSON output validation")
	flag.Parse()

	repo, err := filepath.Abs(cfg.Repo)
	if err != nil {
		return cfg, err
	}
	cfg.Repo = repo
	cfg.Timeout = time.Duration(timeoutSec) * time.Second
	cfg.LongTimeout = time.Duration(longTimeoutSec) * time.Second

	switch cfg.Profile {
	case "smoke", "full", "release", "manual":
	default:
		return cfg, fmt.Errorf("invalid profile %q", cfg.Profile)
	}

	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.Out == "" {
		stamp := time.Now().Format("20060102-150405")
		cfg.Out = filepath.Join(cfg.Repo, "test-results", "aicoding-global-test-"+stamp)
	}
	out, err := filepath.Abs(cfg.Out)
	if err != nil {
		return cfg, err
	}
	cfg.Out = out
	return cfg, nil
}

func runAll(cfg Config) []Result {
	tests := buildTests(cfg)
	var results []Result
	for _, tc := range tests {
		if !profileEnabled(tc, cfg.Profile) {
			results = append(results, Result{
				ID: tc.ID, Category: tc.Category, Title: tc.Title, Severity: tc.Severity,
				Status: Skip, Reason: "not selected by profile", Profile: cfg.Profile,
			})
			continue
		}
		if tc.OptionalPath != "" && !exists(filepath.Join(cfg.Repo, tc.OptionalPath)) {
			status := Skip
			if tc.Severity == Required {
				status = Warn
			}
			results = append(results, Result{
				ID: tc.ID, Category: tc.Category, Title: tc.Title, Severity: tc.Severity,
				Status: status, Reason: "optional path not found: " + tc.OptionalPath, Profile: cfg.Profile,
			})
			continue
		}

		var r Result
		switch tc.Kind {
		case "static":
			r = runStatic(cfg, tc)
		case "concurrent":
			r = runConcurrent(cfg, tc)
		default:
			r = runCommand(cfg, tc)
		}
		results = append(results, r)
	}
	return results
}

func buildTests(cfg Config) []TestCase {
	bin := aicodingBin(cfg.Repo)
	return []TestCase{
		{ID: "ENV-001", Category: "ENV", Title: "仓库根目录识别", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "ENV-002", Category: "ENV", Title: "Go 版本", Severity: Required, Profiles: allProfiles(), Kind: "command", Command: []string{"go", "version"}},
		{ID: "ENV-003", Category: "ENV", Title: "Git 版本", Severity: Required, Profiles: allProfiles(), Kind: "command", Command: []string{"git", "--version"}},
		{ID: "ENV-004", Category: "ENV", Title: "Task 可用性", Severity: WarnOnly, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{"task", "--version"}},
		{ID: "ENV-005", Category: "ENV", Title: "go.mod 模块路径", Severity: Required, Profiles: allProfiles(), Kind: "static"},

		{ID: "BOOT-001", Category: "BOOTSTRAP", Title: "bootstrap 构建 Go CLI", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{"go", "run", "./cmd/aicoding", "bootstrap", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "BOOT-002", Category: "BOOTSTRAP", Title: "CLI bootstrap 基础可用", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{bin, "bootstrap", "--json"}, ExpectJSON: true},

		{ID: "GO-001", Category: "GO", Title: "全仓 Go 单元测试", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{"go", "test", "./..."}, TimeoutKind: "long"},
		{ID: "GO-002", Category: "GO", Title: "Go race 检查", Severity: WarnOnly, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{"go", "test", "-race", "./..."}, TimeoutKind: "long"},
		{ID: "GO-003", Category: "GO", Title: "go vet 基础检查", Severity: WarnOnly, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{"go", "vet", "./..."}, TimeoutKind: "long"},
		{ID: "GO-004", Category: "GO", Title: "CLI 并发只读调用", Severity: Required, Profiles: []string{"full", "release"}, Kind: "concurrent", TimeoutKind: "normal", ExpectJSON: true},

		{ID: "C99-001", Category: "C99_SKILL", Title: "C99 skill status", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{bin, "skill", "c99-standard-c", "status", "--json"}, ExpectJSON: true},
		{ID: "C99-002", Category: "C99_SKILL", Title: "C99 注释模板校验", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{bin, "skill", "c99-standard-c", "templates", "--json"}, ExpectJSON: true},
		{ID: "C99-003", Category: "C99_SKILL", Title: "C99 样例路径格式检查", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{bin, "skill", "c99-standard-c", "check", "--scope", "paths", "--path", "testdata/style-samples/foc_sample.c", "--json"}, OptionalPath: "testdata/style-samples/foc_sample.c", ExpectJSON: true},
		{ID: "C99-004", Category: "C99_SKILL", Title: "C99 staged 检查入口", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "skill", "c99-standard-c", "check", "--scope", "staged", "--json"}, ExpectJSON: true},
		{ID: "C99-005", Category: "C99_SKILL", Title: "C99 source-of-truth 配置", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "C99-006", Category: "C99_SKILL", Title: "C99 排除目录策略", Severity: Required, Profiles: allProfiles(), Kind: "static"},

		{ID: "DOC-001", Category: "DOCSYNC", Title: "DocSync CI", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "docsync", "ci", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "DOC-002", Category: "DOCSYNC", Title: "DocSync all", Severity: WarnOnly, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "docsync", "all", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "DOC-003", Category: "DOCSYNC", Title: "DocSync release", Severity: Required, Profiles: []string{"release"}, Kind: "command", Command: []string{bin, "docsync", "release", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "DOC-004", Category: "DOCSYNC", Title: "文档索引一致性", Severity: Required, Profiles: allProfiles(), Kind: "static"},

		{ID: "LIFE-001", Category: "LIFECYCLE", Title: "kit registry 结构", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "LIFE-002", Category: "LIFECYCLE", Title: "kit manifest 存在性", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "LIFE-003", Category: "LIFECYCLE", Title: "lifecycle install plan", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "lifecycle", "plan", "--action", "install", "--all", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "LIFE-004", Category: "LIFECYCLE", Title: "lifecycle update plan", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "lifecycle", "plan", "--action", "update", "--all", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "LIFE-005", Category: "LIFECYCLE", Title: "lifecycle uninstall plan", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "lifecycle", "plan", "--action", "uninstall", "--all", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "LIFE-006", Category: "LIFECYCLE", Title: "lifecycle rollback 入口", Severity: WarnOnly, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "lifecycle", "rollback", "--last", "--json"}, ExpectJSON: true},

		{ID: "EXP-001", Category: "EXPORT", Title: "export zip", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "export", "--all", "--zip", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "FRESH-001", Category: "FRESH_CLONE", Title: "fresh-clone Smoke", Severity: WarnOnly, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "fresh-clone", "--profile", "Smoke", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "FRESH-002", Category: "FRESH_CLONE", Title: "fresh-clone Release", Severity: WarnOnly, Profiles: []string{"release"}, Kind: "command", Command: []string{bin, "fresh-clone", "--profile", "Release", "--json"}, TimeoutKind: "long", ExpectJSON: true},

		{ID: "DOCS-001", Category: "README_DOCS", Title: "README 三件套", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "DOCS-002", Category: "README_DOCS", Title: "README 架构声明", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "DOCS-003", Category: "README_DOCS", Title: "COMMANDS 命令矩阵", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "DOCS-004", Category: "README_DOCS", Title: "Fast Path 文档", Severity: Required, Profiles: allProfiles(), Kind: "static"},
		{ID: "DOCS-005", Category: "README_DOCS", Title: "C99 skill 文档", Severity: Required, Profiles: allProfiles(), Kind: "static"},

		{ID: "GIT-001", Category: "GIT_GOVERNANCE", Title: "工作区状态", Severity: WarnOnly, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{"git", "status", "--short"}},
		{ID: "GIT-002", Category: "GIT_GOVERNANCE", Title: "hooks verify", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{bin, "verify", "hooks", "--json"}, ExpectJSON: true},
		{ID: "GIT-003", Category: "GIT_GOVERNANCE", Title: "repo-text verify", Severity: Required, Profiles: []string{"smoke", "full", "release"}, Kind: "command", Command: []string{bin, "verify", "repo-text", "--json"}, ExpectJSON: true},
		{ID: "GIT-004", Category: "GIT_GOVERNANCE", Title: "release-notes verify", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "verify", "release-notes", "--json"}, ExpectJSON: true},
		{ID: "GIT-005", Category: "GIT_GOVERNANCE", Title: "governance lint", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "governance", "lint", "--json"}, ExpectJSON: true},
		{ID: "GIT-006", Category: "GIT_GOVERNANCE", Title: "tag audit", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "tag", "audit", "--json"}, ExpectJSON: true},
		{ID: "GIT-007", Category: "GIT_GOVERNANCE", Title: ".gitattributes 策略", Severity: Required, Profiles: allProfiles(), Kind: "static"},

		{ID: "PWSH-001", Category: "PWSH_BOUNDARY", Title: "PowerShell inventory", Severity: WarnOnly, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "doctor", "pwsh", "--json"}, ExpectJSON: true},
		{ID: "PWSH-002", Category: "PWSH_BOUNDARY", Title: "PowerShell budget", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "doctor", "pwsh-budget", "--json"}, ExpectJSON: true},
		{ID: "PWSH-003", Category: "PWSH_BOUNDARY", Title: "默认入口不经 PowerShell 编排", Severity: Required, Profiles: allProfiles(), Kind: "static"},

		{ID: "FULL-001", Category: "RELEASE_GATE", Title: "Full 聚合", Severity: Required, Profiles: []string{"full", "release"}, Kind: "command", Command: []string{bin, "full", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "REL-001", Category: "RELEASE_GATE", Title: "Release gate", Severity: Required, Profiles: []string{"release"}, Kind: "command", Command: []string{bin, "release", "gate", "--json"}, TimeoutKind: "long", ExpectJSON: true},
		{ID: "REL-002", Category: "RELEASE_GATE", Title: "Release policy 文档", Severity: Required, Profiles: allProfiles(), Kind: "static"},
	}
}

func allProfiles() []string {
	return []string{"smoke", "full", "release", "manual"}
}

func profileEnabled(tc TestCase, profile string) bool {
	for _, p := range tc.Profiles {
		if p == profile {
			return true
		}
	}
	return false
}

func aicodingBin(repo string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(repo, "bin", "aicoding.exe")
	}
	return filepath.Join(repo, "bin", "aicoding")
}

func timeoutFor(cfg Config, tc TestCase) time.Duration {
	if tc.TimeoutKind == "long" {
		return cfg.LongTimeout
	}
	return cfg.Timeout
}

func runCommand(cfg Config, tc TestCase) Result {
	start := time.Now()
	r := Result{
		ID: tc.ID, Category: tc.Category, Title: tc.Title, Severity: tc.Severity,
		Command: strings.Join(tc.Command, " "), Profile: cfg.Profile, ExitCode: -1,
	}

	if len(tc.Command) == 0 {
		r.Status = Fail
		r.Reason = "empty command"
		return r
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutFor(cfg, tc))
	defer cancel()

	cmd := exec.CommandContext(ctx, tc.Command[0], tc.Command[1:]...)
	cmd.Dir = cfg.Repo
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	r.DurationMS = time.Since(start).Milliseconds()
	if ctx.Err() == context.DeadlineExceeded {
		r.TimedOut = true
	}
	r.ExitCode = exitCode(err)

	stdoutFile, stderrFile, metaFile := writeLogs(cfg, tc.ID, stdout.Bytes(), stderr.Bytes(), r, err)
	r.StdoutFile = stdoutFile
	r.StderrFile = stderrFile
	r.MetaFile = metaFile

	r.JSONValid = true
	if tc.ExpectJSON && !cfg.NoJSONCheck {
		r.JSONValid = validJSONFromOutput(stdout.String())
	}

	if err == nil && !r.TimedOut && (!tc.ExpectJSON || cfg.NoJSONCheck || r.JSONValid) {
		r.Status = Pass
		r.Reason = "command passed"
		return r
	}

	if r.TimedOut {
		r.Reason = "command timed out"
	} else if tc.ExpectJSON && !cfg.NoJSONCheck && !r.JSONValid {
		r.Reason = "command output is not valid JSON"
	} else if err != nil {
		r.Reason = "command failed: " + err.Error()
	} else {
		r.Reason = "command failed"
	}

	if tc.Severity == Required || cfg.Strict {
		r.Status = Fail
	} else {
		r.Status = Warn
	}
	return r
}

func runConcurrent(cfg Config, tc TestCase) Result {
	start := time.Now()
	bin := aicodingBin(cfg.Repo)
	commands := [][]string{
		{bin, "skill", "c99-standard-c", "status", "--json"},
		{bin, "skill", "c99-standard-c", "templates", "--json"},
		{bin, "governance", "lint", "--json"},
		{bin, "verify", "repo-text", "--json"},
	}

	var mu sync.Mutex
	var details []string
	var failed int
	var timedOut int
	var invalidJSON int

	workers := cfg.Concurrency
	if workers > len(commands) {
		workers = len(commands)
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for i := 0; i < cfg.Concurrency; i++ {
		cmdSpec := commands[i%len(commands)]
		wg.Add(1)
		go func(index int, spec []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, spec[0], spec[1:]...)
			cmd.Dir = cfg.Repo
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			code := exitCode(err)
			jsonOK := validJSONFromOutput(stdout.String())
			to := ctx.Err() == context.DeadlineExceeded

			mu.Lock()
			defer mu.Unlock()
			if err != nil || code != 0 {
				failed++
			}
			if to {
				timedOut++
			}
			if !jsonOK {
				invalidJSON++
			}
			details = append(details, fmt.Sprintf("[%02d] code=%d timeout=%v json=%v cmd=%s stderr=%s", index, code, to, jsonOK, strings.Join(spec, " "), truncate(stderr.String(), 160)))
		}(i, cmdSpec)
	}
	wg.Wait()

	r := Result{
		ID: tc.ID, Category: tc.Category, Title: tc.Title, Severity: tc.Severity,
		Status: Pass, DurationMS: time.Since(start).Milliseconds(), ExitCode: 0,
		TimedOut: timedOut > 0, JSONValid: invalidJSON == 0,
		Command: "concurrent read-only CLI calls x" + strconv.Itoa(cfg.Concurrency),
		Profile: cfg.Profile,
	}
	out := []byte(strings.Join(details, "\n"))
	stdoutFile, stderrFile, metaFile := writeLogs(cfg, tc.ID, out, nil, r, nil)
	r.StdoutFile = stdoutFile
	r.StderrFile = stderrFile
	r.MetaFile = metaFile

	if failed == 0 && timedOut == 0 && invalidJSON == 0 {
		r.Reason = "all concurrent read-only calls passed"
		return r
	}
	r.Reason = fmt.Sprintf("failed=%d timeout=%d invalid_json=%d", failed, timedOut, invalidJSON)
	if tc.Severity == Required || cfg.Strict {
		r.Status = Fail
	} else {
		r.Status = Warn
	}
	return r
}

func runStatic(cfg Config, tc TestCase) Result {
	start := time.Now()
	r := Result{
		ID: tc.ID, Category: tc.Category, Title: tc.Title, Severity: tc.Severity,
		Status: Pass, ExitCode: 0, JSONValid: true, Profile: cfg.Profile,
	}

	var err error
	switch tc.ID {
	case "ENV-001":
		err = requirePaths(cfg.Repo, ".git", "go.mod", "README.md")
	case "ENV-005":
		err = checkGoMod(cfg.Repo)
	case "C99-005":
		err = requirePaths(cfg.Repo,
			"config/skills/c99-standard-c/skill.json",
			"config/skills/c99-standard-c/style/clang-format.yaml",
			"config/skills/c99-standard-c/templates/comment-templates.json",
			"config/skills/c99-standard-c/rules/embedded-c-rules.md",
			".clang-format",
		)
		if err == nil {
			err = checkC99Projection(cfg.Repo)
		}
	case "C99-006":
		err = checkC99ExcludedDirs(cfg.Repo)
	case "DOC-004":
		err = checkDocIndex(cfg.Repo)
	case "LIFE-001":
		err = checkKitRegistry(cfg.Repo, false)
	case "LIFE-002":
		err = checkKitRegistry(cfg.Repo, true)
	case "DOCS-001":
		err = requirePaths(cfg.Repo, "README.md", "README_CN.md", "README_EN.md")
	case "DOCS-002":
		err = fileContainsAll(filepath.Join(cfg.Repo, "README.md"), []string{"Go CLI", "DocSync", "skill verify", "lifecycle", "export", "fresh-clone"})
	case "DOCS-003":
		err = fileContainsAll(filepath.Join(cfg.Repo, "docs/COMMANDS.md"), []string{"bootstrap", "smoke", "ci", "full", "release", "c99-standard-c", "docsync", "lifecycle", "export", "fresh-clone"})
	case "DOCS-004":
		err = fileContainsAll(filepath.Join(cfg.Repo, "docs/COMMANDS.md"), []string{"Go CLI", "Full", "Release", "DocSync", "Lifecycle", "PowerShell Boundary"})
	case "DOCS-005":
		err = fileContainsAll(filepath.Join(cfg.Repo, "docs/guides/C99_STANDARD_C_SKILL.md"), []string{"config/skills/c99-standard-c", "skill c99-standard-c", "fmt", "check", "templates"})
	case "GIT-007":
		err = checkGitAttributes(cfg.Repo)
	case "PWSH-003":
		err = checkTaskfileGoRoutes(cfg.Repo)
	case "REL-002":
		err = requirePaths(cfg.Repo, "docs/governance/TAGGING_POLICY.md", "docs/governance/RELEASE_POLICY.md")
	default:
		err = errors.New("static check not implemented")
	}

	r.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		r.Reason = err.Error()
		if tc.Severity == Required || cfg.Strict {
			r.Status = Fail
		} else {
			r.Status = Warn
		}
	} else {
		r.Reason = "static check passed"
	}
	meta := map[string]any{"id": tc.ID, "status": r.Status, "reason": r.Reason}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_, _, metaFile := writeLogs(cfg, tc.ID, []byte(r.Reason), nil, r, nil)
	_ = os.WriteFile(filepath.Join(cfg.Out, metaFile), metaBytes, 0o644)
	r.MetaFile = metaFile
	return r
}

func requirePaths(repo string, rels ...string) error {
	var missing []string
	for _, rel := range rels {
		if !exists(filepath.Join(repo, rel)) {
			missing = append(missing, rel)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing paths: %s", strings.Join(missing, ", "))
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkGoMod(repo string) error {
	b, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		return err
	}
	text := string(b)
	if !strings.Contains(text, "module github.com/JiaxI2/AiCoding") {
		return errors.New("go.mod module is not github.com/JiaxI2/AiCoding")
	}
	if !strings.Contains(text, "go 1.22") {
		return errors.New("go.mod does not declare go 1.22")
	}
	return nil
}

func checkC99Projection(repo string) error {
	source, err := os.ReadFile(filepath.Join(repo, "config/skills/c99-standard-c/style/clang-format.yaml"))
	if err != nil {
		return err
	}
	proj, err := os.ReadFile(filepath.Join(repo, ".clang-format"))
	if err != nil {
		return err
	}
	keys := []string{"IndentWidth", "TabWidth", "ColumnLimit", "BreakBeforeBraces", "PointerAlignment", "SortIncludes"}
	for _, k := range keys {
		if !strings.Contains(string(source), k) || !strings.Contains(string(proj), k) {
			return fmt.Errorf("clang-format key %s missing in source or projection", k)
		}
	}
	if !strings.Contains(string(proj), "Source of truth") && !strings.Contains(string(proj), "source of truth") {
		return errors.New(".clang-format does not declare source-of-truth boundary")
	}
	return nil
}

func checkC99ExcludedDirs(repo string) error {
	b, err := os.ReadFile(filepath.Join(repo, "config/skills/c99-standard-c/skill.json"))
	if err != nil {
		return err
	}
	var data struct {
		Excluded []string `json:"excludedDirectories"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	want := []string{"vendor", "third_party", "generated", "Drivers", "device", "build", "out", "dist"}
	have := map[string]bool{}
	for _, d := range data.Excluded {
		have[d] = true
	}
	var missing []string
	for _, d := range want {
		if !have[d] {
			missing = append(missing, d)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing excludedDirectories: %s", strings.Join(missing, ", "))
	}
	return nil
}

func checkKitRegistry(repo string, requireManifest bool) error {
	b, err := os.ReadFile(filepath.Join(repo, "config/kit-registry.json"))
	if err != nil {
		return err
	}
	var data struct {
		SchemaVersion int    `json:"schemaVersion"`
		Name          string `json:"name"`
		Kits          []struct {
			ID       string `json:"id"`
			Enabled  bool   `json:"enabled"`
			Order    int    `json:"order"`
			Manifest string `json:"manifest"`
		} `json:"kits"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if data.SchemaVersion == 0 || data.Name == "" || len(data.Kits) == 0 {
		return errors.New("kit registry missing schemaVersion/name/kits")
	}
	seen := map[string]bool{}
	for _, k := range data.Kits {
		if k.ID == "" || k.Manifest == "" {
			return errors.New("kit registry contains empty id or manifest")
		}
		if seen[k.ID] {
			return fmt.Errorf("duplicate kit id: %s", k.ID)
		}
		seen[k.ID] = true
		if requireManifest && !exists(filepath.Join(repo, k.Manifest)) {
			return fmt.Errorf("manifest not found for kit %s: %s", k.ID, k.Manifest)
		}
	}
	return nil
}

func checkDocIndex(repo string) error {
	// README.md is the top-level entry document. It should reference stable hub
	// documents only, not every leaf skill document. Leaf skill documents remain
	// verified directly below, and docs/COMMANDS.md is allowed to carry detailed
	// skill command coverage. This avoids forcing README churn whenever a specific
	// skill is added, renamed, or split.
	checks := map[string][]string{
		"README.md":                           {"docs/COMMANDS.md"},
		"docs/COMMANDS.md":                    {"docsync", "skill verify", "lifecycle", "fresh-clone", "c99-standard-c"},
		"docs/guides/C99_STANDARD_C_SKILL.md": {"skill c99-standard-c", "config/skills/c99-standard-c"},
	}
	for rel, words := range checks {
		if err := fileContainsAll(filepath.Join(repo, rel), words); err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
	}
	return nil
}

func fileContainsAll(path string, words []string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(b)
	var missing []string
	for _, w := range words {
		if !strings.Contains(text, w) {
			missing = append(missing, w)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing text: %s", strings.Join(missing, ", "))
	}
	return nil
}

func checkGitAttributes(repo string) error {
	b, err := os.ReadFile(filepath.Join(repo, ".gitattributes"))
	if err != nil {
		return err
	}
	rules := parseGitAttributes(string(b))
	type rule struct {
		pattern string
		attrs   []string
	}
	want := []rule{
		{pattern: "*.md", attrs: []string{"text", "eol=lf"}},
		{pattern: "*.go", attrs: []string{"text", "eol=lf"}},
		{pattern: "*.json", attrs: []string{"text", "eol=lf"}},
		{pattern: "*.yml", attrs: []string{"text", "eol=lf"}},
		{pattern: "*.yaml", attrs: []string{"text", "eol=lf"}},
		{pattern: "*.ps1", attrs: []string{"text", "eol=crlf"}},
		{pattern: "*.psm1", attrs: []string{"text", "eol=crlf"}},
		{pattern: "*.zip", attrs: []string{"binary"}},
		{pattern: "*.exe", attrs: []string{"binary"}},
	}
	var missing []string
	for _, w := range want {
		if !gitAttrRuleHas(rules, w.pattern, w.attrs) {
			missing = append(missing, w.pattern+" "+strings.Join(w.attrs, " "))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(".gitattributes missing policy: %s", strings.Join(missing, ", "))
	}
	return nil
}

func parseGitAttributes(text string) map[string]map[string]bool {
	rules := map[string]map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pattern := fields[0]
		if rules[pattern] == nil {
			rules[pattern] = map[string]bool{}
		}
		for _, attr := range fields[1:] {
			rules[pattern][attr] = true
		}
	}
	return rules
}

func gitAttrRuleHas(rules map[string]map[string]bool, pattern string, attrs []string) bool {
	have := rules[pattern]
	if have == nil {
		return false
	}
	for _, attr := range attrs {
		if !have[attr] {
			return false
		}
	}
	return true
}

func checkTaskfileGoRoutes(repo string) error {
	b, err := os.ReadFile(filepath.Join(repo, "Taskfile.yml"))
	if err != nil {
		return err
	}
	norm := normalizeTaskfileText(string(b))
	checks := map[string][]string{
		"smoke": {
			"aicoding.exe smoke --json",
			"aicoding smoke --json",
			"aicoding.exe ci --profile smoke --json",
			"aicoding ci --profile smoke --json",
		},
		"full": {
			"aicoding.exe test full --json",
			"aicoding test full --json",
			"aicoding.exe full --json",
			"aicoding full --json",
		},
		"release": {
			"aicoding.exe test release --json",
			"aicoding test release --json",
			"aicoding.exe release gate --json",
			"aicoding release gate --json",
		},
	}
	var missing []string
	for name, alternatives := range checks {
		matched := false
		for _, alt := range alternatives {
			if strings.Contains(norm, alt) {
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("Taskfile missing Go-native default routes: %s", strings.Join(missing, ", "))
	}
	return nil
}

func normalizeTaskfileText(text string) string {
	text = strings.ReplaceAll(text, "\\", "/")
	text = strings.ReplaceAll(text, "{{.AICODING_BIN}}", "aicoding.exe")
	text = strings.ReplaceAll(text, "${AICODING_BIN}", "aicoding.exe")
	text = strings.ReplaceAll(text, "$AICODING_BIN", "aicoding.exe")
	text = strings.ReplaceAll(text, "\"", "")
	text = strings.ReplaceAll(text, "'", "")
	text = strings.ToLower(text)
	return strings.Join(strings.Fields(text), " ")
}

func validJSONFromOutput(out string) bool {
	out = strings.TrimSpace(out)
	if out == "" {
		return false
	}
	if json.Valid([]byte(out)) {
		return true
	}
	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start >= 0 && end > start {
		return json.Valid([]byte(out[start : end+1]))
	}
	start = strings.Index(out, "[")
	end = strings.LastIndex(out, "]")
	if start >= 0 && end > start {
		return json.Valid([]byte(out[start : end+1]))
	}
	return false
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func writeLogs(cfg Config, id string, stdout []byte, stderr []byte, r Result, runErr error) (string, string, string) {
	logDir := filepath.Join(cfg.Out, "logs")
	_ = os.MkdirAll(logDir, 0o755)

	stdoutRel := filepath.ToSlash(filepath.Join("logs", id+".stdout.txt"))
	stderrRel := filepath.ToSlash(filepath.Join("logs", id+".stderr.txt"))
	metaRel := filepath.ToSlash(filepath.Join("logs", id+".meta.json"))

	_ = os.WriteFile(filepath.Join(cfg.Out, stdoutRel), stdout, 0o644)
	_ = os.WriteFile(filepath.Join(cfg.Out, stderrRel), stderr, 0o644)

	meta := map[string]any{
		"id":          r.ID,
		"category":    r.Category,
		"title":       r.Title,
		"command":     r.Command,
		"exit_code":   r.ExitCode,
		"timed_out":   r.TimedOut,
		"duration_ms": r.DurationMS,
		"run_error":   "",
	}
	if runErr != nil {
		meta["run_error"] = runErr.Error()
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(cfg.Out, metaRel), metaBytes, 0o644)

	return stdoutRel, stderrRel, metaRel
}

func summarize(cfg Config, start, end time.Time, results []Result) Summary {
	s := Summary{
		Repo:       cfg.Repo,
		Profile:    cfg.Profile,
		StartedAt:  start.Format(time.RFC3339),
		EndedAt:    end.Format(time.RFC3339),
		DurationMS: end.Sub(start).Milliseconds(),
		Total:      len(results),
	}
	for _, r := range results {
		switch r.Status {
		case Pass:
			s.Pass++
		case Fail:
			s.Fail++
		case Warn:
			s.Warn++
		case Skip:
			s.Skip++
		}
	}
	if s.Fail > 0 {
		s.Conclusion = "FAIL"
	} else if s.Warn > 0 {
		s.Conclusion = "PASS_WITH_WARNINGS"
	} else {
		s.Conclusion = "PASS"
	}
	return s
}

func writeJSON(path string, v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
}

func writeMarkdown(path string, report Report) {
	var b strings.Builder
	s := report.Summary
	b.WriteString("# AiCoding Global Test Report\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString("| Item | Value |\n|---|---:|\n")
	b.WriteString(fmt.Sprintf("| Repo | `%s` |\n", s.Repo))
	b.WriteString(fmt.Sprintf("| Profile | `%s` |\n", s.Profile))
	b.WriteString(fmt.Sprintf("| Started | `%s` |\n", s.StartedAt))
	b.WriteString(fmt.Sprintf("| Ended | `%s` |\n", s.EndedAt))
	b.WriteString(fmt.Sprintf("| Duration ms | %d |\n", s.DurationMS))
	b.WriteString(fmt.Sprintf("| Total | %d |\n", s.Total))
	b.WriteString(fmt.Sprintf("| PASS | %d |\n", s.Pass))
	b.WriteString(fmt.Sprintf("| FAIL | %d |\n", s.Fail))
	b.WriteString(fmt.Sprintf("| WARN | %d |\n", s.Warn))
	b.WriteString(fmt.Sprintf("| SKIP | %d |\n", s.Skip))
	b.WriteString(fmt.Sprintf("| Conclusion | **%s** |\n", s.Conclusion))

	writeStatusSection(&b, "Failed Cases", report.Results, Fail)
	writeStatusSection(&b, "Warning Cases", report.Results, Warn)
	writeStatusSection(&b, "Skipped Cases", report.Results, Skip)

	b.WriteString("\n## Results By Category\n\n")
	cats := categories(report.Results)
	for _, cat := range cats {
		b.WriteString("### " + cat + "\n\n")
		b.WriteString("| ID | Status | Severity | Duration ms | Reason | Logs |\n|---|---|---|---:|---|---|\n")
		for _, r := range report.Results {
			if r.Category != cat {
				continue
			}
			logs := ""
			if r.StdoutFile != "" {
				logs = fmt.Sprintf("[%s](%s)", "stdout", r.StdoutFile)
			}
			if r.StderrFile != "" {
				if logs != "" {
					logs += " / "
				}
				logs += fmt.Sprintf("[%s](%s)", "stderr", r.StderrFile)
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s |\n",
				r.ID, r.Status, r.Severity, r.DurationMS, mdEscape(r.Reason), logs))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## Slowest Cases\n\n")
	b.WriteString("| ID | Category | Status | Duration ms | Command |\n|---|---|---|---:|---|\n")
	slowest := append([]Result(nil), report.Results...)
	sort.Slice(slowest, func(i, j int) bool { return slowest[i].DurationMS > slowest[j].DurationMS })
	limit := 10
	if len(slowest) < limit {
		limit = len(slowest)
	}
	for i := 0; i < limit; i++ {
		r := slowest[i]
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | `%s` |\n", r.ID, r.Category, r.Status, r.DurationMS, mdEscape(r.Command)))
	}

	b.WriteString("\n## Review Guidance\n\n")
	b.WriteString("1. 先处理 `FAIL` 的 REQUIRED 用例。\n")
	b.WriteString("2. 对 `WARN` 用例，结合本机环境判断是否属于工具链、网络或可选能力问题。\n")
	b.WriteString("3. C99 风格一致性重点查看 `C99_SKILL` 区域和对应 stdout。\n")
	b.WriteString("4. DocSync 与 README 治理重点查看 `DOCSYNC`、`README_DOCS` 区域。\n")
	b.WriteString("5. Go 并发重点查看 `GO-004`；race 检查重点查看 `GO-002`。\n")

	_ = os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeStatusSection(b *strings.Builder, title string, results []Result, status Status) {
	b.WriteString("\n## " + title + "\n\n")
	b.WriteString("| ID | Category | Title | Severity | Reason | Logs |\n|---|---|---|---|---|---|\n")
	found := false
	for _, r := range results {
		if r.Status != status {
			continue
		}
		found = true
		logs := ""
		if r.StdoutFile != "" {
			logs = fmt.Sprintf("[%s](%s)", "stdout", r.StdoutFile)
		}
		if r.StderrFile != "" {
			if logs != "" {
				logs += " / "
			}
			logs += fmt.Sprintf("[%s](%s)", "stderr", r.StderrFile)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			r.ID, r.Category, mdEscape(r.Title), r.Severity, mdEscape(r.Reason), logs))
	}
	if !found {
		b.WriteString("| - | - | - | - | - | - |\n")
	}
}

func categories(results []Result) []string {
	set := map[string]bool{}
	for _, r := range results {
		set[r.Category] = true
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
