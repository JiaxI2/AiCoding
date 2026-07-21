package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func TestSelectChangeProfileUsesHighestMatchedImpactAndSafeDefault(t *testing.T) {
	policy := changeImpactPolicy{
		DefaultProfile: testengine.ProfileFull,
		Rules: []changeImpactRule{
			{Pattern: "docs/**", Profile: testengine.ProfileSmoke, Reason: "docs"},
			{Pattern: "**/*.go", Profile: testengine.ProfileFull, Reason: "go"},
		},
	}
	for _, tc := range []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "docs", paths: []string{"docs/guide.md"}, want: testengine.ProfileSmoke},
		{name: "go dominates", paths: []string{"docs/guide.md", "internal/plan/plan.go"}, want: testengine.ProfileFull},
		{name: "unknown is safe", paths: []string{"unknown.asset"}, want: testengine.ProfileFull},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := selectChangeProfile(policy, tc.paths)
			if err != nil || got != tc.want {
				t.Fatalf("profile = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestChangeVerifySelectsProfileAndPreservesSubsteps(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		content string
		want    string
	}{
		{name: "docs", path: "docs/guide.md", content: "# Guide\n", want: testengine.ProfileSmoke},
		{name: "go", path: "internal/plan/probe.go", content: "package plan\n", want: testengine.ProfileFull},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newValidationCLIRepo(t)
			writeChangeImpactPolicy(t, repo)
			mustValidationCLIGit(t, repo, "add", impactPolicyPath)
			mustValidationCLIGit(t, repo, "commit", "-m", "impact policy")
			mustWrite(t, filepath.Join(repo, filepath.FromSlash(tc.path)), tc.content)
			mustValidationCLIGit(t, repo, "add", tc.path)

			previousProbe := probeChangeReceipt
			previousEngine := runTestEngine
			t.Cleanup(func() {
				probeChangeReceipt = previousProbe
				runTestEngine = previousEngine
			})
			var probedProfile, executedProfile string
			probeChangeReceipt = func(_ string, profile string, subject validationevidence.Subject) (changeReceiptProbe, error) {
				probedProfile = profile
				if subject.Mode != validationevidence.SubjectIndex || !subject.Reusable || subject.TreeOID == "" {
					t.Fatalf("subject = %#v, want reusable INDEX", subject)
				}
				return changeReceiptProbe{
					Subject:     validationevidence.Subject{TreeOID: "tree", Mode: validationevidence.SubjectIndex, Reusable: true},
					Fingerprint: validationevidence.Fingerprint{Identity: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Profile: profile},
					Decision:    validationevidence.ReuseDecision{Code: validationevidence.CodeReceiptMiss, Reason: "missing"},
				}, nil
			}
			runTestEngine = func(_ context.Context, cfg testengine.Config) (testengine.Report, error) {
				executedProfile = cfg.Profile
				return testengine.Report{
					ExecutionMode: "executed",
					Summary:       testengine.Summary{Profile: cfg.Profile, Total: 1, Pass: 1, Conclusion: "PASS"},
					Results:       []testengine.Result{{ID: "FIXTURE-001", Status: testengine.Pass, Reason: "executed", Profile: cfg.Profile}},
				}, nil
			}

			result, err := runChange([]string{"verify", "--staged", "--repo-root", repo}, time.Now())
			if err != nil || !result.OK || probedProfile != tc.want || executedProfile != tc.want {
				t.Fatalf("change verify = %#v, %v; probe=%q run=%q want=%q", result, err, probedProfile, executedProfile, tc.want)
			}
			data, ok := result.Data.(changeVerifyData)
			if !ok || data.ChosenProfile != displayTestProfile(tc.want) || data.ExecutedCases != 1 || data.ReusedCases != 0 || len(data.Steps) != 4 || data.TestReport == nil {
				t.Fatalf("change verify data lost decisions or substeps: %#v", result.Data)
			}
		})
	}
}

func TestChangeVerifyReceiptHitExecutesZeroCases(t *testing.T) {
	repo := newValidationCLIRepo(t)
	writeChangeImpactPolicy(t, repo)
	mustValidationCLIGit(t, repo, "add", impactPolicyPath)
	mustValidationCLIGit(t, repo, "commit", "-m", "impact policy")
	mustWrite(t, filepath.Join(repo, "docs", "guide.md"), "# Guide\n")
	mustValidationCLIGit(t, repo, "add", "docs/guide.md")

	previousProbe := probeChangeReceipt
	previousEngine := runTestEngine
	t.Cleanup(func() {
		probeChangeReceipt = previousProbe
		runTestEngine = previousEngine
	})
	probeChangeReceipt = func(_ string, profile string, _ validationevidence.Subject) (changeReceiptProbe, error) {
		identity := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		return changeReceiptProbe{
			Subject:     validationevidence.Subject{TreeOID: "tree", Mode: validationevidence.SubjectIndex, Reusable: true},
			Fingerprint: validationevidence.Fingerprint{Identity: identity, Profile: profile},
			Decision: validationevidence.ReuseDecision{
				Hit: true, Code: validationevidence.CodeReceiptHit, Reason: "hit",
				Receipt: &validationevidence.Receipt{ReceiptID: identity},
			},
		}, nil
	}
	executions := 0
	runTestEngine = func(_ context.Context, _ testengine.Config) (testengine.Report, error) {
		executions++
		return testengine.Report{}, nil
	}

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"change", "verify", "--staged", "--repo-root", repo, "--json"}, &stdout, &stderr)
	if code != ExitSuccess || executions != 0 {
		t.Fatalf("receipt hit code=%d executions=%d stderr=%s stdout=%s", code, executions, stderr.String(), stdout.String())
	}
	var result report.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	data, ok := result.Data.(map[string]any)
	if !ok || result.Category != report.CategoryNone || result.Retryable || data["executionMode"] != "receipt-hit" || data["executedCases"] != float64(0) {
		t.Fatalf("receipt hit envelope is not machine-decidable: %#v data=%#v", result, data)
	}
}

func TestChangeVerifyStagedRejectsWorktreeOnlyContent(t *testing.T) {
	repo := newValidationCLIRepo(t)
	writeChangeImpactPolicy(t, repo)
	mustValidationCLIGit(t, repo, "add", impactPolicyPath)
	mustValidationCLIGit(t, repo, "commit", "-m", "impact policy")
	mustWrite(t, filepath.Join(repo, "docs", "guide.md"), "# Guide\n")
	mustValidationCLIGit(t, repo, "add", "docs/guide.md")
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "worktree-only\n")

	result, err := runChange([]string{"verify", "--staged", "--repo-root", repo}, time.Now())
	if err == nil || result.OK || result.Category != report.CategoryValidation || result.Retryable || result.NextAction != "git status --short" {
		t.Fatalf("mixed staged/worktree content did not fail closed: %#v err=%v", result, err)
	}
}

func TestEveryJSONResultCarriesClosedDecisionFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"does-not-exist", "--json"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("unknown command exit=%d stderr=%s", code, stderr.String())
	}
	var result report.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Category != report.CategoryUsage || result.Retryable || result.NextAction == "" || !report.ValidCategory(result.Category) {
		t.Fatalf("JSON decision fields are incomplete: %#v", result)
	}
}

func TestKitVerifyUnknownKitReturnsUsageDecision(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"kit", "verify", "--kit", "does-not-exist", "--profile", "Lifecycle", "--repo-root", repo, "--json"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("unknown kit exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var result report.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Category != report.CategoryUsage || result.Retryable || result.NextAction != "aicoding kit list --json" {
		t.Fatalf("unknown kit decision is incomplete: %#v", result)
	}
}

func TestLoadChangeImpactPolicyRejectsUnsafeConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{
			name: "unsupported profile",
			content: `{"schemaVersion":1,"raceScope":{},"changeVerify":{"defaultProfile":"Quick","rules":[{"pattern":"docs/**","profile":"Smoke","reason":"docs"}]}}
`,
		},
		{
			name: "path traversal",
			content: `{"schemaVersion":1,"raceScope":{},"changeVerify":{"defaultProfile":"Full","rules":[{"pattern":"../**","profile":"Smoke","reason":"unsafe"}]}}
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			mustWrite(t, filepath.Join(repo, filepath.FromSlash(impactPolicyPath)), tc.content)
			if _, err := loadChangeImpactPolicy(repo); err == nil {
				t.Fatal("unsafe impact policy was accepted")
			}
		})
	}
}

func writeChangeImpactPolicy(t *testing.T, repo string) {
	t.Helper()
	content := `{
  "schemaVersion": 1,
  "raceScope": {},
  "changeVerify": {
    "defaultProfile": "Full",
    "rules": [
      {"pattern":"docs/**","profile":"Smoke","reason":"docs"},
      {"pattern":"**/*.go","profile":"Full","reason":"go"}
    ]
  }
}
`
	mustWrite(t, filepath.Join(repo, filepath.FromSlash(impactPolicyPath)), content)
}
