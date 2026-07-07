package kit

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPlanLifecycleSkipsUnsupportedMissingAndNoDryRun(t *testing.T) {
	repo := t.TempDir()
	writeLifecycleRegistry(t, repo, []string{"unsupported-kit", "missing-action-kit", "no-dry-run-kit"})
	writeLifecycleManifest(t, repo, "unsupported-kit", `"install":{"type":"unsupported","reason":"not installable"}`, "")
	writeLifecycleManifest(t, repo, "missing-action-kit", `"status":{"type":"builtin-check","requiredPaths":[]}`, "")
	writeLifecycleManifest(t, repo, "no-dry-run-kit", `"install":{"type":"powershell-script","path":"scripts/install-no-dry-run.ps1","supportsDryRun":false}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	plan := PlanLifecycle(repo, entries, LifecycleOptions{Action: "install", Mode: "all", DryRun: true})
	if !plan.OK {
		t.Fatalf("expected skipped-only plan to be ok: %#v", plan)
	}
	if plan.Summary.Total != 3 || plan.Summary.Skipped != 3 || plan.Summary.Failed != 0 {
		t.Fatalf("unexpected summary: %#v", plan.Summary)
	}
}

func TestPlanLifecycleWarnsForMissingGeneratedPluginPackage(t *testing.T) {
	repo := t.TempDir()
	mustWriteLifecycle(t, filepath.Join(repo, "scripts", "install-codex-kit.ps1"), "param()\n")
	writeLifecycleRegistry(t, repo, []string{"aicoding-platform"})
	writeLifecycleManifest(t, repo, "aicoding-platform", `"install":{"type":"powershell-script","path":"scripts/install-codex-kit.ps1","supportsDryRun":true}`, `"pluginRoot":"CodingKit/agents/skills/plugins/AiCoding"`)

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	plan := PlanLifecycle(repo, entries, LifecycleOptions{Action: "install", Mode: "all", DryRun: true})
	if !plan.OK || plan.Summary.Warnings != 1 {
		t.Fatalf("expected ok plan with one warning: %#v", plan)
	}
	if len(plan.Kits) != 1 || len(plan.Kits[0].Warnings) != 1 {
		t.Fatalf("expected kit warning: %#v", plan.Kits)
	}
}

func TestPlanLifecycleRequiredPathsMissingFails(t *testing.T) {
	repo := t.TempDir()
	writeLifecycleRegistry(t, repo, []string{"required-kit"})
	writeLifecycleManifest(t, repo, "required-kit", `"status":{"type":"builtin-check","requiredPaths":["README.md"]}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	plan := PlanLifecycle(repo, entries, LifecycleOptions{Action: "status", Mode: "all", DryRun: false})
	if plan.OK || plan.Summary.Failed != 1 {
		t.Fatalf("expected required path failure: %#v", plan)
	}
	if len(plan.Kits[0].MissingRequiredPaths) != 1 || plan.Kits[0].MissingRequiredPaths[0] != "README.md" {
		t.Fatalf("expected missing README.md, got %#v", plan.Kits[0].MissingRequiredPaths)
	}
}

func TestPlanLifecycleAggregatesOK(t *testing.T) {
	repo := t.TempDir()
	mustWriteLifecycle(t, filepath.Join(repo, "scripts", "install.ps1"), "param()\n")
	writeLifecycleRegistry(t, repo, []string{"script-kit", "asset-kit"})
	writeLifecycleManifest(t, repo, "script-kit", `"install":{"type":"powershell-script","path":"scripts/install.ps1","supportsDryRun":true}`, "")
	writeLifecycleManifest(t, repo, "asset-kit", `"install":{"type":"unsupported","reason":"repository asset only"}`, "")

	entries, err := LoadRegistry(repo)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	plan := PlanLifecycle(repo, entries, LifecycleOptions{Action: "install", Mode: "all", DryRun: true})
	if !plan.OK || plan.Summary.Total != 2 || plan.Summary.OK != 2 || plan.Summary.Skipped != 1 {
		t.Fatalf("unexpected aggregation: %#v", plan)
	}
}

func writeLifecycleRegistry(t *testing.T, repo string, ids []string) {
	t.Helper()
	kits := ""
	for i, id := range ids {
		if i > 0 {
			kits += ","
		}
		kits += `{"id":"` + id + `","enabled":true,"order":` + strconv.Itoa(i+1) + `,"manifest":"config/kits/` + id + `.json"}`
	}
	mustWriteLifecycle(t, filepath.Join(repo, "config", "kit-registry.json"), `{"schemaVersion":1,"name":"test","defaultMode":"repo-scoped","kits":[`+kits+`]}`)
}

func writeLifecycleManifest(t *testing.T, repo, id, commands, paths string) {
	t.Helper()
	if paths == "" {
		paths = `"root":"."`
	}
	content := `{"schemaVersion":2,"id":"` + id + `","name":"` + id + `","version":"0.1.0","kind":["test"],"mode":"script-adapter","paths":{` + paths + `},"commands":{` + commands + `}}`
	mustWriteLifecycle(t, filepath.Join(repo, "config", "kits", id+".json"), content)
}

func mustWriteLifecycle(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
