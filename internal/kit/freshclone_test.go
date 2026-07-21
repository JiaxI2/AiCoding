package kit

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

func TestFreshCloneChecksAreLeafCommands(t *testing.T) {
	bin := `C:\\repo\\bin\\aicoding.exe`
	for _, tc := range []struct {
		profile string
		want    [][]string
	}{
		{"Smoke", [][]string{{bin, "version"}}},
		{"Full", [][]string{{"go", "test", "./..."}}},
		{"Release", [][]string{{bin, "release", "verify", "--json"}}},
	} {
		got, err := freshCloneChecks(bin, tc.profile)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("freshCloneChecks(%q) = %#v, %v; want %#v", tc.profile, got, err, tc.want)
		}
	}
	if _, err := freshCloneChecks(bin, "Nightly"); err == nil {
		t.Fatal("unsupported fresh-clone profile must fail")
	}
}

func TestFreshCloneDoesNotRepeatSubmoduleInitialization(t *testing.T) {
	command := strings.Join(freshCloneSubmoduleArgs(), " ")
	if strings.Contains(command, "update") || strings.Contains(command, "--init") {
		t.Fatalf("fresh clone repeats submodule initialization: %s", command)
	}
	if command != "git submodule status --recursive" {
		t.Fatalf("unexpected submodule verification command: %s", command)
	}

	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckFreshCloneContract(repo); err != nil {
		t.Fatal(err)
	}
}

func TestFreshCloneStepElapsedMSIsAlwaysSerialized(t *testing.T) {
	payload, err := json.Marshal(FreshCloneStep{Name: "git.clone", OK: true, Message: "passed", ElapsedMS: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"elapsed_ms":0`) {
		t.Fatalf("elapsed_ms is missing from FreshCloneStep JSON: %s", payload)
	}
}

func TestFreshCloneFailureRetainsRegisteredEvidence(t *testing.T) {
	repo := t.TempDir()
	command := exec.Command("git", "init", "-q")
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v: %s", err, output)
	}
	report := FreshClone(repo, "Smoke", false)
	if report.OK || !report.KeptTemp || report.TempRoot == "" {
		t.Fatalf("fresh-clone failure was not retained: %#v", report)
	}
	t.Cleanup(func() { _ = platform.ReleaseTempDir(repo, report.TempRoot, "fresh-clone") })
	if _, err := os.Stat(report.TempRoot); err != nil {
		t.Fatalf("failed evidence directory is missing: %v", err)
	}
	records, err := platform.ReadTempLedger(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 2 || records[len(records)-1].Outcome != "failed" || records[len(records)-1].Path != report.TempRoot {
		t.Fatalf("failed outcome missing from ledger: %#v", records)
	}
	t.Logf("failure retained=%t outcome=%s path=%s", report.KeptTemp, records[len(records)-1].Outcome, report.TempRoot)
}
