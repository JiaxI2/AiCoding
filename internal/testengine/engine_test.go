package testengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestNormalizeConfigAndRegistry(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 180*time.Second || cfg.LongTimeout != 600*time.Second || cfg.Concurrency != 1 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
	if !filepath.IsAbs(cfg.Repo) || !filepath.IsAbs(cfg.Out) {
		t.Fatalf("config paths must be absolute: %#v", cfg)
	}

	seen := map[string]bool{}
	for _, testCase := range Registry(cfg) {
		if testCase.ID == "" || seen[testCase.ID] {
			t.Fatalf("registry contains empty or duplicate test id: %q", testCase.ID)
		}
		seen[testCase.ID] = true
	}
	for _, id := range []string{"ENV-001", "GO-001", "GO-005", "GO-006", "GIT-009", "EXP-002", "FRESH-001", "FRESH-003", "REL-002"} {
		if !seen[id] {
			t.Fatalf("registry is missing %s", id)
		}
	}
	for _, removed := range []string{"FULL-001", "REL-001", "FRESH-002"} {
		if seen[removed] {
			t.Fatalf("registry still contains recursive aggregate case %s", removed)
		}
	}

	for _, testCase := range Registry(cfg) {
		if len(testCase.Command) < 2 || !strings.Contains(strings.ToLower(filepath.Base(testCase.Command[0])), "aicoding") {
			continue
		}
		subcommand := strings.ToLower(testCase.Command[1])
		if subcommand == "smoke" || subcommand == "ci" || subcommand == "full" {
			t.Fatalf("%s recursively calls compatibility aggregate: %s", testCase.ID, strings.Join(testCase.Command, " "))
		}
		if subcommand == "release" && len(testCase.Command) > 2 && strings.EqualFold(testCase.Command[2], "gate") {
			t.Fatalf("%s recursively calls release gate: %s", testCase.ID, strings.Join(testCase.Command, " "))
		}
	}
	for _, testCase := range Registry(cfg) {
		if testCase.ID != "LIFE-006" {
			continue
		}
		command := strings.Join(testCase.Command, " ")
		if !strings.Contains(command, "lifecycle rollback --scope kit --help") {
			t.Fatalf("rollback contract check must be read-only, got %q", command)
		}
		if strings.Contains(command, "--last") {
			t.Fatalf("test profiles must not apply rollback state: %q", command)
		}
	}

	if _, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: "nightly"}); err == nil {
		t.Fatal("invalid profile must fail")
	}
}

func TestCommandTimingBreakdownAndSlowestCases(t *testing.T) {
	repo := t.TempDir()
	cfg := Config{Repo: repo, Out: filepath.Join(repo, "results"), Profile: ProfileFull, Timeout: 10 * time.Second}
	result := runCommand(context.Background(), cfg, TestCase{
		ID: "TIMING-001", Category: "TEST", Title: "timed command", Severity: Required,
		Command: []string{"go", "version"},
	})
	if result.Status != Pass {
		t.Fatalf("timed command = %#v", result)
	}
	if result.QueueMS == nil || result.SetupMS == nil || result.ExecuteMS == nil || result.PersistMS == nil {
		t.Fatalf("timing fields are not all present: %#v", result)
	}
	if got := timingValue(result.QueueMS) + timingValue(result.SetupMS) + timingValue(result.ExecuteMS) + timingValue(result.PersistMS); got != result.DurationMS {
		t.Fatalf("timing sum = %d, duration = %d: %#v", got, result.DurationMS, result)
	}
	legacyJSON, err := json.Marshal(Result{ID: "SKIPPED", Status: Skip})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"queue_ms", "setup_ms", "execute_ms", "persist_ms"} {
		if bytes.Contains(legacyJSON, []byte(field)) {
			t.Fatalf("zero-value result must omit %s for compatibility: %s", field, legacyJSON)
		}
	}

	results := []Result{}
	for index := 1; index <= 6; index++ {
		results = append(results, Result{ID: fmt.Sprintf("CASE-%d", index), Status: Pass, DurationMS: int64(index)})
	}
	summary := summarize(cfg, time.Unix(0, 0), time.Unix(1, 0), results)
	if summary.CacheHitRatio == nil || *summary.CacheHitRatio != 0 {
		t.Fatalf("executed cache hit ratio = %#v", summary.CacheHitRatio)
	}
	if len(summary.SlowestCases) != 5 || summary.SlowestCases[0].ID != "CASE-6" || summary.SlowestCases[4].ID != "CASE-2" {
		t.Fatalf("slowest cases = %#v", summary.SlowestCases)
	}
}

func TestRegistryKeepsHermeticAndZipChecksInRelease(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileFull})
	if err != nil {
		t.Fatal(err)
	}
	full := map[string]TestCase{}
	release := map[string]TestCase{}
	for _, testCase := range Registry(cfg) {
		if profileEnabled(testCase, ProfileFull) {
			full[testCase.ID] = testCase
		}
		if profileEnabled(testCase, ProfileRelease) {
			release[testCase.ID] = testCase
		}
	}
	for _, id := range []string{"EXP-001", "FRESH-001"} {
		if _, exists := full[id]; exists {
			t.Fatalf("Full still contains expensive command case %s", id)
		}
		if testCase, exists := release[id]; !exists || testCase.Kind != "command" {
			t.Fatalf("Release lost expensive command case %s: %#v", id, testCase)
		}
	}
	for _, id := range []string{"EXP-002", "FRESH-003"} {
		if testCase, exists := full[id]; !exists || testCase.Kind != "static" {
			t.Fatalf("Full lost static replacement %s: %#v", id, testCase)
		}
	}
	if !containsString(release["EXP-001"].Command, "--zip") {
		t.Fatalf("Release export is not a real ZIP command: %#v", release["EXP-001"])
	}
	if !containsString(release["FRESH-001"].Command, "Release") {
		t.Fatalf("Release fresh clone uses the wrong profile: %#v", release["FRESH-001"])
	}
}

func TestRegistryHasPrimitiveChecklistGate(t *testing.T) {
	// todolist 0001: the registry must own the "new-primitive ADR carries a §12
	// self-review" gate as a static case so it runs in every profile.
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke})
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range Registry(cfg) {
		if testCase.ID != "ADR-001" {
			continue
		}
		if testCase.Kind != "static" || testCase.Severity != Required {
			t.Fatalf("ADR-001 must be a required static gate: %#v", testCase)
		}
		if len(testCase.Profiles) != len(allProfiles()) {
			t.Fatalf("ADR-001 must run in all profiles: %#v", testCase.Profiles)
		}
		return
	}
	t.Fatal("registry is missing the ADR-001 primitive-checklist gate")
}

func TestRegistryBuildCommandsAreBoundedAndBootstrapCoverageRemains(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileFull})
	if err != nil {
		t.Fatal(err)
	}
	buildCommands := 0
	goRunCommands := map[string]string{
		"GO-005": staticcheckCommand,
		"GO-006": govulncheckCommand,
	}
	seenGoRun := map[string]bool{}
	found := map[string]TestCase{}
	for _, testCase := range Registry(cfg) {
		found[testCase.ID] = testCase
		if len(testCase.Command) == 0 || testCase.Command[0] != "go" {
			continue
		}
		if len(testCase.Command) > 1 && testCase.Command[1] == "build" {
			buildCommands++
		}
		if len(testCase.Command) > 1 && testCase.Command[1] == "run" {
			expected, ok := goRunCommands[testCase.ID]
			if !ok || len(testCase.Command) < 3 || testCase.Command[2] != expected {
				t.Fatalf("registry contains an unapproved or unpinned go run command: %#v", testCase)
			}
			seenGoRun[testCase.ID] = true
		}
	}
	if buildCommands > 1 {
		t.Fatalf("registry contains %d direct go build commands, want at most 1", buildCommands)
	}
	for id := range goRunCommands {
		if !seenGoRun[id] {
			t.Fatalf("registry is missing pinned go run command %s", id)
		}
	}
	if _, exists := found["BOOT-001"]; exists {
		t.Fatal("BOOT-001 duplicate build case is still registered")
	}
	if boot := found["BOOT-002"]; len(boot.Command) == 0 || !containsString(boot.Command, "--no-build") {
		t.Fatalf("BOOT-002 does not preserve the no-build CLI contract: %#v", boot)
	}
	if boot := found["BOOT-003"]; boot.Kind != "static" || boot.Severity != Required {
		t.Fatalf("BOOT-003 static prerequisite coverage is missing: %#v", boot)
	}
}

func TestRegistryPinsStaticcheckAndGovulncheckPolicy(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileFull})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]TestCase{}
	for _, testCase := range Registry(cfg) {
		found[testCase.ID] = testCase
	}
	staticcheck := found["GO-005"]
	if staticcheck.Severity != WarnOnly || strings.Join(staticcheck.Command, " ") != "go run "+staticcheckCommand+" ./..." {
		t.Fatalf("GO-005 policy mismatch: %#v", staticcheck)
	}
	govulncheck := found["GO-006"]
	if govulncheck.Severity != Required || !govulncheck.NetworkFailureWarn || strings.Join(govulncheck.Command, " ") != "go run "+govulncheckCommand+" ./..." {
		t.Fatalf("GO-006 policy mismatch: %#v", govulncheck)
	}
	for _, testCase := range []TestCase{staticcheck, govulncheck} {
		if !profileEnabled(testCase, ProfileFull) || !profileEnabled(testCase, ProfileRelease) || profileEnabled(testCase, ProfileSmoke) {
			t.Fatalf("%s profile selection mismatch: %#v", testCase.ID, testCase.Profiles)
		}
	}
}

func TestGovulncheckOnlyDowngradesRecognizableNetworkFailures(t *testing.T) {
	for _, output := range []string{
		`Get "https://vuln.go.dev/index/modules.json": dial tcp: lookup vuln.go.dev: no such host`,
		`proxyconnect tcp: connection refused`,
	} {
		if !isNetworkFailure(output) {
			t.Fatalf("network failure was not recognized: %q", output)
		}
	}
	if isNetworkFailure("Your code is affected by GO-2026-5856; exit status 3") {
		t.Fatal("a real vulnerability must not be downgraded as a network failure")
	}
}

func TestTaskfileEnsureBinUsesChecksumIncrementalBuild(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	start := strings.Index(text, "  ensure-bin:")
	if start < 0 {
		t.Fatal("Taskfile is missing ensure-bin")
	}
	ensureBin := text[start:]
	for _, required := range []string{
		"internal: true",
		"method: checksum",
		"'cmd/**/*.go'",
		"'internal/**/*.go'",
		"go.mod",
		"go.sum",
		"bin/aicoding.exe",
		"go build -o bin/aicoding.exe ./cmd/aicoding",
	} {
		if !strings.Contains(ensureBin, required) {
			t.Fatalf("ensure-bin is missing %q", required)
		}
	}
	if strings.Contains(ensureBin, "go run ./cmd/aicoding bootstrap") {
		t.Fatal("ensure-bin still compiles the CLI through go run")
	}
}

func TestScheduledCIKeepsCleanCloneFullCoverage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "aicoding-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"schedule:",
		"clean-clone-full:",
		"github.event_name == 'workflow_dispatch' || github.event_name == 'schedule'",
		"fresh-clone --profile Full --json",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("scheduled CI is missing clean-clone Full contract %q", required)
		}
	}
}

func TestScheduledCISeedsAndAuditsReleaseBeforeDefaultPromotion(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "aicoding-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"release-gate:",
		"test --profile Release --reuse off --json",
		"test --profile Release --verify-reuse --json",
		"three consecutive successful release-gate runs on main",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("scheduled CI is missing Release reuse promotion contract %q", required)
		}
	}
}

func TestCIPinsEffectiveGoVersionAndEveryActionBySHA(t *testing.T) {
	workflowPaths := []string{
		filepath.Join("..", "..", ".github", "workflows", "aicoding-ci.yml"),
		filepath.Join("..", "..", ".github", "workflows", "issue-governance.yml"),
	}
	pinnedUse := regexp.MustCompile(`^\s*-?\s*uses:\s+[^@\s]+@[0-9a-f]{40}\s+#\s+v[0-9]+\s*$`)
	for _, path := range workflowPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "uses:") && !pinnedUse.MatchString(line) {
				t.Fatalf("workflow action is not pinned by full SHA with a major-version comment: %s", strings.TrimSpace(line))
			}
		}
	}
	data, err := os.ReadFile(workflowPaths[0])
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	if got := strings.Count(workflow, "go-version: '1.26.5'"); got != 3 {
		t.Fatalf("setup-go explicit go-version count = %d, want 3", got)
	}
	if strings.Contains(workflow, "go-version-file:") {
		t.Fatal("CI still uses go-version-file even though setup-go reads the go directive instead of the toolchain directive")
	}
	if got := strings.Count(workflow, "go version"); got != 3 {
		t.Fatalf("effective Go version log count = %d, want 3", got)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestRegistryTitlesPreserveReadableUTF8InJSON(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke})
	if err != nil {
		t.Fatal(err)
	}
	testCases := Registry(cfg)
	if err := validateRegistryTitles(testCases); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"ENV-001": "仓库根目录识别",
		"ENV-002": "Go 版本",
	}
	results := make([]Result, 0, len(testCases))
	for _, testCase := range testCases {
		if !utf8.ValidString(testCase.Title) || strings.ContainsRune(testCase.Title, utf8.RuneError) {
			t.Fatalf("%s title is not readable UTF-8: %q", testCase.ID, testCase.Title)
		}
		results = append(results, Result{ID: testCase.ID, Title: testCase.Title})
	}

	encoded, err := json.Marshal(Report{Results: results})
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, result := range decoded.Results {
		if expected, ok := want[result.ID]; ok && result.Title != expected {
			t.Fatalf("%s title = %q, want %q", result.ID, result.Title, expected)
		}
		delete(want, result.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing expected registry titles after JSON round trip: %#v", want)
	}
}

func TestValidateRegistryTitlesRejectsUnreadableText(t *testing.T) {
	for _, title := range []string{"Go \uFFFD汾", string([]byte{'G', 'o', ' ', 0xff})} {
		err := validateRegistryTitles([]TestCase{{ID: "ENV-002", Title: title}})
		if err == nil {
			t.Fatalf("validateRegistryTitles(%q) succeeded, want error", title)
		}
	}
}

func TestWriteLoadAndLatestDir(t *testing.T) {
	repo := t.TempDir()
	older := filepath.Join(repo, "test-results", "aicoding-global-test-20260101-000000")
	newer := filepath.Join(repo, "test-results", "aicoding-global-test-20260102-000000")
	for index, dir := range []string{older, newer} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		testReport := Report{
			Summary: Summary{Repo: repo, Profile: ProfileFull, Total: 1, Pass: 1, Conclusion: "PASS"},
			Results: []Result{{ID: "FIX-001", Status: Pass, Severity: Required, Profile: ProfileFull}},
		}
		if err := Write(dir, testReport); err != nil {
			t.Fatal(err)
		}
		stamp := time.Now().Add(time.Duration(index) * time.Second)
		if err := os.Chtimes(dir, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}

	latest, err := LatestDir(repo)
	if err != nil || latest != newer {
		t.Fatalf("LatestDir() = %q, %v; want %q", latest, err, newer)
	}
	loaded, err := Load(latest)
	if err != nil || loaded.Summary.Conclusion != "PASS" || len(loaded.Results) != 1 {
		t.Fatalf("unexpected loaded report: %#v, %v", loaded, err)
	}
}

func TestRunCanceledContextStillWritesFailureReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "test-results", "aicoding-global-test-canceled")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	testReport, err := Run(ctx, Config{Repo: t.TempDir(), Out: out, Profile: ProfileSmoke})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
	if testReport.Summary.Conclusion != "FAIL" || ExitCode(testReport, err) != 1 {
		t.Fatalf("unexpected canceled report: %#v", testReport)
	}
	if _, loadErr := Load(out); loadErr != nil {
		t.Fatalf("canceled run must persist its report: %v", loadErr)
	}
}

func TestSuccessfulTestResultRetentionKeepsLatestFive(t *testing.T) {
	repo := t.TempDir()
	for index := 0; index < 8; index++ {
		dir := filepath.Join(repo, "test-results", fmt.Sprintf("aicoding-global-test-%02d", index))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		testReport := Report{Summary: Summary{Conclusion: "PASS"}}
		if err := Write(dir, testReport); err != nil {
			t.Fatal(err)
		}
		stamp := time.Date(2026, 7, 21, 0, index, 0, 0, time.UTC)
		if err := os.Chtimes(filepath.Join(dir, "summary.json"), stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	if err := retainSuccessfulTestResults(repo, Report{Summary: Summary{Conclusion: "PASS"}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(repo, "test-results"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("retained result count = %d, want 5", len(entries))
	}
	if _, err := os.Stat(filepath.Join(repo, "test-results", "aicoding-global-test-07")); err != nil {
		t.Fatalf("latest result was removed: %v", err)
	}
}

func TestFailedRunDoesNotTriggerTestResultRetention(t *testing.T) {
	repo := t.TempDir()
	for index := 0; index < 8; index++ {
		dir := filepath.Join(repo, "test-results", fmt.Sprintf("aicoding-global-test-%02d", index))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Write(dir, Report{Summary: Summary{Conclusion: "PASS"}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := retainSuccessfulTestResults(repo, Report{Summary: Summary{Conclusion: "FAIL", Fail: 1}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(repo, "test-results"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 8 {
		t.Fatalf("failed run triggered retention: retained %d, want 8", len(entries))
	}
}

func TestExecuteRejectsInvalidProfile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Execute([]string{"--profile", "nightly"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Execute invalid profile exit code = %d, want 2", code)
	}
	if stderr.Len() == 0 || stdout.Len() != 0 {
		t.Fatalf("unexpected streams: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLIAndCompatibilityToolUseSingleEngine(t *testing.T) {
	cliSource, err := os.ReadFile(filepath.Join("..", "cli", "test.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"go run ./tools/aicoding-global-tester", "exec.CommandContext"} {
		if strings.Contains(string(cliSource), forbidden) {
			t.Fatalf("internal/cli/test.go still owns a second test runner: %q", forbidden)
		}
	}
	if !strings.Contains(string(cliSource), "testengine.Run") {
		t.Fatal("internal/cli/test.go does not route through testengine.Run")
	}

	toolSource, err := os.ReadFile(filepath.Join("..", "..", "tools", "aicoding-global-tester", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(toolSource), "testengine.Execute") {
		t.Fatal("compatibility tester tool does not route through testengine.Execute")
	}
}
