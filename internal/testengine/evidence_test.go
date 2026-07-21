package testengine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/registry"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func TestRunCreatesReusesForcesAndAuditsReceipt(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	defer func() { executeTestCases = originalExecute }()
	executions := 0
	warnExecution := false
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		executions++
		results := syntheticResults(cfg, tests, false)
		if warnExecution {
			for index, testCase := range tests {
				if profileEnabled(testCase, cfg.Profile) && testCase.Severity == WarnOnly {
					results[index].Status = Warn
					results[index].Reason = "injected warning"
					break
				}
			}
		}
		return results
	}

	base := evidenceRunConfig(t, repo)
	base.Out = filepath.Join(t.TempDir(), "executed")
	first, err := Run(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutionMode != "executed" || !first.Reusable || first.ReceiptID == "" || first.ValidationIdentity == "" || executions != 1 {
		t.Fatalf("first execution did not create evidence: %#v executions=%d", first, executions)
	}
	if first.Summary.CacheHitRatio == nil || *first.Summary.CacheHitRatio != 0 {
		t.Fatalf("executed cache hit ratio = %#v", first.Summary.CacheHitRatio)
	}
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	head, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	zero := strings.Repeat("0", len(head))
	gate := store.GatePush(validationevidence.Policy{SchemaVersion: 1, UnmatchedAction: "allow", Contexts: []validationevidence.PushContext{{
		ID: "stable", RemoteRef: "refs/heads/main", RequiredProfile: ProfileSmoke,
	}}}, []gitx.PushUpdate{{LocalRef: "refs/heads/main", LocalOID: head, RemoteRef: "refs/heads/main", RemoteOID: zero}})
	if !gate.OK || gate.Updates[0].Code != validationevidence.CodeReceiptHit {
		t.Fatalf("executed HEAD Receipt did not bind its commit alias: %#v", gate)
	}

	auto := base
	auto.Out = filepath.Join(t.TempDir(), "reused")
	auto.Reuse = ReuseAuto
	reused, err := Run(context.Background(), auto)
	if err != nil {
		t.Fatal(err)
	}
	if reused.ExecutionMode != "reused" || reused.ReceiptID != first.ReceiptID || reused.Summary.Conclusion != "PASS" || executions != 1 {
		t.Fatalf("auto reuse did not short-circuit: %#v executions=%d", reused, executions)
	}
	if reused.Summary.CacheHitRatio == nil || *reused.Summary.CacheHitRatio != 1 {
		t.Fatalf("reused cache hit ratio = %#v", reused.Summary.CacheHitRatio)
	}
	loaded, err := Load(auto.Out)
	if err != nil || loaded.ExecutionMode != "reused" || loaded.ReceiptID != first.ReceiptID {
		t.Fatalf("reused report was not persisted as a view: %#v %v", loaded, err)
	}

	forced := auto
	forced.Out = filepath.Join(t.TempDir(), "forced")
	forced.Force = true
	forcedReport, err := Run(context.Background(), forced)
	if err != nil {
		t.Fatal(err)
	}
	if forcedReport.ExecutionMode != "executed" || executions != 2 {
		t.Fatalf("--force did not execute: %#v executions=%d", forcedReport, executions)
	}

	warnExecution = true
	audit := base
	audit.Out = filepath.Join(t.TempDir(), "audit")
	audit.VerifyReuse = true
	audited, err := Run(context.Background(), audit)
	if err != nil {
		t.Fatal(err)
	}
	if audited.ExecutionMode != "executed" || audited.Summary.Conclusion != "FAIL" || audited.Summary.Warn != 1 || audited.ResultsDigest == first.ResultsDigest || audited.ValidationCode != validationevidence.CodeReuseAuditMismatch || audited.Reusable || executions != 3 {
		t.Fatalf("--verify-reuse missed polluted evidence: %#v executions=%d", audited, executions)
	}

	plain := base
	plain.Out = filepath.Join(t.TempDir(), "plain-status-drift")
	drifted, err := Run(context.Background(), plain)
	if err != nil {
		t.Fatal(err)
	}
	if drifted.Summary.Conclusion != "PASS_WITH_WARNINGS" || drifted.ReceiptID != "" || drifted.Reusable || drifted.ValidationCode != validationevidence.CodeReuseAuditMismatch || executions != 4 {
		t.Fatalf("status drift replaced or claimed the existing Receipt: %#v executions=%d", drifted, executions)
	}
}

func TestAutoMissReportsReceiptInvalidReason(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	defer func() { executeTestCases = originalExecute }()
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		return syntheticResults(cfg, tests, false)
	}

	cfg := evidenceRunConfig(t, repo)
	cfg.Out = filepath.Join(t.TempDir(), "auto-miss")
	cfg.Reuse = ReuseAuto
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.CacheHitRatio == nil || *report.Summary.CacheHitRatio != 0 {
		t.Fatalf("auto miss cache hit ratio = %#v", report.Summary.CacheHitRatio)
	}
	wantPrefix := string(validationevidence.CodeReceiptMiss) + ":"
	if !strings.HasPrefix(report.Summary.ReceiptInvalidReason, wantPrefix) {
		t.Fatalf("receipt invalid reason = %q, want prefix %q", report.Summary.ReceiptInvalidReason, wantPrefix)
	}
}

func TestResultStatusDigestIsOrderedProfileScopedAndStatusSensitive(t *testing.T) {
	cfg := Config{Profile: ProfileSmoke}
	tests := []TestCase{
		{ID: "A", Profiles: []Profile{ProfileSmoke}},
		{ID: "B", Profiles: []Profile{ProfileSmoke}},
		{ID: "OTHER", Profiles: []Profile{ProfileFull}},
	}
	first, err := resultStatusDigest(cfg, tests, []Result{{ID: "B", Status: Pass}, {ID: "OTHER", Status: Skip}, {ID: "A", Status: Pass}})
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := resultStatusDigest(cfg, tests, []Result{{ID: "A", Status: Pass}, {ID: "OTHER", Status: Fail}, {ID: "B", Status: Pass}})
	if err != nil {
		t.Fatal(err)
	}
	if reordered != first {
		t.Fatal("result order or an unselected profile changed the status digest")
	}
	changed, err := resultStatusDigest(cfg, tests, []Result{{ID: "A", Status: Pass}, {ID: "B", Status: Warn}})
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("PASS to WARN did not change the per-case status digest")
	}
}

func TestFailNeverCreatesReceipt(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	defer func() { executeTestCases = originalExecute }()
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		return syntheticResults(cfg, tests, true)
	}
	cfg := evidenceRunConfig(t, repo)
	cfg.Out = filepath.Join(t.TempDir(), "failed")
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Conclusion != "FAIL" || report.ReceiptID != "" || report.Reusable {
		t.Fatalf("failed execution produced reusable evidence: %#v", report)
	}
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := store.List(ProfileSmoke)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 0 {
		t.Fatalf("FAIL wrote %d Receipts", len(receipts))
	}
}

func TestContentDriftPreservesConclusionButDisablesReceipt(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	originalCapture := captureValidationSubject
	defer func() {
		executeTestCases = originalExecute
		captureValidationSubject = originalCapture
	}()
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		return syntheticResults(cfg, tests, false)
	}
	captures := 0
	captureValidationSubject = func(store validationevidence.Repository) (validationevidence.Subject, error) {
		subject, err := store.Capture(validationevidence.TargetAuto)
		captures++
		if err == nil && captures == 2 {
			subject.TreeOID = strings.Repeat("a", 40)
		}
		return subject, err
	}
	cfg := evidenceRunConfig(t, repo)
	cfg.Out = filepath.Join(t.TempDir(), "drift")
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Conclusion != "PASS" || report.Reusable || report.ReceiptID != "" || report.ValidationCode != validationevidence.CodeContentChangedDuringRun {
		t.Fatalf("drift handling changed conclusion or created evidence: %#v", report)
	}
}

func TestEvidenceSpecIsPathStableAndTracksSemantics(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	mustEngineGit(t, repo, "worktree", "add", "--detach", linked, "HEAD")
	base := evidenceRunConfig(t, repo)
	first, err := EvidenceSpec(base)
	if err != nil {
		t.Fatal(err)
	}
	linkedConfig := base
	linkedConfig.Repo = linked
	second, err := EvidenceSpec(linkedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if first.ValidationPlanDigest != second.ValidationPlanDigest || first.EngineSemanticDigest != second.EngineSemanticDigest || first.OptionsDigest != second.OptionsDigest {
		t.Fatalf("worktree path changed semantic digests: %#v %#v", first, second)
	}
	full := base
	full.Profile = ProfileFull
	fullSpec, err := EvidenceSpec(full)
	if err != nil {
		t.Fatal(err)
	}
	if fullSpec.ValidationPlanDigest == first.ValidationPlanDigest {
		t.Fatal("profile-selected Registry change did not change the plan digest")
	}
	strict := base
	strict.Strict = true
	strictSpec, err := EvidenceSpec(strict)
	if err != nil {
		t.Fatal(err)
	}
	if strictSpec.OptionsDigest == first.OptionsDigest {
		t.Fatal("strict option did not change options digest")
	}
	changedCatalog := base
	changedCatalog.CommandCatalogDigest = testSnapshotDigest(t, "changed-catalog")
	changedSpec, err := EvidenceSpec(changedCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if changedSpec.EngineSemanticDigest == first.EngineSemanticDigest {
		t.Fatal("command catalog change did not change engine semantics")
	}
	changedImpl, err := engineSemanticDigest(base.CommandCatalogDigest, first.ValidationPlanDigest, evidenceImplVersion+1)
	if err != nil {
		t.Fatal(err)
	}
	if changedImpl == first.EngineSemanticDigest {
		t.Fatal("implementation version change did not change engine semantics")
	}
}

func TestReceiptEligibilityUsesSeverityAndUnexpectedSkipPolicy(t *testing.T) {
	cfg := Config{Profile: ProfileSmoke}
	subject := validationevidence.Subject{Reusable: true}
	tests := []TestCase{
		{ID: "REQ", Severity: Required, Profiles: []Profile{ProfileSmoke}},
		{ID: "WARN", Severity: WarnOnly, Profiles: []Profile{ProfileSmoke}},
		{ID: "OTHER", Severity: Required, Profiles: []Profile{ProfileFull}},
	}
	results := []Result{{ID: "REQ", Status: Pass}, {ID: "WARN", Status: Warn}, {ID: "OTHER", Status: Skip, Reason: "not selected by profile"}}
	if ok, reason, _ := receiptEligible(cfg, tests, results, subject); !ok {
		t.Fatalf("non-blocking WARN or profile skip blocked Receipt: %s", reason)
	}
	results[0].Status = Skip
	if ok, _, _ := receiptEligible(cfg, tests, results, subject); ok {
		t.Fatal("required SKIP produced a Receipt")
	}
	results[0].Status = Pass
	results[1].Status = Skip
	if ok, _, _ := receiptEligible(cfg, tests, results, subject); ok {
		t.Fatal("unexpected selected SKIP produced a Receipt")
	}
	tests[1].OptionalPath = "optional/tool"
	if ok, reason, _ := receiptEligible(cfg, tests, results, subject); !ok {
		t.Fatalf("declared optional-path SKIP blocked Receipt: %s", reason)
	}
}

func TestNormalizeConfigDefaultsReuseOffAndRejectsAuditForce(t *testing.T) {
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Reuse != ReuseOff {
		t.Fatalf("default reuse = %q", cfg.Reuse)
	}
	if _, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke, Reuse: "always"}); err == nil {
		t.Fatal("invalid reuse mode was accepted")
	}
	if _, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke, Force: true, VerifyReuse: true}); err == nil {
		t.Fatal("--force and --verify-reuse were accepted together")
	}
	var stderr bytes.Buffer
	parsed, err := ParseConfig([]string{"--repo", t.TempDir(), "--profile", "smoke", "--reuse", "auto", "--force", "--allow-dirty"}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Reuse != ReuseAuto || !parsed.Force || !parsed.AllowDirty {
		t.Fatalf("evidence flags were not parsed: %#v", parsed)
	}
}

func evidenceRunConfig(t *testing.T, repo string) Config {
	t.Helper()
	return Config{
		Repo: repo, Profile: ProfileSmoke, Timeout: time.Second, LongTimeout: 2 * time.Second,
		Concurrency: 1, Reuse: ReuseOff, CommandCatalogDigest: testSnapshotDigest(t, "catalog"),
	}
}

func syntheticResults(cfg Config, tests []TestCase, fail bool) []Result {
	results := make([]Result, 0, len(tests))
	failed := false
	for _, testCase := range tests {
		result := Result{ID: testCase.ID, Category: testCase.Category, Title: testCase.Title, Severity: testCase.Severity, Status: Pass, Profile: cfg.Profile}
		if !profileEnabled(testCase, cfg.Profile) {
			result.Status = Skip
			result.Reason = "not selected by profile"
		} else if fail && !failed && testCase.Severity == Required {
			result.Status = Fail
			result.ExitCode = 1
			result.Reason = "injected failure"
			failed = true
		}
		results = append(results, result)
	}
	return results
}

func newEngineEvidenceRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustEngineGit(t, repo, "init", "-q")
	mustEngineGit(t, repo, "config", "user.email", "test@example.com")
	mustEngineGit(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustEngineGit(t, repo, "add", "tracked.txt")
	mustEngineGit(t, repo, "commit", "-m", "initial")
	return repo
}

func mustEngineGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := gitx.Run(repo, args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(out)
}

func testSnapshotDigest(t *testing.T, value string) string {
	t.Helper()
	snapshot, err := registry.NewSnapshot("test", value)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Digest()
}
