package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistrySnapshotNormalizesOrderAndProtectsEntries(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "config", "kit-registry.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	first := `{"schemaVersion":1,"name":"test","defaultMode":"enabled","kits":[{"id":"late","enabled":true,"order":20,"manifest":"late.json"},{"id":"early","enabled":true,"order":10,"manifest":"early.json"}]}`
	if err := os.WriteFile(path, []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadRegistrySnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	entries := snapshot.Entries()
	if len(entries) != 2 || entries[0].ID != "early" || !strings.HasPrefix(snapshot.Digest(), "sha256:") {
		t.Fatalf("unexpected snapshot: %q %#v", snapshot.Digest(), entries)
	}
	entries[0].ID = "mutated"
	if snapshot.Entries()[0].ID != "early" {
		t.Fatal("snapshot entries were mutable through the returned slice")
	}

	second := `{
  "kits": [
    {"manifest":"early.json","order":10,"enabled":true,"id":"early"},
    {"manifest":"late.json","order":20,"enabled":true,"id":"late"}
  ],
  "defaultMode":"enabled", "name":"test", "schemaVersion":1
}`
	if err := os.WriteFile(path, []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadRegistrySnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Digest() != reloaded.Digest() {
		t.Fatalf("format-only rewrite changed digest: %q != %q", snapshot.Digest(), reloaded.Digest())
	}
}

func TestCatalogSnapshotIncludesManifestContentAndReturnsDetachedValues(t *testing.T) {
	repo := t.TempDir()
	writeRegistryTestFile(t, filepath.Join(repo, "config", "kit-registry.json"), `{
  "schemaVersion":1,
  "name":"test",
  "defaultMode":"enabled",
  "kits":[{"id":"sample","enabled":true,"order":10,"manifest":"config/kits/sample.json"}]
}`)
	manifestPath := filepath.Join(repo, "config", "kits", "sample.json")
	writeRegistryTestFile(t, manifestPath, `{
  "schemaVersion":2,
  "id":"sample",
  "name":"Sample",
  "version":"1.0.0",
  "kind":["test"],
  "mode":"go-builtin",
  "commands":{"status":{"type":"builtin-check"}}
}`)
	first, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	items := first.Kits()
	if len(items) != 1 || !strings.HasPrefix(first.Digest(), "sha256:") {
		t.Fatalf("unexpected catalog: %q %#v", first.Digest(), items)
	}
	manifest, err := items[0].Manifest()
	if err != nil {
		t.Fatal(err)
	}
	manifest.Commands["status"] = CommandDef{Type: "mutated"}
	detached, err := first.Kits()[0].Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if detached.Commands["status"].Type != "builtin-check" {
		t.Fatal("catalog manifest was mutable through decoded value")
	}
	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}
	plan := PlanCatalogLifecycle(repo, first.Kits(), LifecycleOptions{Action: "status", Mode: "all"})
	if !plan.OK {
		t.Fatalf("catalog plan reread a removed manifest: %#v", plan)
	}
	action := RunCatalogAction(repo, first.Kits(), ActionOptions{Action: "status", Mode: "all"})
	if !action.OK {
		t.Fatalf("catalog action reread a removed manifest: %#v", action)
	}
	smoke := SmokeCatalogKits(repo, first.Kits())
	if len(smoke) != 1 || !smoke[0].OK {
		t.Fatalf("catalog smoke reread a removed manifest: %#v", smoke)
	}

	writeRegistryTestFile(t, manifestPath, `{
  "schemaVersion":2,
  "id":"sample",
  "name":"Changed",
  "version":"1.0.0",
  "kind":["test"],
  "mode":"go-builtin",
  "commands":{"status":{"type":"builtin-check"}}
}`)
	second, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if first.RegistryDigest() != second.RegistryDigest() || first.Digest() == second.Digest() {
		t.Fatalf("manifest-only change was not isolated: registry %q/%q catalog %q/%q",
			first.RegistryDigest(), second.RegistryDigest(), first.Digest(), second.Digest())
	}
}

func writeRegistryTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
