package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/repohealth"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func TestProductDoctorAndStatusCompatibilityUseSameChecks(t *testing.T) {
	previous := productDoctorChecks
	t.Cleanup(func() { productDoctorChecks = previous })
	calls := 0
	productDoctorChecks = func(context.Context, string, repohealth.ProductOptions) []report.Check {
		calls++
		return []report.Check{{
			ID:       "doctor.fixture",
			Category: "TEST",
			OK:       true,
			Status:   "PASS",
		}}
	}
	repo := t.TempDir()

	var doctorOut bytes.Buffer
	var doctorErr bytes.Buffer
	if code := Execute([]string{"doctor", "--all", "--repo-root", repo, "--json"}, &doctorOut, &doctorErr); code != ExitSuccess {
		t.Fatalf("doctor exit code = %d: stdout=%q stderr=%q", code, doctorOut.String(), doctorErr.String())
	}
	var doctor report.Result
	if err := json.Unmarshal(doctorOut.Bytes(), &doctor); err != nil {
		t.Fatal(err)
	}
	standard, ok := doctor.Data.(map[string]interface{})
	if !ok || standard["schemaVersion"] != float64(report.SchemaVersion) || standard["status"] != "PASS" {
		t.Fatalf("unexpected doctor standard report: %#v", doctor.Data)
	}

	var statusOut bytes.Buffer
	var statusErr bytes.Buffer
	if code := Execute([]string{"status", "--all", "--repo-root", repo, "--json"}, &statusOut, &statusErr); code != ExitSuccess {
		t.Fatalf("status exit code = %d: stdout=%q stderr=%q", code, statusOut.String(), statusErr.String())
	}
	var status report.Result
	if err := json.Unmarshal(statusOut.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !containsMessage(status.Warnings, "CLI_DEPRECATED: use aicoding doctor --all") {
		t.Fatalf("status compatibility did not route to product doctor: calls=%d result=%#v", calls, status)
	}
}

func TestProductVerifyFailureUsesValidationErrorKind(t *testing.T) {
	previous := productVerifyChecks
	t.Cleanup(func() { productVerifyChecks = previous })
	productVerifyChecks = func(_ context.Context, _ string, opts repohealth.ProductOptions) []report.Check {
		if opts.Profile != "Smoke" {
			t.Fatalf("profile = %q, want Smoke", opts.Profile)
		}
		return []report.Check{{
			ID:       "verify.fixture",
			Category: "TEST",
			OK:       false,
			Status:   "FAIL",
			Errors:   []string{"fixture failed"},
		}}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Execute([]string{"verify", "--profile", "Smoke", "--repo-root", t.TempDir(), "--json"}, &stdout, &stderr)
	if code != ExitFailure || stderr.Len() != 0 {
		t.Fatalf("verify exit code = %d: stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var result report.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.OK || result.ErrorKind != report.ErrorKindValidation {
		t.Fatalf("unexpected verify result: %#v", result)
	}
	data, ok := result.Data.(map[string]interface{})
	if !ok || data["status"] != "FAIL" || data["profile"] != "Smoke" {
		t.Fatalf("unexpected verify standard report: %#v", result.Data)
	}
}

func TestUnknownJSONCommandAndMissingProfileUseUsageErrorKind(t *testing.T) {
	for _, args := range [][]string{
		{"unknown", "--json"},
		{"verify", "--json"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Execute(args, &stdout, &stderr); code != ExitUsage {
			t.Fatalf("Execute(%#v) exit code = %d: stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
		var result report.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("Execute(%#v) returned invalid JSON: %v: %q", args, err, stdout.String())
		}
		if result.ErrorKind != report.ErrorKindUsage || stderr.Len() != 0 {
			t.Fatalf("unexpected usage result for %#v: %#v stderr=%q", args, result, stderr.String())
		}
	}
}

func containsMessage(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
