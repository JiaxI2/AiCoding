package kit

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
