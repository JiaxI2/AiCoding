package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyStructureManifestMismatchFails(t *testing.T) {
	repo := structureRepo(t, false)
	writeLifecycleRegistry(t, repo, []string{"mismatch-kit"})
	mustWriteLifecycle(t, filepath.Join(repo, "config", "kits", "mismatch-kit.json"), `{"schemaVersion":2,"id":"other-kit","name":"mismatch","version":"0.1.0","kind":["test"],"mode":"go-builtin","paths":{"root":"."},"commands":{"status":{"type":"builtin-check","requiredPaths":[]}}}`)

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if report.OK {
		t.Fatalf("expected mismatch to fail: %#v", report)
	}
	if !containsError(report.Errors, "manifest id mismatch") {
		t.Fatalf("expected manifest id mismatch, got %#v", report.Errors)
	}
}

func TestVerifyStructureMissingManifestFails(t *testing.T) {
	repo := structureRepo(t, false)
	writeLifecycleRegistry(t, repo, []string{"missing-kit"})

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if report.OK {
		t.Fatalf("expected missing manifest to fail: %#v", report)
	}
	if !containsError(report.Errors, "manifest file missing") {
		t.Fatalf("expected missing manifest error, got %#v", report.Errors)
	}
}

func TestVerifyStructureMissingRequiredPathFails(t *testing.T) {
	repo := structureRepo(t, false)
	writeLifecycleRegistry(t, repo, []string{"required-kit"})
	writeLifecycleManifest(t, repo, "required-kit", `"status":{"type":"builtin-check","requiredPaths":["README.md"]}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if report.OK {
		t.Fatalf("expected missing required path to fail: %#v", report)
	}
	if !containsError(report.Errors, "missing required path: README.md") {
		t.Fatalf("expected missing required path error, got %#v", report.Errors)
	}
}

func TestVerifyStructureMissingCodexAssetDirsWarnOnly(t *testing.T) {
	repo := structureRepo(t, false)
	for _, rel := range []string{"CodingKit/examples", "CodingKit/platforms", "CodingKit/tools"} {
		if err := os.RemoveAll(filepath.Join(repo, rel)); err != nil {
			t.Fatal(err)
		}
	}
	writeLifecycleRegistry(t, repo, []string{"minimal-kit"})
	writeLifecycleManifest(t, repo, "minimal-kit", `"status":{"type":"builtin-check","requiredPaths":[]}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if !report.OK {
		t.Fatalf("missing empty asset dirs should warn only: %#v", report)
	}
	if !containsError(report.Warnings, "asset path missing: CodingKit/examples") {
		t.Fatalf("expected asset path warning, got %#v", report.Warnings)
	}
}
func TestVerifyStructureDryRunSkippedSemanticsStayOK(t *testing.T) {
	repo := structureRepo(t, false)
	mustWriteLifecycle(t, filepath.Join(repo, "tools", "specialty", "install-no-dry-run.ps1"), "param()\n")
	writeLifecycleRegistry(t, repo, []string{"unsupported-kit", "missing-action-kit", "no-dry-run-kit", "release-governance-overlay-kit"})
	writeLifecycleManifest(t, repo, "unsupported-kit", `"install":{"type":"unsupported","reason":"not installable"},"update":{"type":"unsupported","reason":"not updateable"}`, "")
	writeLifecycleManifest(t, repo, "missing-action-kit", `"status":{"type":"builtin-check","requiredPaths":[]}`, "")
	writeLifecycleManifest(t, repo, "no-dry-run-kit", `"install":{"type":"specialty-pwsh","path":"tools/specialty/install-no-dry-run.ps1","supportsDryRun":false}`, "")
	writeLifecycleManifest(t, repo, "release-governance-overlay-kit", `"status":{"type":"builtin-check","requiredPaths":[]}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if !report.OK {
		t.Fatalf("expected skipped-only lifecycle plans to stay ok: %#v", report)
	}
	for _, plan := range report.LifecyclePlans {
		if !plan.OK {
			t.Fatalf("expected lifecycle plan %s to be ok: %#v", plan.Action, plan)
		}
	}
}

func TestVerifyStructureWarnsForMissingGeneratedPluginPackage(t *testing.T) {
	repo := structureRepo(t, false)
	mustWriteLifecycle(t, filepath.Join(repo, "tools", "specialty", "install-codex-kit.ps1"), "param()\n")
	writeLifecycleRegistry(t, repo, []string{"aicoding-platform"})
	writeLifecycleManifest(t, repo, "aicoding-platform", `"install":{"type":"builtin-lifecycle","lifecycleAction":"install","supportsDryRun":true}`, `"pluginRoot":"CodingKit/agents/skills/plugins/AiCoding"`)

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if !report.OK {
		t.Fatalf("missing generated plugin package should be warning only: %#v", report)
	}
	if !containsError(report.Warnings, "generated plugin package") && !containsError(report.Warnings, "missing generated plugin package") {
		t.Fatalf("expected generated plugin package warning, got %#v", report.Warnings)
	}
}

func TestVerifyStructureAllValidManifestsOK(t *testing.T) {
	repo := structureRepo(t, true)
	mustWriteLifecycle(t, filepath.Join(repo, "README.md"), "test\n")
	mustWriteLifecycle(t, filepath.Join(repo, "tools", "specialty", "install-agent-patch-kit.ps1"), "param()\n")
	writeLifecycleRegistry(t, repo, []string{"common-control-kit", "agent-patch-kit"})
	writeLifecycleManifest(t, repo, "common-control-kit", `"status":{"type":"builtin-check","requiredPaths":["README.md"]},"install":{"type":"unsupported","reason":"asset kit"},"update":{"type":"unsupported","reason":"asset kit"}`, "")
	writeLifecycleManifest(t, repo, "agent-patch-kit", `"install":{"type":"specialty-pwsh","path":"tools/specialty/install-agent-patch-kit.ps1","supportsDryRun":false},"status":{"type":"external-command","executable":"apatch"}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	report := VerifyStructure(repo, entries)
	if !report.OK {
		t.Fatalf("expected valid manifests to pass: %#v", report)
	}
	if report.Summary.Kits != 2 || report.Summary.EnabledKits != 2 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
}

func TestPluginProjectionCheckWarnsInSmokeAndFailsInBlockingProfiles(t *testing.T) {
	repo := t.TempDir()
	writeLifecycleRegistry(t, repo, []string{"bad-plugin-view"})
	mustWriteLifecycle(t, filepath.Join(repo, "config", "kits", "bad-plugin-view.json"), `{
  "schemaVersion":2,
  "id":"bad-plugin-view",
  "name":"Bad Plugin View",
  "version":"0.1.0",
  "kind":["test"],
  "mode":"go-builtin",
  "description":"",
  "commands":{"status":{"type":"external-command","executable":"bin/aicoding.exe","args":["status","--all","--json"]}}
}`)
	catalog, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	policy := PluginProjectionPolicy{
		Adapter: PluginAdapter{
			Scope:      "kit",
			StateOwner: "kit",
			Entrypoint: "go-static",
			Actions:    []PluginAdapterAction{{Name: "status", Effect: "read"}},
		},
		TypedCommands: []string{"kit", "lifecycle"},
	}

	smoke := PluginProjectionCheck(repo, catalog.Kits(), policy, false)
	if !smoke.OK || len(smoke.Warnings) != 2 || len(smoke.Errors) != 0 {
		t.Fatalf("Smoke projection severity mismatch: %#v", smoke)
	}
	blocking := PluginProjectionCheck(repo, catalog.Kits(), policy, true)
	t.Logf("Smoke warnings: %v", smoke.Warnings)
	t.Logf("blocking errors: %v", blocking.Errors)
	if blocking.OK || len(blocking.Errors) != 2 || len(blocking.Warnings) != 0 {
		t.Fatalf("blocking projection severity mismatch: %#v", blocking)
	}
	if !containsError(blocking.Errors, "manifest description is empty") || !containsError(blocking.Errors, "unknown typed command: status") {
		t.Fatalf("projection issues were not precise: %#v", blocking.Errors)
	}
}

func structureRepo(t *testing.T, withPluginPackage bool) string {
	t.Helper()
	repo := t.TempDir()
	for _, rel := range []string{
		"CodingKit/examples",
		"CodingKit/modules",
		"CodingKit/platforms",
		"CodingKit/tests",
		"CodingKit/tools",
		".agents/plugins",
	} {
		mustWriteLifecycle(t, filepath.Join(repo, rel, ".keep"), "")
	}
	if withPluginPackage {
		mustWriteLifecycle(t, filepath.Join(repo, "CodingKit", "agents", "skills", "plugins", "AiCoding", "skills", "aicoding-embedded", ".keep"), "")
	}
	mustWriteLifecycle(t, filepath.Join(repo, ".agents", "plugins", "marketplace.json"), `{"name":"aicoding-platform","plugins":[{"name":"aicoding","source":{"source":"local","path":"./CodingKit/agents/skills/plugins/AiCoding"}}]}`)
	mustWriteLifecycle(t, filepath.Join(repo, "config", "codex-kit.json"), `{"name":"AiCoding","version":"0.1.0","codingKitRoot":"./CodingKit","agents":{"skillsSubmodule":"./CodingKit/agents/skills","pluginPath":"./CodingKit/agents/skills/plugins/AiCoding","marketplacePath":"./.agents/plugins/marketplace.json"},"assets":{"examples":"./CodingKit/examples","modules":"./CodingKit/modules","platforms":"./CodingKit/platforms","tests":"./CodingKit/tests","tools":"./CodingKit/tools"},"rules":{"buildPluginInSubmodule":false,"pluginInstallUsesMarketplace":true,"hooksAreAuxiliaryConstraints":true}}`)
	return repo
}
