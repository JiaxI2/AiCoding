package registry

import "testing"

func TestCatalogSnapshotCombinesRegistryAndReferencedObjects(t *testing.T) {
	registry, err := NewSnapshot("test-registry", map[string]interface{}{
		"entries": []string{"alpha", "beta"},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := NewCatalogSnapshot("test-catalog", registry, []CatalogEntry{
		{ID: "beta", Path: "manifests/beta.json", Digest: "sha256:beta"},
		{ID: "alpha", Path: "manifests/alpha.json", Digest: "sha256:alpha"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCatalogSnapshot("test-catalog", registry, []CatalogEntry{
		{ID: "alpha", Path: "manifests/alpha.json", Digest: "sha256:alpha"},
		{ID: "beta", Path: "manifests/beta.json", Digest: "sha256:beta"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() {
		t.Fatalf("catalog digest depends on load order: %q != %q", first.Digest(), second.Digest())
	}
	if first.RegistryDigest() != registry.Digest() {
		t.Fatalf("registry digest = %q, want %q", first.RegistryDigest(), registry.Digest())
	}

	changed, err := NewCatalogSnapshot("test-catalog", registry, []CatalogEntry{
		{ID: "alpha", Path: "manifests/alpha.json", Digest: "sha256:changed"},
		{ID: "beta", Path: "manifests/beta.json", Digest: "sha256:beta"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() == changed.Digest() {
		t.Fatal("referenced object change did not change catalog digest")
	}
}

func TestCatalogSnapshotRejectsIncompleteOrDuplicateEntries(t *testing.T) {
	registry, err := NewSnapshot("test-registry", map[string]string{"name": "test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, entries := range [][]CatalogEntry{
		{{ID: "", Path: "a", Digest: "sha256:a"}},
		{{ID: "a", Path: "", Digest: "sha256:a"}},
		{{ID: "a", Path: "a", Digest: ""}},
		{{ID: "a", Path: "a", Digest: "sha256:a"}, {ID: "a", Path: "b", Digest: "sha256:b"}},
	} {
		if _, err := NewCatalogSnapshot("test-catalog", registry, entries); err == nil {
			t.Fatalf("invalid catalog entries were accepted: %#v", entries)
		}
	}
}
