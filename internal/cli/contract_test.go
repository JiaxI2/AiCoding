package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/report"
)

func TestExecuteHelpAndUsageExitCodes(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Execute(nil, &stdout, &stderr); code != ExitUsage {
		t.Fatalf("no arguments exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("no arguments must print usage to stderr: %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--help"}, &stdout, &stderr); code != ExitSuccess {
		t.Fatalf("--help exit code = %d, want %d", code, ExitSuccess)
	}
	if !strings.Contains(stdout.String(), "Formal product workflow:") || stderr.Len() != 0 {
		t.Fatalf("unexpected help streams: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"unknown"}, &stdout, &stderr); code != ExitUsage {
		t.Fatalf("unknown command exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "unknown command: unknown") {
		t.Fatalf("unknown command diagnostic missing: %q", stderr.String())
	}
}

func TestExecuteFlagHelpAndJSONUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Execute([]string{"bootstrap", "--help"}, &stdout, &stderr); code != ExitSuccess {
		t.Fatalf("bootstrap --help exit code = %d, want %d", code, ExitSuccess)
	}
	if !strings.Contains(stdout.String(), "Usage: aicoding bootstrap [options]") || stderr.Len() != 0 {
		t.Fatalf("unexpected command help streams: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"lifecycle", "--help"}, &stdout, &stderr); code != ExitSuccess {
		t.Fatalf("lifecycle --help exit code = %d, want %d", code, ExitSuccess)
	}
	if !strings.Contains(stdout.String(), "aicoding lifecycle plan") || stderr.Len() != 0 {
		t.Fatalf("unexpected namespace help streams: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Execute([]string{"bootstrap", "--unknown", "--json"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("invalid flag exit code = %d, want %d; stdout=%q stderr=%q", code, ExitUsage, stdout.String(), stderr.String())
	}
	var res report.Result
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("invalid flag must preserve JSON-only stdout: %v: %q", err, stdout.String())
	}
	if res.OK || len(res.Errors) == 0 || stderr.Len() != 0 {
		t.Fatalf("unexpected usage result: res=%#v stderr=%q", res, stderr.String())
	}
}

func TestUnsupportedSubcommandsUseExitTwoWithoutExecuting(t *testing.T) {
	for _, args := range [][]string{
		{"governance", "unknown", "--json"},
		{"lifecycle", "unknown", "--json"},
		{"mcp", "unknown", "--json"},
		{"kit", "unknown", "--json"},
		{"verify", "unknown", "--json"},
		{"doctor", "unknown", "--json"},
		{"cache", "unknown", "--json"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Execute(args, &stdout, &stderr); code != ExitUsage {
			t.Fatalf("Execute(%#v) exit code = %d, want %d; stdout=%q stderr=%q", args, code, ExitUsage, stdout.String(), stderr.String())
		}
		var res report.Result
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			t.Fatalf("Execute(%#v) must emit JSON usage result: %v: %q", args, err, stdout.String())
		}
		if res.OK || stderr.Len() != 0 {
			t.Fatalf("unexpected usage result for %#v: res=%#v stderr=%q", args, res, stderr.String())
		}
	}
}

func TestExecuteExecutionFailureUsesExitOne(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	code := Execute([]string{"codex", "usage", "parse", "--file", missing, "--json"}, &stdout, &stderr)
	if code != ExitFailure {
		t.Fatalf("execution failure exit code = %d, want %d; stdout=%q stderr=%q", code, ExitFailure, stdout.String(), stderr.String())
	}
	var res report.Result
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("execution failure must emit JSON: %v: %q", err, stdout.String())
	}
	if res.OK || stderr.Len() != 0 {
		t.Fatalf("unexpected execution failure result: res=%#v stderr=%q", res, stderr.String())
	}
}

func TestDeprecationContract(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		canonical string
	}{
		{[]string{"smoke"}, "aicoding test --profile Smoke"},
		{[]string{"ci", "--profile", "Release"}, "aicoding test --profile Release"},
		{[]string{"full"}, "aicoding test --profile Full"},
		{[]string{"test", "full"}, "aicoding test --profile Full"},
		{[]string{"test", "release"}, "aicoding test --profile Release"},
	} {
		canonical, ok := deprecatedCommand(tc.args)
		if !ok || canonical != tc.canonical {
			t.Fatalf("deprecatedCommand(%#v) = %q, %t; want %q, true", tc.args, canonical, ok, tc.canonical)
		}
	}

	res := addDeprecation(report.Result{SchemaVersion: 1, OK: true}, "aicoding test --profile Smoke")
	var out bytes.Buffer
	report.WriteTextTo(&out, res)
	if !strings.Contains(out.String(), "CLI_DEPRECATED: use aicoding test --profile Smoke") {
		t.Fatalf("text report does not expose deprecation warning: %q", out.String())
	}
}

func TestNormalizeTestProfile(t *testing.T) {
	for _, tc := range []struct {
		input   string
		runner  string
		display string
	}{
		{"Smoke", "smoke", "Smoke"},
		{"full", "full", "Full"},
		{"RELEASE", "release", "Release"},
	} {
		runner, display, err := normalizeTestProfile(tc.input)
		if err != nil || runner != tc.runner || display != tc.display {
			t.Fatalf("normalizeTestProfile(%q) = %q, %q, %v", tc.input, runner, display, err)
		}
	}
	if _, _, err := normalizeTestProfile("nightly"); !isUsageError(err) {
		t.Fatalf("invalid profile must be a usage error: %v", err)
	}
}
