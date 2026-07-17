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
