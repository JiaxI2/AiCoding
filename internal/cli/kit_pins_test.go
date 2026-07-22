package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func TestKitRegisterStartsBackgroundPrefetchAndListStaysFast(t *testing.T) {
	repo := newPinnedCLIRepo(t)
	writePinnedCLIManifest(t, repo, "https://example.invalid/external.git", "0123456789abcdef0123456789abcdef01234567")
	originalStarter := startKitPrefetchJob
	startedID := ""
	startKitPrefetchJob = func(_ string, id string) (kitPrefetchJob, error) {
		startedID = id
		return kitPrefetchJob{Status: "started", PID: 42, LogPath: "prefetch.json", Command: "aicoding kit prefetch --id " + id + " --json"}, nil
	}
	t.Cleanup(func() { startKitPrefetchJob = originalStarter })

	registered, err := runKit([]string{"register", "--manifest", "config/kits/external-skill.json", "--prefetch", "--repo-root", repo, "--json"}, time.Now())
	data, ok := registered.Data.(kitRegisterResult)
	if err != nil || !registered.OK || !ok || startedID != "external-skill" || data.Prefetch == nil || data.Prefetch.Status != "started" {
		t.Fatalf("register did not schedule background prefetch: result=%#v data=%#v err=%v", registered, data, err)
	}
	if registered.ElapsedMS >= 300 {
		t.Fatalf("kit register exceeded 300ms: %dms", registered.ElapsedMS)
	}
	maxElapsed := int64(0)
	for index := 0; index < 5; index++ {
		listed, err := runKit([]string{"list", "--repo-root", repo, "--json"}, time.Now())
		if err != nil || !listed.OK {
			t.Fatalf("kit list failed: result=%#v err=%v", listed, err)
		}
		if listed.ElapsedMS > maxElapsed {
			maxElapsed = listed.ElapsedMS
		}
		views, ok := listed.Data.([]kit.View)
		if !ok || len(views) != 1 || views[0].ID != "external-skill" || views[0].SourceIdentity == "" {
			t.Fatalf("registered pinned Kit is not visible: %#v", listed.Data)
		}
	}
	if maxElapsed >= 300 {
		t.Fatalf("kit list exceeded 300ms: %dms", maxElapsed)
	}
	t.Logf("background_prefetch=started registration_ms=%d kit_list_max_ms=%d", registered.ElapsedMS, maxElapsed)
}

func TestLifecycleCLIReportsEvidenceMissingWithoutImplicitFetch(t *testing.T) {
	external, commit := newPinnedCLIExternalRepo(t)
	repo := newPinnedCLIRepo(t)
	writePinnedCLIManifest(t, repo, external, commit)
	registered, err := runKit([]string{"register", "--manifest", "config/kits/external-skill.json", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !registered.OK {
		t.Fatalf("register failed: result=%#v err=%v", registered, err)
	}

	result, err := runLifecycle([]string{"install", "--scope", "kit", "--kit", "external-skill", "--repo-root", repo, "--json"}, time.Now())
	want := "aicoding kit prefetch --id external-skill --json"
	if err == nil || result.OK || result.Category != report.CategoryEvidenceMissing || result.NextAction != want {
		t.Fatalf("missing pin did not produce executable evidence decision: result=%#v err=%v", result, err)
	}
	cacheRoot, cacheErr := kit.PinCacheRoot(repo)
	entries, readErr := os.ReadDir(cacheRoot)
	if cacheErr != nil || (readErr == nil && len(entries) != 0) {
		t.Fatalf("import unexpectedly populated the pin cache: root=%s cacheErr=%v readErr=%v entries=%v", cacheRoot, cacheErr, readErr, entries)
	}
	t.Logf("category=%s requiredAction=%q implicit_fetch=0", result.Category, result.NextAction)
}

func TestKitPrefetchThenLifecycleInstallUsesLocalPin(t *testing.T) {
	external, commit := newPinnedCLIExternalRepo(t)
	repo := newPinnedCLIRepo(t)
	writePinnedCLIManifest(t, repo, external, commit)
	if registered, err := runKit([]string{"register", "--manifest", "config/kits/external-skill.json", "--repo-root", repo, "--json"}, time.Now()); err != nil || !registered.OK {
		t.Fatalf("register failed: result=%#v err=%v", registered, err)
	}
	prefetched, err := runKit([]string{"prefetch", "--id", "external-skill", "--repo-root", repo, "--json"}, time.Now())
	status, ok := prefetched.Data.(kit.PinStatus)
	if err != nil || !prefetched.OK || !ok || !status.Resolved || status.NetworkCalls != 1 {
		t.Fatalf("prefetch failed: result=%#v status=%#v err=%v", prefetched, status, err)
	}
	started := time.Now()
	installed, err := runLifecycle([]string{"install", "--scope", "kit", "--kit", "external-skill", "--repo-root", repo, "--json"}, started)
	if err != nil || !installed.OK || installed.Category != "" && installed.Category != report.CategoryNone {
		t.Fatalf("local lifecycle install failed: result=%#v err=%v", installed, err)
	}
	elapsed := time.Since(started)
	materialized := filepath.Join(repo, ".aicoding", "state", "kits", "external-skill", "source", "skills", "external", "SKILL.md")
	if _, err := os.Stat(materialized); err != nil || elapsed >= 10*time.Second {
		t.Fatalf("local materialization missed performance boundary: path=%s elapsed=%s err=%v", materialized, elapsed, err)
	}
	t.Logf("prefetch_network_calls=%d import_elapsed_ms=%d import_network_calls=0", status.NetworkCalls, elapsed.Milliseconds())
}

func newPinnedCLIRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitWorkTest(t, repo, "init")
	gitWorkTest(t, repo, "config", "user.email", "pins@example.invalid")
	gitWorkTest(t, repo, "config", "user.name", "Pinned CLI")
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.invalid/pinned\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(repo, "config", "kit-registry.json"), `{"schemaVersion":1,"name":"test","defaultMode":"repo-scoped","kits":[]}`)
	mustWrite(t, filepath.Join(repo, "config", "dependency-governance.json"), `{"schemaVersion":1,"name":"test","direction":"higher-rank-may-depend-on-equal-or-lower-rank","kitRegistry":{"path":"config/kit-registry.json","bindings":[]}}`)
	return repo
}

func newPinnedCLIExternalRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	gitWorkTest(t, repo, "init")
	gitWorkTest(t, repo, "config", "user.email", "pins@example.invalid")
	gitWorkTest(t, repo, "config", "user.name", "Pinned CLI")
	mustWrite(t, filepath.Join(repo, "skills", "external", "SKILL.md"), "---\nname: pinned-external\ndescription: Pinned CLI Skill\n---\n\n# Pinned CLI Skill\n")
	gitWorkTest(t, repo, "add", ".")
	gitWorkTest(t, repo, "commit", "-m", "pinned external")
	return repo, strings.TrimSpace(gitWorkTest(t, repo, "rev-parse", "HEAD"))
}

func writePinnedCLIManifest(t *testing.T, repo, locator, commit string) {
	t.Helper()
	skill, err := json.Marshal(map[string]interface{}{
		"id": "pinned-external", "path": "skills/external/SKILL.md", "role": "router",
		"description": "Use the pinned external skill.", "tags": []string{"pinned"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := kit.Manifest{
		SchemaVersion: 2, ID: "external-skill", Name: "External Skill", Version: "1.0.0",
		Kind: []string{"skill"}, Mode: "go-builtin", Description: "Makes a pinned external skill available without vendoring.",
		Source: &kit.PinnedSource{Kind: "git", URL: locator, Commit: commit},
		Commands: map[string]kit.CommandDef{
			"install":   {Type: "builtin-lifecycle", SupportsDryRun: true, RequiredPaths: []string{"skills/external/SKILL.md"}},
			"status":    {Type: "builtin-check", RequiredPaths: []string{"skills/external/SKILL.md"}},
			"uninstall": {Type: "builtin-lifecycle", SupportsDryRun: true},
			"update":    {Type: "builtin-lifecycle", SupportsDryRun: true, RequiredPaths: []string{"skills/external/SKILL.md"}},
		},
		Skills: map[string]json.RawMessage{"umbrella": skill},
		Trust:  map[string]interface{}{"thirdParty": true, "updatePolicy": "pinned", "level": "third-party"},
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, "config", "kits", "external-skill.json"), string(content)+"\n")
}
