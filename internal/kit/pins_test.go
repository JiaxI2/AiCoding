package kit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func TestPinnedSourceRejectsMutablePurePathAndKeepsOldManifestCompatible(t *testing.T) {
	base := `{"schemaVersion":2,"id":"external-skill","name":"External","version":"1.0.0","kind":["skill"],"mode":"go-builtin","commands":{"status":{"type":"builtin-check"}},"source":%s}`
	for name, source := range map[string]string{
		"branch":    `{"kind":"git","url":"https://example.invalid/repo.git","commit":"main"}`,
		"pure-path": `{"kind":"git","path":"../mutable-skill"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var manifest Manifest
			if err := json.Unmarshal([]byte(fmt.Sprintf(base, source)), &manifest); err == nil {
				t.Fatalf("invalid source was accepted: %s", source)
			}
		})
	}

	var old Manifest
	oldContent := `{"schemaVersion":2,"id":"local-kit","name":"Local","version":"1.0.0","kind":["test"],"mode":"go-builtin","commands":{"status":{"type":"builtin-check"}}}`
	if err := json.Unmarshal([]byte(oldContent), &old); err != nil || old.Source != nil {
		t.Fatalf("old source-less manifest is no longer compatible: manifest=%#v err=%v", old, err)
	}
	t.Log("negative_branch=REJECT negative_pure_path=REJECT old_manifest=COMPATIBLE")
}

func TestPrefetchBadCommitFailsClosed(t *testing.T) {
	external, _ := newPinnedExternalRepository(t)
	consumer := newPinnedConsumerRepository(t)
	source := &PinnedSource{Kind: "git", URL: external, Commit: strings.Repeat("f", 40)}
	status, err := PrefetchPin(context.Background(), consumer, "external-skill", source)
	if err == nil || status.Resolved {
		t.Fatalf("nonexistent commit was not rejected: status=%#v err=%v", status, err)
	}
	if _, statErr := os.Stat(status.CachePath); !os.IsNotExist(statErr) {
		t.Fatalf("failed prefetch published a cache entry: %s err=%v", status.CachePath, statErr)
	}
	t.Logf("negative_bad_sha=FAIL_CLOSED networkCalls=%d error=%v", status.NetworkCalls, err)
}

func TestPinnedLifecycleMaterializesLocallyWithZeroImportNetworkCalls(t *testing.T) {
	external, commit := newPinnedExternalRepository(t)
	consumer := newPinnedConsumerRepository(t)
	manifest := pinnedManifest("external-skill", external, commit)
	writePinnedCatalog(t, consumer, manifest)

	status, err := PrefetchRegisteredKit(context.Background(), consumer, manifest.ID)
	if err != nil || !status.Resolved || status.NetworkCalls != 1 {
		t.Fatalf("prefetch failed: status=%#v err=%v", status, err)
	}
	if _, err := os.Stat(filepath.Join(consumer, "skills", "external", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("external source was vendored into the consumer repository: %v", err)
	}

	originalFetch := runPinFetch
	networkCalls := 0
	runPinFetch = func(objectRepository, locator, commit string) error {
		networkCalls++
		return originalFetch(objectRepository, locator, commit)
	}
	t.Cleanup(func() { runPinFetch = originalFetch })

	catalog, err := LoadCatalogSnapshot(consumer)
	if err != nil {
		t.Fatal(err)
	}
	plan := PlanCatalogLifecycle(consumer, catalog.Kits(), LifecycleOptions{Action: "install", Mode: "selected", DryRun: true})
	if !plan.OK || plan.RequiredAction != "" {
		t.Fatalf("prefetched pin did not satisfy the bounded path check: %#v", plan)
	}
	action := RunCatalogAction(consumer, catalog.Kits(), ActionOptions{Action: "install", Mode: "selected"})
	if !action.OK || networkCalls != 0 || action.RequiredAction != "" {
		t.Fatalf("local import failed or fetched network content: action=%#v networkCalls=%d", action, networkCalls)
	}
	materialized := filepath.Join(consumer, ".aicoding", "state", "kits", manifest.ID, "source", "skills", "external", "SKILL.md")
	content, err := os.ReadFile(materialized)
	if err != nil || !strings.Contains(string(content), "Pinned External Skill") {
		t.Fatalf("materialized skill is invalid: %s err=%v", content, err)
	}
	state, err := readInstallState(statePath(consumer, manifest, manifest.ID))
	if err != nil || state.SourceIdentity != status.Identity || state.MaterializedPath == "" {
		t.Fatalf("install state did not bind the pin: state=%#v err=%v", state, err)
	}
	t.Logf("positive_prefetch=RESOLVED identity=%s import_network_calls=%d materialized=%s", status.Identity, networkCalls, materialized)
}

func TestPinnedCacheKeepsLooseObjectsReadableAfterLongFinalRename(t *testing.T) {
	base := filepath.Join(t.TempDir(), "long-path")
	external, commit := newPinnedExternalRepositoryAt(t, filepath.Join(base, "external"))
	consumer := newPinnedConsumerRepositoryAt(t, filepath.Join(base, "consumer"))
	source := &PinnedSource{Kind: "git", URL: external, Commit: commit}

	status, err := PrefetchPin(context.Background(), consumer, "external-skill", source)
	if err != nil || !status.Resolved {
		t.Fatalf("long final pin path was not readable: cachePath=%s status=%#v err=%v", status.CachePath, status, err)
	}
	objectRepository := filepath.Join(status.CachePath, pinObjectDir)
	configured := strings.TrimSpace(pinGit(t, objectRepository, "config", "--bool", "core.longpaths"))
	if configured != "true" {
		t.Fatalf("pin object cache core.longpaths = %q, want true", configured)
	}
	materialized, err := MaterializePinnedSource(context.Background(), consumer, "external-skill", source)
	if err != nil || materialized.NetworkCalls != 0 {
		t.Fatalf("long final pin path was not materialized locally: result=%#v err=%v", materialized, err)
	}
	if _, err := os.Stat(filepath.Join(materialized.MaterializedPath, "skills", "external", "SKILL.md")); err != nil {
		t.Fatalf("long-path materialization is missing the pinned Skill: %v", err)
	}
	objectPath := filepath.Join(objectRepository, "objects", commit[:2], commit[2:])
	t.Logf("long_path_cache=%s objectPathLength=%d", status.CachePath, len(objectPath))
}

func TestPinnedLifecycleMissingPrefetchReturnsRequiredActionWithoutFetch(t *testing.T) {
	external, commit := newPinnedExternalRepository(t)
	consumer := newPinnedConsumerRepository(t)
	manifest := pinnedManifest("external-skill", external, commit)
	writePinnedCatalog(t, consumer, manifest)

	originalFetch := runPinFetch
	networkCalls := 0
	runPinFetch = func(objectRepository, locator, commit string) error {
		networkCalls++
		return fmt.Errorf("unexpected network fetch")
	}
	t.Cleanup(func() { runPinFetch = originalFetch })

	catalog, err := LoadCatalogSnapshot(consumer)
	if err != nil {
		t.Fatal(err)
	}
	action := RunCatalogAction(consumer, catalog.Kits(), ActionOptions{Action: "install", Mode: "selected"})
	wantAction := "aicoding kit prefetch --id external-skill --json"
	if action.OK || action.Category != "evidence-missing" || action.RequiredAction != wantAction || networkCalls != 0 {
		t.Fatalf("missing prefetch did not fail closed: action=%#v networkCalls=%d", action, networkCalls)
	}
	t.Logf("negative_missing_prefetch=EVIDENCE_MISSING requiredAction=%q import_network_calls=%d", action.RequiredAction, networkCalls)
}

func TestContentDigestPinUsesPreseededCacheWithoutNetwork(t *testing.T) {
	consumer := newPinnedConsumerRepository(t)
	seed := t.TempDir()
	writeRegistryTestFile(t, filepath.Join(seed, "notes", "knowledge.md"), "immutable knowledge\n")
	digest, err := digestDirectory(seed)
	if err != nil {
		t.Fatal(err)
	}
	source := &PinnedSource{Kind: "content", Digest: digest}
	root, err := PinCacheRoot(consumer)
	if err != nil {
		t.Fatal(err)
	}
	cacheContent := filepath.Join(root, strings.TrimPrefix(digest, "sha256:"), pinContentDir)
	writeRegistryTestFile(t, filepath.Join(cacheContent, "notes", "knowledge.md"), "immutable knowledge\n")
	status, err := PrefetchPin(context.Background(), consumer, "knowledge", source)
	if err != nil || !status.Resolved || status.NetworkCalls != 0 {
		t.Fatalf("preseeded content pin did not resolve locally: status=%#v err=%v", status, err)
	}
	materialized, err := MaterializePinnedSource(context.Background(), consumer, "knowledge", source)
	if err != nil || materialized.NetworkCalls != 0 || materialized.ContentIdentity != digest {
		t.Fatalf("content pin materialization failed: result=%#v err=%v", materialized, err)
	}
	if _, err := os.Stat(filepath.Join(materialized.MaterializedPath, "notes", "knowledge.md")); err != nil {
		t.Fatalf("content pin file was not materialized: %v", err)
	}
	t.Logf("content_digest=%s prefetch_network_calls=%d import_network_calls=%d", digest, status.NetworkCalls, materialized.NetworkCalls)
}

func TestPinnedSourceChangeInvalidatesOldValidationReceipt(t *testing.T) {
	repo := newPinnedConsumerRepository(t)
	manifest := pinnedManifest("external-skill", "https://example.invalid/upstream.git", strings.Repeat("a", 40))
	writePinnedCatalog(t, repo, manifest)
	pinGit(t, repo, "add", ".")
	pinGit(t, repo, "commit", "-m", "first pin")
	firstCatalog, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	firstSubject, err := store.Capture(validationevidence.TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	spec := validationevidence.FingerprintSpec{
		Profile: "smoke", ValidationPlanDigest: pinTestDigest("plan"),
		EngineSemanticDigest: pinTestDigest("engine"), OptionsDigest: pinTestDigest("options"),
	}
	firstFingerprint, err := store.Fingerprint(firstSubject, spec)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.Put(validationevidence.Receipt{
		ValidationIdentity: firstFingerprint.Identity, Fingerprint: firstFingerprint,
		Conclusion: "PASS", ResultsDigest: pinTestDigest("results"), Reusable: true,
		Scope: validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}, validationevidence.ReportBundle{
		ResultsJSON: []byte(`{"results":[]}`), SummaryJSON: []byte(`{"conclusion":"PASS"}`), ReportMarkdown: []byte("# PASS\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	manifest.Source.Commit = strings.Repeat("b", 40)
	writePinnedManifest(t, repo, manifest)
	pinGit(t, repo, "add", ".")
	pinGit(t, repo, "commit", "-m", "move pin")
	secondCatalog, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	secondSubject, err := store.Capture(validationevidence.TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	secondFingerprint, err := store.Fingerprint(secondSubject, spec)
	if err != nil {
		t.Fatal(err)
	}
	decision := store.Check(secondSubject, secondFingerprint)
	if firstCatalog.Digest() == secondCatalog.Digest() || firstFingerprint.Identity == secondFingerprint.Identity || decision.Hit {
		t.Fatalf("pin change reused stale evidence: catalog=%s/%s identity=%s/%s decision=%#v", firstCatalog.Digest(), secondCatalog.Digest(), firstFingerprint.Identity, secondFingerprint.Identity, decision)
	}
	t.Logf("negative_pin_change=RECEIPT_MISS oldReceipt=%s oldIdentity=%s newIdentity=%s", stored.ReceiptID, firstFingerprint.Identity, secondFingerprint.Identity)
}

func newPinnedExternalRepository(t *testing.T) (string, string) {
	t.Helper()
	return newPinnedExternalRepositoryAt(t, t.TempDir())
}

func newPinnedExternalRepositoryAt(t *testing.T, repo string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	pinGit(t, repo, "init")
	configurePinnedGitIdentity(t, repo)
	content := "---\nname: pinned-external\ndescription: Pinned External Skill\n---\n\n# Pinned External Skill\n\nUse immutable content.\n"
	writeRegistryTestFile(t, filepath.Join(repo, "skills", "external", "SKILL.md"), content)
	pinGit(t, repo, "add", ".")
	pinGit(t, repo, "commit", "-m", "external skill")
	return repo, strings.TrimSpace(pinGit(t, repo, "rev-parse", "HEAD"))
}

func newPinnedConsumerRepository(t *testing.T) string {
	t.Helper()
	return newPinnedConsumerRepositoryAt(t, t.TempDir())
}

func newPinnedConsumerRepositoryAt(t *testing.T, repo string) string {
	t.Helper()
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	pinGit(t, repo, "init")
	configurePinnedGitIdentity(t, repo)
	return repo
}

func pinnedManifest(id, locator, commit string) Manifest {
	skill, _ := json.Marshal(map[string]interface{}{
		"id": "pinned-external", "path": "skills/external/SKILL.md", "role": "router",
		"description": "Use the immutable external skill.", "tags": []string{"pinned"},
	})
	return Manifest{
		SchemaVersion: 2, ID: id, Name: "Pinned External", Version: "1.0.0", Kind: []string{"skill"},
		Mode: "go-builtin", Description: "Makes one immutable external skill available without vendoring.",
		Source: &PinnedSource{Kind: "git", URL: locator, Commit: commit},
		Commands: map[string]CommandDef{
			"install":   {Type: "builtin-lifecycle", SupportsDryRun: true, RequiredPaths: []string{"skills/external/SKILL.md"}},
			"status":    {Type: "builtin-check", RequiredPaths: []string{"skills/external/SKILL.md"}},
			"update":    {Type: "builtin-lifecycle", SupportsDryRun: true, RequiredPaths: []string{"skills/external/SKILL.md"}},
			"uninstall": {Type: "builtin-lifecycle", SupportsDryRun: true},
		},
		Skills: map[string]json.RawMessage{"umbrella": skill},
		Trust:  map[string]interface{}{"thirdParty": true, "updatePolicy": "pinned", "level": "third-party"},
	}
}

func writePinnedCatalog(t *testing.T, repo string, manifest Manifest) {
	t.Helper()
	writeRegistryTestFile(t, filepath.Join(repo, "config", "kit-registry.json"), `{"schemaVersion":1,"name":"test","defaultMode":"repo-scoped","kits":[{"id":"`+manifest.ID+`","enabled":true,"order":10,"manifest":"config/kits/`+manifest.ID+`.json"}]}`)
	writePinnedManifest(t, repo, manifest)
}

func writePinnedManifest(t *testing.T, repo string, manifest Manifest) {
	t.Helper()
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeRegistryTestFile(t, filepath.Join(repo, "config", "kits", manifest.ID+".json"), string(content)+"\n")
}

func configurePinnedGitIdentity(t *testing.T, repo string) {
	t.Helper()
	pinGit(t, repo, "config", "user.email", "pins@example.invalid")
	pinGit(t, repo, "config", "user.name", "Pinned Test")
}

func pinGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func pinTestDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("sha256:%x", sum)
}
