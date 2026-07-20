package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func TestPrePushHookChecksActualLocalObject(t *testing.T) {
	repo := newValidationCLIRepo(t)
	writeHookValidationPolicy(t, repo, "smoke")
	mustValidationCLIGit(t, repo, "add", "config/validation-policy.json")
	mustValidationCLIGit(t, repo, "commit", "-m", "policy")
	store, fingerprint := putValidationCLIReceipt(t, repo, validationevidence.TargetHead)
	head, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(validationevidence.TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	decision := store.Check(subject, fingerprint)
	if !decision.Hit || decision.Receipt == nil {
		t.Fatalf("fixture Receipt miss: %#v", decision)
	}
	if err := store.BindCommit(head, *decision.Receipt); err != nil {
		t.Fatal(err)
	}
	zero := strings.Repeat("0", len(head))

	previousInput := hookPrePushInput
	t.Cleanup(func() { hookPrePushInput = previousInput })
	hookPrePushInput = strings.NewReader("refs/heads/topic " + head + " refs/heads/main " + zero + "\n")
	result, err := runHook([]string{"pre-push", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !result.OK {
		t.Fatalf("pre-push hit = %#v, %v", result, err)
	}
	data := result.Data.(prePushData)
	if len(data.Gate.Updates) != 1 || data.Gate.Updates[0].LocalOID != head || data.Gate.Updates[0].Code != validationevidence.CodeReceiptHit {
		t.Fatalf("pre-push did not gate the supplied local_oid: %#v", data)
	}

	mustWrite(t, filepath.Join(repo, "tracked.txt"), "second\n")
	mustValidationCLIGit(t, repo, "commit", "-am", "second")
	newHead, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	hookPrePushInput = strings.NewReader("refs/heads/main " + newHead + " refs/heads/main " + head + "\n")
	result, err = runHook([]string{"pre-push", "--repo-root", repo}, time.Now())
	if err == nil || !report.IsValidationError(err) || result.OK {
		t.Fatalf("pre-push missing Receipt = %#v, %v", result, err)
	}
}

func TestPostCommitRefreshBindsStagedReceiptToNewCommit(t *testing.T) {
	repo := newValidationCLIRepo(t)
	writeHookValidationPolicy(t, repo, "smoke")
	mustValidationCLIGit(t, repo, "add", "config/validation-policy.json")
	mustValidationCLIGit(t, repo, "commit", "-m", "policy")
	mustWrite(t, filepath.Join(repo, "tracked.txt"), "staged\n")
	mustValidationCLIGit(t, repo, "add", "tracked.txt")
	store, fingerprint := putValidationCLIReceipt(t, repo, validationevidence.TargetIndex)
	mustValidationCLIGit(t, repo, "commit", "-m", "validated staged tree")
	head, err := gitx.HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}

	refresh := refreshHeadValidationAliases(repo)
	if len(refresh.Errors) != 0 || refresh.Bound != 1 || refresh.Missed != 0 || refresh.CommitOID != head || refresh.TreeOID != fingerprint.SubjectTreeOID {
		t.Fatalf("alias refresh = %#v", refresh)
	}
	policy, err := validationevidence.LoadPolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	zero := strings.Repeat("0", len(head))
	gate := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "refs/heads/main", LocalOID: head, RemoteRef: "refs/heads/main", RemoteOID: zero,
	}})
	if !gate.OK || gate.Updates[0].Code != validationevidence.CodeReceiptHit {
		t.Fatalf("post-commit alias did not satisfy Context Gate: %#v", gate)
	}
}

func TestRepositoryHooksHaveNoBuildOrTestFallback(t *testing.T) {
	for _, name := range []string{"pre-commit", "commit-msg", "post-commit", "pre-push"} {
		path := filepath.Join("..", "..", ".githooks", name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content := string(raw)
		for _, forbidden := range []string{"go run", "go build", "aicoding test"} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s hook contains forbidden action %q", name, forbidden)
			}
		}
	}
	path := filepath.Join("..", "..", ".githooks", "pre-push")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, forbidden := range []string{"stash", "reset --", "checkout", "git push"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("pre-push hook contains forbidden action %q", forbidden)
		}
	}
	if !strings.Contains(content, "hook pre-push") {
		t.Fatal("pre-push hook does not route to the Go Context Gate")
	}
}

func writeHookValidationPolicy(t *testing.T, repo, profile string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "config", "validation-policy.json"), `{
  "schemaVersion": 1,
  "unmatchedAction": "allow",
  "contexts": [{
    "id": "stable-main",
    "remoteRef": "refs/heads/main",
    "requiredProfile": "`+profile+`",
    "requireFastForward": true,
    "allowDelete": false
  }]
}`)
}
