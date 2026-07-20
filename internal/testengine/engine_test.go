package testengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	for _, id := range []string{"ENV-001", "GO-001", "GIT-009", "FRESH-001", "FRESH-002", "REL-002"} {
		if !seen[id] {
			t.Fatalf("registry is missing %s", id)
		}
	}
	for _, removed := range []string{"FULL-001", "REL-001"} {
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
