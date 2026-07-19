package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAllPlanUsesStaticAdaptersWithoutWritingCodexConfig(t *testing.T) {
	repo := t.TempDir()
	writeLifecycleFixture(t, repo)
	configPath := filepath.Join(repo, "config.toml")
	const configText = "[mcp_servers.existing]\nurl = \"https://example.com/mcp\"\n"
	mustWrite(t, configPath, configText)

	executed := 0
	fakeExecutor := func(_ context.Context, dir, _ string, args []string) ([]byte, []byte, error) {
		executed++
		if dir != repo {
			t.Fatalf("runtime adapter dir = %q, want %q", dir, repo)
		}
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "set-codex-skill-profile.ps1") || !strings.Contains(joined, "-DryRun") {
			t.Fatalf("unexpected runtime plan command: %s", joined)
		}
		return []byte(`{"profile":"runtime","dryRun":true,"warnings":[]}`), nil, nil
	}

	result := run(context.Background(), repo, normalizeOptions(Options{
		Action:         "install",
		Scope:          ScopeAll,
		All:            true,
		CodexConfig:    configPath,
		DryRun:         true,
		RuntimeProfile: "runtime",
	}), fakeExecutor)
	if !result.OK || result.Mode != "plan" || len(result.Adapters) != 4 {
		t.Fatalf("unexpected lifecycle plan: %#v", result)
	}
	if !strings.HasPrefix(result.CatalogDigest, "sha256:") || !strings.HasPrefix(result.PlanDigest, "sha256:") {
		t.Fatalf("lifecycle evidence digests are missing: %#v", result)
	}
	if executed != 1 {
		t.Fatalf("runtime adapter executions = %d, want 1", executed)
	}
	for _, adapter := range result.Adapters {
		if !adapter.OK || adapter.Status != "planned" {
			t.Fatalf("unexpected adapter result: %#v", adapter)
		}
		if !strings.HasPrefix(adapter.InputDigest, "sha256:") {
			t.Fatalf("adapter input digest is missing: %#v", adapter)
		}
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != configText {
		t.Fatalf("dry-run changed Codex config:\n%s", after)
	}
	if _, err := os.Stat(filepath.Join(repo, "asset", ".venv")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created MCP venv: %v", err)
	}
}

func TestAdapterCatalogIsInspectableStableAndDetached(t *testing.T) {
	first, err := LoadAdapterCatalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadAdapterCatalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() || !strings.HasPrefix(first.Digest(), "sha256:") {
		t.Fatalf("adapter catalog digest is unstable: %q != %q", first.Digest(), second.Digest())
	}
	descriptors := first.Descriptors()
	if len(descriptors) != 4 {
		t.Fatalf("adapter descriptors = %d, want 4", len(descriptors))
	}
	descriptors[0].Actions[0].Name = "mutated"
	if first.Descriptors()[0].Actions[0].Name == "mutated" {
		t.Fatal("adapter catalog was mutable through returned descriptors")
	}
}

func TestLifecycleExecutionPlanDigestTracksOnlyStableIntent(t *testing.T) {
	catalog, err := LoadAdapterCatalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := selectedAdapterDefinitions(ScopeKit, "install")
	if err != nil {
		t.Fatal(err)
	}
	base := normalizeOptions(Options{Action: "install", Scope: ScopeKit, KitID: "sample", DryRun: true})
	first, err := buildExecutionPlan(t.TempDir(), base, catalog.Digest(), definitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	base.VerifyProfile = "Release"
	second, err := buildExecutionPlan(t.TempDir(), base, catalog.Digest(), definitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstDigest, _ := first.Digest()
	secondDigest, _ := second.Digest()
	if firstDigest != secondDigest {
		t.Fatalf("irrelevant verify profile changed install plan: %q != %q", firstDigest, secondDigest)
	}
	base.KitID = "other"
	changed, err := buildExecutionPlan(t.TempDir(), base, catalog.Digest(), definitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	changedDigest, _ := changed.Digest()
	if firstDigest == changedDigest {
		t.Fatal("selection change did not change lifecycle plan digest")
	}
}

func TestRuntimeSkillVerifyUsesStrictAudit(t *testing.T) {
	repo := t.TempDir()
	var command string
	fakeExecutor := func(_ context.Context, _ string, _ string, args []string) ([]byte, []byte, error) {
		command = strings.Join(args, " ")
		return []byte(`{"ok":true}`), nil, nil
	}
	result := run(context.Background(), repo, normalizeOptions(Options{
		Action:         "verify",
		Scope:          ScopeRuntimeSkill,
		RuntimeProfile: "full",
		RuntimeSkill:   "visio-diagram",
	}), fakeExecutor)
	if !result.OK || len(result.Adapters) != 1 {
		t.Fatalf("unexpected runtime verification: %#v", result)
	}
	for _, expected := range []string{
		"audit-runtime-skills.ps1",
		"-ExpectedProfile full",
		"-Skill visio-diagram",
		"-Strict",
		"-Json",
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("runtime verification command %q is missing %q", command, expected)
		}
	}
}

func TestRuntimeSkillAuditDriftFailsUnifiedReport(t *testing.T) {
	fakeExecutor := func(context.Context, string, string, []string) ([]byte, []byte, error) {
		return []byte(`{"ok":false}`), nil, nil
	}
	result := run(context.Background(), t.TempDir(), normalizeOptions(Options{
		Action: "doctor",
		Scope:  ScopeRuntimeSkill,
	}), fakeExecutor)
	if result.OK || result.Summary.Failed != 1 || len(result.Errors) == 0 {
		t.Fatalf("expected runtime drift failure: %#v", result)
	}
}

func TestRuntimeSkillApplyRequiresResolvableSourceRepository(t *testing.T) {
	executed := false
	fakeExecutor := func(context.Context, string, string, []string) ([]byte, []byte, error) {
		executed = true
		return []byte(`{"profile":"runtime","dryRun":false}`), nil, nil
	}
	result := run(context.Background(), t.TempDir(), normalizeOptions(Options{
		Action:         "install",
		Scope:          ScopeRuntimeSkill,
		RuntimeProfile: "runtime",
	}), fakeExecutor)
	if result.OK || executed || len(result.Errors) == 0 {
		t.Fatalf("unresolved runtime source must block apply: %#v", result)
	}
}

func TestResolveRuntimeSourceRepositoryUsesConfiguredEnvironment(t *testing.T) {
	repo := t.TempDir()
	source := filepath.Join(t.TempDir(), "Codex-Skills")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, "config", "codex-kit.json"), `{
  "skillRuntime":{
    "sourceRepositoryEnv":"TEST_AICODING_SKILL_SOURCE",
    "defaultSourceRepository":"../missing"
  }
}`)
	t.Setenv("TEST_AICODING_SKILL_SOURCE", source)
	resolved, exists := resolveRuntimeSourceRepository(repo)
	if !exists || resolved != filepath.Clean(source) {
		t.Fatalf("resolved source = %q, %t; want %q, true", resolved, exists, source)
	}
}

func TestKitLifecycleRollbackRemainsAvailableThroughAdapter(t *testing.T) {
	repo := t.TempDir()
	writeLifecycleFixture(t, repo)
	apply := Run(context.Background(), repo, Options{
		Action: "install",
		Scope:  ScopeKit,
		All:    true,
	})
	if !apply.OK {
		t.Fatalf("kit apply failed: %#v", apply)
	}
	statePath := filepath.Join(repo, ".aicoding", "state", "kits", "sample-kit", "install-state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected install state: %v", err)
	}

	rollback := Run(context.Background(), repo, Options{
		Action: "rollback",
		Scope:  ScopeKit,
	})
	if !rollback.OK {
		t.Fatalf("kit rollback failed: %#v", rollback)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("rollback did not remove newly created state: %v", err)
	}
}

func writeLifecycleFixture(t *testing.T, repo string) {
	t.Helper()
	sourceRepository := filepath.Join(filepath.Dir(repo), "Codex-Skills")
	if err := os.MkdirAll(sourceRepository, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, "README.md"), "fixture\n")
	mustWrite(t, filepath.Join(repo, "config", "codex-kit.json"), `{
  "skillRuntime":{
    "sourceRepositoryEnv":"TEST_AICODING_SKILL_SOURCE",
    "defaultSourceRepository":"../Codex-Skills"
  }
}`)
	mustWrite(t, filepath.Join(repo, "config", "kit-registry.json"), `{
  "schemaVersion":1,
  "name":"test",
  "defaultMode":"repo-scoped",
  "kits":[{"id":"sample-kit","enabled":true,"order":10,"manifest":"config/kits/sample-kit.json"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "kits", "sample-kit.json"), `{
  "schemaVersion":2,
  "id":"sample-kit",
  "name":"Sample Kit",
  "version":"0.1.0",
  "kind":["test"],
  "mode":"go-builtin",
  "paths":{"root":"."},
  "state":{"installState":".aicoding/state/kits/sample-kit/install-state.json"},
  "commands":{
    "install":{"type":"builtin-lifecycle","supportsDryRun":true,"requiredPaths":["README.md"]},
    "update":{"type":"builtin-lifecycle","supportsDryRun":true,"requiredPaths":["README.md"]},
    "uninstall":{"type":"builtin-lifecycle","supportsDryRun":true,"requiredPaths":["README.md"]},
    "status":{"type":"builtin-check","requiredPaths":["README.md"]}
  }
}`)
	mustWrite(t, filepath.Join(repo, "config", "mcp-registry.json"), `{
  "schemaVersion":1,
  "name":"test",
  "components":[{"id":"sample-mcp","enabled":true,"order":10,"manifest":"config/mcp/components/sample-mcp.json"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "mcp", "components", "sample-mcp.json"), `{
  "schemaVersion":1,
  "id":"sample-mcp",
  "name":"Sample MCP",
  "version":"0.1.0",
  "transport":"stdio",
  "runtime":{
    "kind":"python-venv",
    "root":"asset",
    "requirements":"requirements.txt",
    "minimumPython":"3.10",
    "pythonEnvVar":"SAMPLE_MCP_PYTHON",
    "module":"sample_mcp",
    "packageInstall":["-e","."],
    "serverArgs":["-m","sample_mcp","server"],
    "env":{}
  },
  "codex":{"serverName":"sample-mcp","startupTimeoutSec":30,"toolTimeoutSec":120},
  "doctor":{"args":["-m","sample_mcp","doctor","--json"]},
  "verify":{
    "Smoke":[["-m","sample_mcp","verify","--json"]],
    "Full":[["-m","sample_mcp","verify","--json"]],
    "Release":[["-m","sample_mcp","verify","--json"]]
  }
}`)
	mustWrite(t, filepath.Join(repo, "asset", "requirements.txt"), "\n")
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
