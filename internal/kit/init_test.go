package kit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/loopkit/workspec"
)

func TestKitInitDryRunThenGeneratedScaffoldPassesLifecycle(t *testing.T) {
	t.Parallel()
	repo := structureRepo(t, true)
	registryPath := filepath.Join(repo, "config", "kit-registry.json")
	originalRegistry := []byte("{\n  \"schemaVersion\": 1,\n  \"name\": \"test\",\n  \"defaultMode\": \"repo-scoped\",\n  \"kits\": []\n}\n")
	mustWriteLifecycle(t, registryPath, string(originalRegistry))
	dependencyPath := filepath.Join(repo, "config", "dependency-governance.json")
	originalDependency := writeKitInitDependencyPolicy(t, repo)

	dryRun, err := Init(repo, "tmp-kit", InitOptions{DryRun: true})
	if err != nil || !dryRun.OK || len(dryRun.Files) != 4 || dryRun.Order != 10 {
		t.Fatalf("dry-run failed: report=%#v err=%v", dryRun, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "config", "kits", "tmp-kit.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote a manifest: %v", err)
	}
	if current, err := os.ReadFile(registryPath); err != nil || string(current) != string(originalRegistry) {
		t.Fatalf("dry-run changed registry: err=%v\n%s", err, current)
	}
	if current, err := os.ReadFile(dependencyPath); err != nil || string(current) != string(originalDependency) {
		t.Fatalf("dry-run changed dependency governance: err=%v\n%s", err, current)
	}

	created, err := Init(repo, "tmp-kit", InitOptions{})
	if err != nil || !created.OK || created.Enabled || created.Order != 10 {
		t.Fatalf("init failed: report=%#v err=%v", created, err)
	}
	for _, file := range created.Files {
		if file.Digest == "" || file.Bytes == 0 || (file.Action != "created" && file.Action != "updated") {
			t.Fatalf("incomplete file evidence: %#v", file)
		}
	}

	catalog, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := catalog.Select("tmp-kit", false)
	if err != nil {
		t.Fatal(err)
	}
	policy := PluginProjectionPolicy{Adapter: PluginAdapter{
		Scope: "kit", StateOwner: "kit", Entrypoint: "go-static",
		Actions: []PluginAdapterAction{{Name: "verify", Effect: "read"}},
	}}
	verification := VerifyCatalogStructure(repo, selected, policy)
	if !verification.OK {
		t.Fatalf("generated scaffold did not pass Lifecycle: %#v", verification)
	}
	binding := readKitInitDependencyBinding(t, dependencyPath, "tmp-kit")
	if binding.Layer != "capability" || !binding.PlatformAgnostic || len(binding.Roots) != 0 || len(binding.DependsOn) != 0 {
		t.Fatalf("generated dependency binding is not the conservative scaffold default: %#v", binding)
	}

	workSpecContent, err := os.ReadFile(filepath.Join(repo, "testdata", "kits", "tmp-kit", "workspec-example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var spec workspec.Spec
	if err := json.Unmarshal(workSpecContent, &spec); err != nil || spec.Validate() != nil {
		t.Fatalf("generated workspec is invalid: spec=%#v decode=%v validate=%v", spec, err, spec.Validate())
	}

	manifestBefore, err := os.ReadFile(filepath.Join(repo, "config", "kits", "tmp-kit.json"))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate, duplicateErr := Init(repo, "tmp-kit", InitOptions{}); duplicateErr == nil || duplicate.OK || !strings.Contains(duplicateErr.Error(), "already registered") {
		t.Fatalf("duplicate init was not fail-closed: report=%#v err=%v", duplicate, duplicateErr)
	}
	manifestAfter, err := os.ReadFile(filepath.Join(repo, "config", "kits", "tmp-kit.json"))
	if err != nil || string(manifestAfter) != string(manifestBefore) {
		t.Fatalf("duplicate init changed the manifest: err=%v", err)
	}
	if bindings := countKitInitDependencyBindings(t, dependencyPath, "tmp-kit"); bindings != 1 {
		t.Fatalf("duplicate init changed dependency bindings: count=%d", bindings)
	}
}

func TestKitInitExternalBoundaryAndInputPolicy(t *testing.T) {
	t.Parallel()
	repo := structureRepo(t, true)
	writeLifecycleRegistry(t, repo, []string{"existing-kit"})
	writeLifecycleManifest(t, repo, "existing-kit", `"status":{"type":"builtin-check","requiredPaths":[]}`, "")
	writeKitInitDependencyPolicy(t, repo, "existing-kit")

	report, err := Init(repo, "demo-ext", InitOptions{External: true})
	if err != nil || !report.OK || report.Order != 11 || len(report.Files) != 5 {
		t.Fatalf("external init failed: report=%#v err=%v", report, err)
	}
	manifest, err := LoadManifest(repo, "config/kits/demo-ext.json")
	if err != nil {
		t.Fatal(err)
	}
	thirdParty, _ := manifest.Trust["thirdParty"].(bool)
	if !thirdParty || manifest.Trust["updatePolicy"] != "pinned" {
		t.Fatalf("external trust anchors are missing: %#v", manifest.Trust)
	}
	boundary, err := os.ReadFile(filepath.Join(repo, "docs", "reference", "kits", "demo-ext-BOUNDARY.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, heading := range []string{"上游地址与 pin 策略", "控制面声明或入口", "不承担的门禁", "同步纪律"} {
		if !strings.Contains(string(boundary), heading) {
			t.Fatalf("boundary card is missing %q:\n%s", heading, boundary)
		}
	}

	for _, id := range []string{"a", "Bad-Kit", "aicoding-foo", "trailing-"} {
		invalid, invalidErr := Init(repo, id, InitOptions{})
		if invalidErr == nil || invalid.OK {
			t.Fatalf("invalid id passed: %q report=%#v", id, invalid)
		}
	}
}

func writeKitInitDependencyPolicy(t *testing.T, repo string, ids ...string) []byte {
	t.Helper()
	bindings := make([]json.RawMessage, 0, len(ids))
	for _, id := range ids {
		content, err := json.Marshal(kitInitDependencyBinding{
			ID: id, Layer: "capability", PlatformAgnostic: true,
			Roots: []string{}, DependsOn: []string{},
		})
		if err != nil {
			t.Fatal(err)
		}
		bindings = append(bindings, content)
	}
	policy := kitInitDependencyPolicy{
		SchemaVersion: 1,
		Name:          "fixture",
		Direction:     "higher-rank-may-depend-on-equal-or-lower-rank",
		KitRegistry: kitInitDependencyRegistry{
			Path: "config/kit-registry.json", Bindings: bindings,
		},
	}
	content, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	mustWriteLifecycle(t, filepath.Join(repo, "config", "dependency-governance.json"), string(content))
	return content
}

func readKitInitDependencyBinding(t *testing.T, path, id string) kitInitDependencyBinding {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy kitInitDependencyPolicy
	if err := decodeStrictJSON(content, &policy); err != nil {
		t.Fatal(err)
	}
	for _, raw := range policy.KitRegistry.Bindings {
		var binding kitInitDependencyBinding
		if err := decodeStrictJSON(raw, &binding); err != nil {
			t.Fatal(err)
		}
		if binding.ID == id {
			return binding
		}
	}
	t.Fatalf("dependency binding not found: %s", id)
	return kitInitDependencyBinding{}
}

func countKitInitDependencyBindings(t *testing.T, path, id string) int {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy kitInitDependencyPolicy
	if err := decodeStrictJSON(content, &policy); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, raw := range policy.KitRegistry.Bindings {
		var binding struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &binding); err != nil {
			t.Fatal(err)
		}
		if binding.ID == id {
			count++
		}
	}
	return count
}

func TestKitInitManifestValidationRejectsTemplateSchemaDrift(t *testing.T) {
	t.Parallel()
	data := kitInitTemplateData{
		ID: "demo-kit", Name: "Demo Kit", ManifestPath: "config/kits/demo-kit.json",
		BoundaryPath: "docs/reference/kits/demo-kit-BOUNDARY.md",
		WorkSpecPath: "testdata/kits/demo-kit/workspec-example.json",
		WorkSpecRoot: "testdata/kits/demo-kit",
	}
	content, err := renderKitInitTemplate("manifest.tmpl.json", data)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]interface{}
	if err := json.Unmarshal(content, &object); err != nil {
		t.Fatal(err)
	}
	object["schemaDrift"] = true
	broken, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateKitInitManifest(broken, data, false); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("schema drift was not rejected: %v", err)
	}
}
