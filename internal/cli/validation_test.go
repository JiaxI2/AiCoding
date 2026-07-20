package cli

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

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
		{"check", "--profile", "Manual", "--target", "HEAD"},
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
