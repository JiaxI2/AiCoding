package cli

import (
	"context"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func TestValidationExplainHitAndTreeOnlyMissAreDeterministic(t *testing.T) {
	repo := newValidationCLIRepo(t)
	_, fingerprint := putValidationCLIReceipt(t, repo, validationevidence.TargetHead)

	hit, err := runValidation([]string{"explain", "--profile", "Smoke", "--target", "HEAD", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !hit.OK {
		t.Fatalf("explain hit = %#v, %v", hit, err)
	}
	hitData := hit.Data.(validationExplainData)
	if hitData.Decision != "hit" || hitData.ReferenceIdentity != fingerprint.Identity || len(hitData.Changed) != 0 || len(hitData.Unchanged) != 8 {
		t.Fatalf("explain hit data = %#v", hitData)
	}

	mustWrite(t, filepath.Join(repo, "tracked.txt"), "docs-only diagnostic change\n")
	mustValidationCLIGit(t, repo, "add", "tracked.txt")
	args := []string{"explain", "--profile", "Smoke", "--target", "INDEX", "--repo-root", repo, "--json"}
	first, err := runValidation(args, time.Now())
	if err != nil || !first.OK {
		t.Fatalf("explain miss = %#v, %v", first, err)
	}
	second, err := runValidation(args, time.Now())
	if err != nil || !second.OK {
		t.Fatalf("second explain miss = %#v, %v", second, err)
	}
	firstData := first.Data.(validationExplainData)
	secondData := second.Data.(validationExplainData)
	if !reflect.DeepEqual(firstData, secondData) {
		t.Fatalf("explain data is not deterministic:\nfirst=%#v\nsecond=%#v", firstData, secondData)
	}
	if firstData.Decision != "miss" || firstData.CheckCode != validationevidence.CodeReceiptMiss || len(firstData.Changed) != 1 || firstData.Changed[0].Field != "subjectTreeOID" {
		t.Fatalf("tree-only miss was not explained precisely: %#v", firstData)
	}
	if len(firstData.Unchanged) != 7 {
		t.Fatalf("unchanged fingerprint fields = %#v", firstData.Unchanged)
	}
}

func TestValidationCommandsCheckListStatusAndCleanReceipts(t *testing.T) {
	repo := newValidationCLIRepo(t)
	start := time.Now()

	miss, err := runValidation([]string{"check", "--profile", "Smoke", "--target", "HEAD", "--repo-root", repo, "--json"}, start)
	if err == nil || !report.IsValidationError(err) || miss.OK {
		t.Fatalf("initial check = %#v, %v", miss, err)
	}
	missData := miss.Data.(validationCheckData)
	if missData.Decision.Code != validationevidence.CodeReceiptMiss {
		t.Fatalf("initial decision = %#v", missData.Decision)
	}

	store, fingerprint := putValidationCLIReceipt(t, repo, validationevidence.TargetHead)
	hit, err := runValidation([]string{"check", "--profile", "Smoke", "--target", "HEAD", "--repo-root", repo, "--json"}, start)
	if err != nil || !hit.OK || hit.InputDigest != fingerprint.Identity {
		t.Fatalf("matching check = %#v, %v", hit, err)
	}
	hitData := hit.Data.(validationCheckData)
	if !hitData.Decision.Hit || hitData.Decision.Code != validationevidence.CodeReceiptHit {
		t.Fatalf("matching decision = %#v", hitData.Decision)
	}

	status, err := runValidation([]string{"status", "--repo-root", repo, "--json"}, start)
	if err != nil || !status.OK || status.Data.(validationStatusData).ReceiptCount != 1 {
		t.Fatalf("status = %#v, %v", status, err)
	}
	listed, err := runValidation([]string{"list", "--profile", "Smoke", "--repo-root", repo, "--json"}, start)
	if err != nil || !listed.OK || listed.Data.(validationListData).ReceiptCount != 1 {
		t.Fatalf("list = %#v, %v", listed, err)
	}

	cleaned, err := runValidation([]string{"clean", "--profile", "Smoke", "--repo-root", repo, "--json"}, start)
	if err != nil || !cleaned.OK || cleaned.Data.(validationCleanData).RemovedReceipts != 1 {
		t.Fatalf("clean = %#v, %v", cleaned, err)
	}
	if receipts, listErr := store.List("smoke"); listErr != nil || len(receipts) != 0 {
		t.Fatalf("receipts after clean = %#v, %v", receipts, listErr)
	}
}

func TestValidationIndexReceiptSurvivesCommitAndMessageAmend(t *testing.T) {
	repo := newValidationCLIRepo(t)
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "staged\n")
	mustValidationCLIGit(t, repo, "add", "tracked.txt")
	_, fingerprint := putValidationCLIReceipt(t, repo, validationevidence.TargetIndex)

	for _, target := range []string{"INDEX"} {
		result, err := runValidation([]string{"check", "--profile", "Smoke", "--target", target, "--repo-root", repo, "--json"}, time.Now())
		if err != nil || !result.OK || result.InputDigest != fingerprint.Identity {
			t.Fatalf("%s check before commit = %#v, %v", target, result, err)
		}
	}
	mustValidationCLIGit(t, repo, "commit", "-m", "staged tree")
	for _, message := range []string{"after commit", "after message-only amend"} {
		if message == "after message-only amend" {
			mustValidationCLIGit(t, repo, "commit", "--amend", "-m", "new message")
		}
		result, err := runValidation([]string{"check", "--profile", "Smoke", "--target", "HEAD", "--repo-root", repo, "--json"}, time.Now())
		if err != nil || !result.OK || result.InputDigest != fingerprint.Identity {
			t.Fatalf("HEAD check %s = %#v, %v", message, result, err)
		}
	}

	mustWrite(t, filepath.Join(repo, "tracked.txt"), "dirty\n")
	result, err := runValidation([]string{"check", "--profile", "Smoke", "--target", "HEAD", "--repo-root", repo, "--json"}, time.Now())
	if err == nil || result.OK {
		t.Fatalf("dirty HEAD unexpectedly reusable: %#v, %v", result, err)
	}
	if data := result.Data.(validationCheckData); data.Decision.Code != validationevidence.CodeSubjectNotReusable {
		t.Fatalf("dirty decision = %#v", data.Decision)
	}
}

func TestValidationHeadCheckExplicitlyRepairsOnlyMetadataOnlyTipAlias(t *testing.T) {
	repo := newValidationCLIRepo(t)
	store, fingerprint := putValidationCLIReceipt(t, repo, validationevidence.TargetHead)
	mustValidationCLIGit(t, repo, "commit", "--allow-empty", "-m", "metadata-only replacement one")
	parent, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	mustValidationCLIGit(t, repo, "commit", "--allow-empty", "-m", "metadata-only replacement two")
	head, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	zero := strings.Repeat("0", len(head))
	policy := validationevidence.Policy{SchemaVersion: 1, UnmatchedAction: "allow", Contexts: []validationevidence.PushContext{{
		ID: "stable-main", RemoteRef: "refs/heads/main", RequiredProfile: testengine.ProfileSmoke,
	}}}
	update := gitx.PushUpdate{LocalRef: "refs/heads/topic", LocalOID: head, RemoteRef: "refs/heads/main", RemoteOID: zero}
	if gate := store.GatePush(policy, []gitx.PushUpdate{update}); gate.OK || gate.Updates[0].Code != validationevidence.CodeReceiptMiss {
		t.Fatalf("metadata-only commit unexpectedly had an alias: %#v", gate)
	}

	result, err := runValidation([]string{"check", "--profile", "Smoke", "--target", "HEAD", "--bind-alias", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !result.OK || result.InputDigest != fingerprint.Identity {
		t.Fatalf("alias repair check = %#v, %v", result, err)
	}
	data := result.Data.(validationCheckData)
	if !data.CommitAliasBound {
		t.Fatalf("alias repair was not reported: %#v", data)
	}
	if gate := store.GatePush(policy, []gitx.PushUpdate{update}); !gate.OK || gate.Updates[0].Code != validationevidence.CodeReceiptHit {
		t.Fatalf("repaired metadata-only commit did not pass Context Gate: %#v", gate)
	}
	parentUpdate := update
	parentUpdate.LocalOID = parent
	if gate := store.GatePush(policy, []gitx.PushUpdate{parentUpdate}); gate.OK || gate.Updates[0].Code != validationevidence.CodeReceiptMiss {
		t.Fatalf("alias repair bound a non-tip commit: %#v", gate)
	}
}

func TestValidationHeadCheckDoesNotBindAliasOnMiss(t *testing.T) {
	repo := newValidationCLIRepo(t)
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	result, checkErr := runValidation([]string{"check", "--profile", "Smoke", "--target", "HEAD", "--bind-alias", "--repo-root", repo, "--json"}, time.Now())
	if checkErr == nil || result.OK {
		t.Fatalf("missing Receipt unexpectedly repaired an alias: %#v, %v", result, checkErr)
	}
	head, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	zero := strings.Repeat("0", len(head))
	policy := validationevidence.Policy{SchemaVersion: 1, UnmatchedAction: "allow", Contexts: []validationevidence.PushContext{{
		ID: "stable-main", RemoteRef: "refs/heads/main", RequiredProfile: testengine.ProfileSmoke,
	}}}
	gate := store.GatePush(policy, []gitx.PushUpdate{{LocalRef: "refs/heads/topic", LocalOID: head, RemoteRef: "refs/heads/main", RemoteOID: zero}})
	if gate.OK || gate.Updates[0].Code != validationevidence.CodeReceiptMiss {
		t.Fatalf("Receipt miss manufactured an alias: %#v", gate)
	}
}

func TestTestCommandWiresExplicitEvidenceFlagsAndDefaultsOff(t *testing.T) {
	repo := newValidationCLIRepo(t)
	previous := runTestEngine
	t.Cleanup(func() { runTestEngine = previous })
	configs := make([]testengine.Config, 0, 3)
	runTestEngine = func(_ context.Context, cfg testengine.Config) (testengine.Report, error) {
		configs = append(configs, cfg)
		return testengine.Report{Summary: testengine.Summary{Repo: cfg.Repo, Profile: cfg.Profile, Conclusion: "PASS"}}, nil
	}

	commands := [][]string{
		{"--profile", "Smoke", "--reuse", "auto", "--force", "--allow-dirty", "--repo-root", repo, "--out", "out-force", "--json"},
		{"--profile", "Smoke", "--reuse", "auto", "--verify-reuse", "--repo-root", repo, "--out", "out-audit", "--json"},
		{"--profile", "Smoke", "--repo-root", repo, "--out", "out-default", "--json"},
	}
	for _, args := range commands {
		if result, err := runTest(args, time.Now()); err != nil || !result.OK {
			t.Fatalf("runTest(%v) = %#v, %v", args, result, err)
		}
	}
	if len(configs) != 3 {
		t.Fatalf("captured configs = %d", len(configs))
	}
	if configs[0].Reuse != testengine.ReuseAuto || !configs[0].Force || !configs[0].AllowDirty || configs[0].VerifyReuse {
		t.Fatalf("force config = %#v", configs[0])
	}
	if configs[1].Reuse != testengine.ReuseAuto || configs[1].Force || !configs[1].VerifyReuse {
		t.Fatalf("audit config = %#v", configs[1])
	}
	if configs[2].Reuse != testengine.ReuseOff || configs[2].Force || configs[2].AllowDirty || configs[2].VerifyReuse {
		t.Fatalf("default config = %#v", configs[2])
	}
	for _, cfg := range configs {
		if cfg.CommandCatalogDigest != CatalogSnapshot().Digest() {
			t.Fatalf("command catalog digest was not wired: %#v", cfg)
		}
	}
}

func TestValidationCommandRejectsRemovedAndInvalidForms(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"show"},
		{"check", "--profile", "Smoke", "--target", "AUTO"},
		{"check", "--profile", "Smoke", "--target", "INDEX", "--bind-alias"},
		{"check", "--profile", "Manual", "--target", "HEAD"},
		{"explain", "--profile", "Manual", "--target", "HEAD"},
		{"explain", "--profile", "Smoke", "--target", "AUTO"},
		{"list", "--profile", "Manual"},
	} {
		if _, err := runValidation(args, time.Now()); err == nil {
			t.Fatalf("validation accepted invalid form %v", args)
		}
	}
	for _, args := range [][]string{
		{"--profile", "Smoke", "--reuse", "always"},
		{"--profile", "Smoke", "--force", "--verify-reuse"},
	} {
		if _, err := runTest(args, time.Now()); err == nil {
			t.Fatalf("test accepted invalid evidence flags %v", args)
		}
	}
}

func newValidationCLIRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustValidationCLIGit(t, repo, "init")
	mustValidationCLIGit(t, repo, "config", "user.email", "validation@example.invalid")
	mustValidationCLIGit(t, repo, "config", "user.name", "Validation Test")
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "initial\n")
	mustValidationCLIGit(t, repo, "add", "tracked.txt")
	mustValidationCLIGit(t, repo, "commit", "-m", "initial")
	return repo
}

func putValidationCLIReceipt(t *testing.T, repo string, target validationevidence.Target) (validationevidence.Repository, validationevidence.Fingerprint) {
	t.Helper()
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(target)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := testengine.EvidenceSpec(validationTestConfig(repo, testengine.ProfileSmoke))
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := store.Fingerprint(subject, spec)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(validationevidence.Receipt{
		ValidationIdentity: fingerprint.Identity,
		Fingerprint:        fingerprint,
		Conclusion:         "PASS",
		ResultsDigest:      "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Reusable:           true,
		Scope:              subject.Scope,
	}, validationevidence.ReportBundle{
		ResultsJSON:    []byte("{\"summary\":{},\"results\":[]}\n"),
		SummaryJSON:    []byte("{}\n"),
		ReportMarkdown: []byte("# Validation fixture\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, fingerprint
}

func mustValidationCLIGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
