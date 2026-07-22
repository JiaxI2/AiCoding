package testengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

func TestVerifyReuseTreatsV1ToolchainReceiptAsOrdinaryMiss(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	defer func() { executeTestCases = originalExecute }()
	executions := 0
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		executions++
		return syntheticResults(cfg, tests, false)
	}

	cfg := evidenceRunConfig(t, repo)
	cfg.Out = filepath.Join(t.TempDir(), "v2-seed")
	seed, err := Run(context.Background(), cfg)
	if err != nil || !seed.Reusable || seed.Summary.Conclusion != "PASS" {
		t.Fatalf("v2 seed = %#v err=%v", seed, err)
	}
	bundle, err := loadEvidenceBundle(cfg.Out)
	if err != nil {
		t.Fatal(err)
	}
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Clean(ProfileSmoke); err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(validationevidence.TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := EvidenceSpec(cfg)
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.Fingerprint(subject, spec)
	if err != nil {
		t.Fatal(err)
	}
	legacy := current
	legacy.ToolchainDigest = testSnapshotDigest(t, "toolchainDigest.v1 legacy identity")
	legacy.Identity = ""
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	legacy.Identity = fmt.Sprintf("sha256:%x", sum)
	if _, err := store.Put(validationevidence.Receipt{
		ValidationIdentity: legacy.Identity, Fingerprint: legacy, Conclusion: "PASS",
		ResultsDigest: seed.ResultsDigest, Reusable: true, Scope: subject.Scope,
	}, bundle); err != nil {
		t.Fatal(err)
	}

	audit := cfg
	audit.Out = filepath.Join(t.TempDir(), "v1-miss-audit")
	audit.VerifyReuse = true
	report, err := Run(context.Background(), audit)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Conclusion != "PASS" || report.ValidationCode == validationevidence.CodeReceiptInvalid ||
		report.ValidationCode == validationevidence.CodeReuseAuditMismatch || executions != 2 {
		t.Fatalf("v1 identity was not an ordinary audit miss: %#v executions=%d", report, executions)
	}
	wantPrefix := string(validationevidence.CodeReceiptMiss) + ":"
	if !strings.HasPrefix(report.Summary.ReceiptInvalidReason, wantPrefix) {
		t.Fatalf("v1 miss reason=%q, want prefix %q", report.Summary.ReceiptInvalidReason, wantPrefix)
	}
	t.Logf("v1-receipt=%s v2-identity=%s verify-reuse=PASS miss=%s", legacy.Identity, current.Identity, report.Summary.ReceiptInvalidReason)
}

func TestNodeReceiptsReuseExpectedDomainsWithOneTreeListing(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	originalList := listValidationTreeEntries
	defer func() {
		executeTestCases = originalExecute
		listValidationTreeEntries = originalList
	}()
	batches := [][]string{}
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		ids := make([]string, 0, len(tests))
		for _, testCase := range tests {
			ids = append(ids, testCase.ID)
		}
		batches = append(batches, ids)
		return syntheticResults(cfg, tests, false)
	}
	treeListings := 0
	listValidationTreeEntries = func(repo, tree string) ([]gitx.TreeEntry, error) {
		treeListings++
		return originalList(repo, tree)
	}

	base := evidenceRunConfig(t, repo)
	base.Profile = ProfileFull
	base.Out = filepath.Join(t.TempDir(), "seed")
	seed, err := Run(context.Background(), base)
	if err != nil || seed.Summary.Conclusion != "PASS" {
		t.Fatalf("seed run = %#v, %v", seed, err)
	}

	writeEngineEvidenceFile(t, repo, "README.md", "docs change\n")
	mustEngineGit(t, repo, "add", "README.md")
	mustEngineGit(t, repo, "commit", "-m", "docs change")
	docs := base
	docs.Out = filepath.Join(t.TempDir(), "docs")
	docs.Reuse = ReuseAuto
	docsReport, err := Run(context.Background(), docs)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 2 {
		t.Fatalf("docs run execution batches = %d, want 2 total", len(batches))
	}
	docsExecuted := stringSet(batches[1])
	for _, id := range []string{"DOC-001", "GIT-002", "ENV-001"} {
		if !docsExecuted[id] {
			t.Fatalf("docs change did not execute invalidated case %s: %v", id, batches[1])
		}
	}
	for _, id := range []string{"GO-001", "LIFE-001"} {
		if docsExecuted[id] {
			t.Fatalf("docs change unnecessarily executed cached case %s: %v", id, batches[1])
		}
	}
	assertNodeReuseReason(t, docsReport, "GO-001", nodeGo)
	assertNodeReuseReason(t, docsReport, "LIFE-001", nodeLifecycleReadonly)
	assertFractionalNodeHit(t, docsReport)

	writeEngineEvidenceFile(t, repo, "internal/example/example.go", "package example\n")
	mustEngineGit(t, repo, "add", "internal/example/example.go")
	mustEngineGit(t, repo, "commit", "-m", "go change")
	goChange := base
	goChange.Out = filepath.Join(t.TempDir(), "go")
	goChange.Reuse = ReuseAuto
	goReport, err := Run(context.Background(), goChange)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 3 {
		t.Fatalf("Go run execution batches = %d, want 3 total", len(batches))
	}
	goExecuted := stringSet(batches[2])
	if !goExecuted["GO-001"] {
		t.Fatalf("Go change reused the go node: %v", batches[2])
	}
	if goExecuted["DOC-001"] {
		t.Fatalf("unrelated Go change invalidated docsync: %v", batches[2])
	}
	assertNodeReuseReason(t, goReport, "DOC-001", nodeDocSync)
	assertFractionalNodeHit(t, goReport)
	if treeListings != 3 {
		t.Fatalf("node input collection used %d tree listings, want one per executed clean tree", treeListings)
	}

	writeEngineEvidenceFile(t, repo, "untracked.txt", "dirty\n")
	dirty := base
	dirty.Out = filepath.Join(t.TempDir(), "dirty")
	dirty.Reuse = ReuseAuto
	dirty.AllowDirty = true
	dirtyReport, err := Run(context.Background(), dirty)
	if err != nil {
		t.Fatal(err)
	}
	if treeListings != 3 {
		t.Fatalf("dirty execution performed a node tree listing: %d", treeListings)
	}
	if dirtyReport.Reusable || dirtyReport.SubjectMode != validationevidence.SubjectDirty || dirtyReport.Summary.CacheHitRatio == nil || *dirtyReport.Summary.CacheHitRatio != 0 {
		t.Fatalf("dirty execution reused or published node evidence: %#v", dirtyReport)
	}
	if len(batches) != 4 || !stringSet(batches[3])["GO-001"] || !stringSet(batches[3])["DOC-001"] {
		t.Fatalf("dirty execution did not run the full registry: %v", batches)
	}
}

func TestVerifyReuseFailsOnCorruptNodeReceipt(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	defer func() { executeTestCases = originalExecute }()
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		return syntheticResults(cfg, tests, false)
	}
	cfg := evidenceRunConfig(t, repo)
	cfg.Profile = ProfileFull
	cfg.Out = filepath.Join(t.TempDir(), "seed")
	if report, err := Run(context.Background(), cfg); err != nil || report.Summary.Conclusion != "PASS" {
		t.Fatalf("seed run = %#v, %v", report, err)
	}
	path := nodeReceiptTestPath(t, repo, cfg, nodeGo)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt["resultsDigest"] = "sha256:" + strings.Repeat("0", 64)
	raw, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	audit := cfg
	audit.Out = filepath.Join(t.TempDir(), "audit")
	audit.VerifyReuse = true
	report, err := Run(context.Background(), audit)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Conclusion != "FAIL" || report.ValidationCode != validationevidence.CodeReuseAuditMismatch || report.Reusable {
		t.Fatalf("corrupt node Receipt passed audit: %#v", report)
	}
	found := false
	for _, result := range report.Results {
		if result.ID == "EVIDENCE-NODE-GO" && result.Status == Fail && strings.Contains(result.Reason, string(validationevidence.CodeReceiptInvalid)) {
			found = true
		}
	}
	if !found {
		t.Fatalf("node corruption did not produce a go audit failure: %#v", report.Results)
	}
}

func TestFailedNodeDoesNotPublishReceipt(t *testing.T) {
	repo := newEngineEvidenceRepo(t)
	originalExecute := executeTestCases
	defer func() { executeTestCases = originalExecute }()
	executeTestCases = func(_ context.Context, cfg Config, tests []TestCase) []Result {
		results := syntheticResults(cfg, tests, false)
		for index := range results {
			if results[index].ID == "GO-001" {
				results[index].Status = Fail
				results[index].ExitCode = 1
				results[index].Reason = "injected go failure"
			}
		}
		return results
	}
	cfg := evidenceRunConfig(t, repo)
	cfg.Profile = ProfileFull
	cfg.Out = filepath.Join(t.TempDir(), "failed")
	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Conclusion != "FAIL" || report.ReceiptID != "" {
		t.Fatalf("failed node produced whole evidence: %#v", report)
	}
	if path := nodeReceiptTestPath(t, repo, cfg, nodeGo); pathExists(path) {
		t.Fatalf("failed go node produced a Receipt: %s", path)
	}
}

func TestRegistryNodeAssignmentsRemainConservative(t *testing.T) {
	cfg := evidenceRunConfig(t, newEngineEvidenceRepo(t))
	found := map[string]TestCase{}
	for _, testCase := range Registry(cfg) {
		found[testCase.ID] = testCase
	}
	for _, item := range []struct {
		id   string
		node string
	}{
		{"GO-001", nodeGo}, {"GO-006", nodeGo}, {"DOC-001", nodeDocSync}, {"DOC-004", nodeDocSync},
		{"GIT-002", nodeGovernance}, {"GIT-009", nodeGovernance}, {"LIFE-001", nodeLifecycleReadonly},
		{"LIFE-007", nodeLifecycleReadonly}, {"RC-001", nodeLifecycleReadonly}, {"RC-002", nodeLifecycleReadonly},
	} {
		if found[item.id].Node != item.node {
			t.Fatalf("%s node = %q, want %q", item.id, found[item.id].Node, item.node)
		}
	}
	for _, id := range []string{"ENV-001", "GO-007", "DOC-003", "GIT-001"} {
		if node, err := validationNode(found[id].Node); err != nil || node != nodeRepo {
			t.Fatalf("unmarked case %s does not fail closed to repo: %q, %v", id, node, err)
		}
	}
	for _, check := range []struct {
		node string
		path string
		want bool
	}{
		{nodeGo, "README.md", false}, {nodeLifecycleReadonly, "README.md", false},
		{nodeDocSync, "README.md", true}, {nodeGovernance, "README.md", true},
		{nodeGo, "internal/example/example.go", true}, {nodeDocSync, "internal/example/example.go", false},
	} {
		if got := validationNodeOwnsPath(check.node, check.path); got != check.want {
			t.Fatalf("node path ownership %s %s = %v, want %v", check.node, check.path, got, check.want)
		}
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

func writeEngineEvidenceFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func assertNodeReuseReason(t *testing.T, report Report, id, node string) {
	t.Helper()
	for _, result := range report.Results {
		if result.ID == id {
			if result.Reason != "reused-from-node:"+node {
				t.Fatalf("%s reason = %q, want node %s", id, result.Reason, node)
			}
			return
		}
	}
	t.Fatalf("report is missing %s", id)
}

func assertFractionalNodeHit(t *testing.T, report Report) {
	t.Helper()
	if report.ExecutionMode != "executed" || report.Summary.CacheHitRatio == nil || *report.Summary.CacheHitRatio <= 0 || *report.Summary.CacheHitRatio >= 1 {
		t.Fatalf("node reuse did not report a fractional executed hit: %#v", report)
	}
}

func nodeReceiptTestPath(t *testing.T, repo string, cfg Config, node string) string {
	t.Helper()
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(validationevidence.TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := EvidenceSpec(cfg)
	if err != nil {
		t.Fatal(err)
	}
	whole, err := store.Fingerprint(subject, spec)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := gitx.TreeEntries(repo, subject.TreeOID)
	if err != nil {
		t.Fatal(err)
	}
	inputDigest, err := validationNodeInputDigest(node, entries)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := store.DeriveNodeFingerprint(whole, node, inputDigest)
	if err != nil {
		t.Fatal(err)
	}
	commonDir, err := gitx.CommonDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(commonDir, "aicoding", "validation", "receipts", cfg.Profile, "nodes", node, strings.TrimPrefix(fingerprint.Identity, "sha256:")+".json")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
