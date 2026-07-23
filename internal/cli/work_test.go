package cli

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/loopkit/gateref"
	"github.com/JiaxI2/AiCoding/internal/loopkit/transition"
	"github.com/JiaxI2/AiCoding/internal/report/tokenusage"
)

func TestWorkCommandsValidateDecideRecordAndReportStatus(t *testing.T) {
	t.Parallel()
	repo := initWorkTestRepo(t)
	specFile := filepath.Join(repo, "work.json")
	mustWrite(t, specFile, workTestSpec("work-contract", []string{"**"}, nil))

	validated, err := runWork([]string{"validate", "--repo-root", repo, "--file", specFile, "--json"}, time.Now())
	if err != nil || !validated.OK || validated.Command != "work validate" {
		t.Fatalf("validate failed: result=%#v err=%v", validated, err)
	}

	first, err := runWork([]string{"next", "--repo-root", repo, "--file", specFile, "--json"}, time.Now())
	if err != nil || !first.OK {
		t.Fatalf("first next failed: result=%#v err=%v", first, err)
	}
	second, err := runWork([]string{"next", "--repo-root", repo, "--file", specFile, "--json"}, time.Now())
	if err != nil || !second.OK {
		t.Fatalf("second next failed: result=%#v err=%v", second, err)
	}
	firstData, ok := first.Data.(workEvaluationData)
	if !ok {
		t.Fatalf("first next data type = %T", first.Data)
	}
	secondData, ok := second.Data.(workEvaluationData)
	if !ok {
		t.Fatalf("second next data type = %T", second.Data)
	}
	if !reflect.DeepEqual(firstData.Decision, secondData.Decision) || firstData.Decision.State != transition.Continue {
		t.Fatalf("next decision is not deterministic: first=%#v second=%#v", firstData.Decision, secondData.Decision)
	}

	tree := gitWorkTest(t, repo, "rev-parse", "HEAD^{tree}")
	when := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	attempt := transition.Attempt{
		Number: 1, SubjectTreeOID: tree,
		TokenUsage: tokenusage.Usage{InputTokens: 7, OutputTokens: 3, TotalTokens: 10, UpdatedAt: when},
		GateRefs:   []gateref.GateRef{},
		StartedAt:  when, EndedAt: when.Add(time.Second),
	}
	attemptFile := filepath.Join(repo, "attempt.json")
	writeWorkJSON(t, attemptFile, attempt)

	recorded, err := runWork([]string{"record", "--repo-root", repo, "--file", specFile, "--attempt", attemptFile, "--json"}, time.Now())
	if err != nil || !recorded.OK {
		t.Fatalf("record failed: result=%#v err=%v", recorded, err)
	}
	status, err := runWork([]string{"status", "--repo-root", repo, "--file", specFile, "--json"}, time.Now())
	if err != nil || !status.OK {
		t.Fatalf("status failed: result=%#v err=%v", status, err)
	}
	statusData, ok := status.Data.(workEvaluationData)
	if !ok || !statusData.Session.Exists || statusData.Session.Snapshot.Attempts != 1 || len(statusData.Session.History) != 1 {
		t.Fatalf("status did not project recorded history: %#v", status.Data)
	}
	if duplicate, duplicateErr := runWork([]string{"record", "--repo-root", repo, "--file", specFile, "--attempt", attemptFile, "--json"}, time.Now()); duplicateErr == nil || duplicate.OK {
		t.Fatalf("duplicate record unexpectedly succeeded: result=%#v err=%v", duplicate, duplicateErr)
	}
}

func TestWorkDetectsScopeViolationAndRejectsExecutorSubcommands(t *testing.T) {
	t.Parallel()
	repo := initWorkTestRepo(t)
	specFile := filepath.Join(repo, "work.json")
	mustWrite(t, specFile, workTestSpec("scope-contract", []string{"src/**", "work.json"}, []string{"src/private/**"}))
	mustWrite(t, filepath.Join(repo, "src", "private", "secret.txt"), "out of scope\n")

	result, err := runWork([]string{"next", "--repo-root", repo, "--file", specFile, "--json"}, time.Now())
	if err != nil || !result.OK {
		t.Fatalf("next failed: result=%#v err=%v", result, err)
	}
	data := result.Data.(workEvaluationData)
	if data.Decision.State != transition.StopViolation || len(data.Scope.Violations) != 1 || data.Scope.Violations[0] != "src/private/secret.txt" {
		t.Fatalf("scope violation was not adjudicated: %#v", data)
	}
	for _, subcommand := range []string{"run", "prepare", "step"} {
		if _, err := runWork([]string{subcommand}, time.Now()); err == nil {
			t.Fatalf("work %s unexpectedly exists", subcommand)
		}
	}
}

func initWorkTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	initCLITestGitRepo(t, repo)
	gitWorkTest(t, repo, "config", "user.email", "work@example.invalid")
	gitWorkTest(t, repo, "config", "user.name", "Work Test")
	mustWrite(t, filepath.Join(repo, "README.md"), "# fixture\n")
	gitWorkTest(t, repo, "add", "README.md")
	gitWorkTest(t, repo, "commit", "-m", "test: initialize fixture")
	return repo
}

func gitWorkTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeWorkJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, string(data)+"\n")
}

func workTestSpec(id string, allow, deny []string) string {
	data := map[string]any{
		"schemaVersion": 1,
		"id":            id,
		"domain":        "project-development",
		"control": map[string]any{
			"trigger": "explicit",
			"stop": map[string]any{
				"maxAttempts": 3, "maxElapsedSeconds": 300, "maxTotalTokens": 1000,
				"stallThreshold": 2, "contextPressureThreshold": 80,
			},
			"authority": map[string]any{
				"writeScope":    map[string]any{"allow": allow, "deny": deny},
				"requiredGates": []string{"full"}, "checkpoints": []string{"merge"},
			},
		},
		"goal": "exercise bounded-work adjudication", "acceptance": []string{"commands remain deterministic"},
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	return string(raw) + "\n"
}
