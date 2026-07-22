package kit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterAddsPinnedManifestWithoutVendoringSource(t *testing.T) {
	repo := newPinnedConsumerRepository(t)
	writeRegistryTestFile(t, filepath.Join(repo, "config", "kit-registry.json"), `{"schemaVersion":1,"name":"test","defaultMode":"repo-scoped","kits":[]}`)
	writeRegistryTestFile(t, filepath.Join(repo, "config", "dependency-governance.json"), `{"schemaVersion":1,"name":"test","direction":"higher-rank-may-depend-on-equal-or-lower-rank","kitRegistry":{"path":"config/kit-registry.json","bindings":[]}}`)
	manifest := pinnedManifest("external-skill", "https://example.invalid/external.git", "0123456789abcdef0123456789abcdef01234567")
	writePinnedManifest(t, repo, manifest)

	report, err := Register(repo, "config/kits/external-skill.json")
	if err != nil || !report.OK || !report.Enabled || report.SourceIdentity == "" || len(report.Files) != 2 {
		t.Fatalf("register failed: report=%#v err=%v", report, err)
	}
	entries, err := LoadRegistry(repo)
	if err != nil || len(entries) != 1 || entries[0].ID != manifest.ID || !entries[0].Enabled {
		t.Fatalf("registered catalog is invalid: entries=%#v err=%v", entries, err)
	}
	var policy kitInitDependencyPolicy
	content, err := os.ReadFile(filepath.Join(repo, "config", "dependency-governance.json"))
	if err != nil || decodeStrictJSON(content, &policy) != nil || len(policy.KitRegistry.Bindings) != 1 {
		t.Fatalf("dependency binding was not added: policy=%#v err=%v", policy, err)
	}
	var binding kitInitDependencyBinding
	if err := json.Unmarshal(policy.KitRegistry.Bindings[0], &binding); err != nil || binding.ID != manifest.ID {
		t.Fatalf("dependency binding is invalid: binding=%#v err=%v", binding, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "skills", "external", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("registration vendored external source: %v", err)
	}
	t.Logf("registered=%s sourceIdentity=%s vendoredFiles=0", report.ID, report.SourceIdentity)
}

func TestRegisterRequiresPinnedSource(t *testing.T) {
	repo := newPinnedConsumerRepository(t)
	writeRegistryTestFile(t, filepath.Join(repo, "config", "kit-registry.json"), `{"schemaVersion":1,"name":"test","defaultMode":"repo-scoped","kits":[]}`)
	writeRegistryTestFile(t, filepath.Join(repo, "config", "dependency-governance.json"), `{"schemaVersion":1,"name":"test","direction":"higher-rank-may-depend-on-equal-or-lower-rank","kitRegistry":{"path":"config/kit-registry.json","bindings":[]}}`)
	manifest := pinnedManifest("external-skill", "https://example.invalid/external.git", "0123456789abcdef0123456789abcdef01234567")
	manifest.Source = nil
	writePinnedManifest(t, repo, manifest)
	before, err := os.ReadFile(filepath.Join(repo, "config", "kit-registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Register(repo, "config/kits/external-skill.json")
	after, readErr := os.ReadFile(filepath.Join(repo, "config", "kit-registry.json"))
	if err == nil || report.OK || readErr != nil || string(before) != string(after) {
		t.Fatalf("source-less registration did not fail atomically: report=%#v err=%v read=%v", report, err, readErr)
	}
}
