package capability

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadListAndDescribeReturnDetachedSortedViews(t *testing.T) {
	repo := t.TempDir()
	writeCapabilityTestFile(t, repo, CatalogPath, `{
  "schemaVersion": 1,
  "name": "test capabilities",
  "capabilities": [
    {
      "id": "zeta",
      "package": "internal/zeta",
      "name": "Zeta",
      "type": "internal-only",
      "status": "stable",
      "summary": "zeta capability",
      "publicEntries": [],
      "verification": ["go test ./internal/zeta/..."]
    },
    {
      "id": "alpha",
      "package": "internal/alpha",
      "name": "Alpha",
      "type": "domain-capability",
      "status": "beta",
      "summary": "alpha capability",
      "publicEntries": ["aicoding alpha list"],
      "architectureDoc": "docs/architecture/alpha.md",
      "quickstart": {
        "steps": ["aicoding alpha list --json"],
        "exampleInput": "testdata/alpha.json"
      },
      "activation": {
        "kind": "cli-entry",
        "note": "already available",
        "agentUsage": "aicoding alpha list --json"
      },
      "verification": ["go test ./internal/alpha/..."]
    }
  ]
}`)

	catalog, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(catalog.Digest, "sha256:") {
		t.Fatalf("digest = %q", catalog.Digest)
	}
	if got := []string{catalog.Capabilities[0].ID, catalog.Capabilities[1].ID}; !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Fatalf("sorted ids = %#v", got)
	}

	selected, err := List(catalog, "domain-capability", "beta")
	if err != nil || len(selected) != 1 || selected[0].ID != "alpha" {
		t.Fatalf("filtered list = %#v, err = %v", selected, err)
	}
	selected[0].PublicEntries[0] = "mutated"
	selected[0].Quickstart.Steps[0] = "mutated"
	selected[0].Activation.Note = "mutated"
	described, err := Describe(catalog, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if described.PublicEntries[0] != "aicoding alpha list" {
		t.Fatalf("catalog leaked mutable slice: %#v", described.PublicEntries)
	}
	if described.Quickstart.Steps[0] != "aicoding alpha list --json" || described.Activation.Note != "already available" {
		t.Fatalf("catalog leaked nested usage fields: %#v %#v", described.Quickstart, described.Activation)
	}
	if _, err := List(catalog, "unknown", ""); err == nil {
		t.Fatal("unknown type filter should fail")
	}
	if _, err := List(catalog, "", "unknown"); err == nil {
		t.Fatal("unknown status filter should fail")
	}
	if _, err := Describe(catalog, "missing"); err == nil {
		t.Fatal("unknown capability should fail")
	}
}

func TestVerifyReportsOrphansMissingEvidenceAndInvalidEntries(t *testing.T) {
	repo := t.TempDir()
	writeCapabilityTestFile(t, repo, "internal/alpha/alpha.go", "package alpha\n")
	writeCapabilityTestFile(t, repo, "internal/orphan/orphan.go", "package orphan\n")
	writeCapabilityTestFile(t, repo, CatalogPath, `{
  "schemaVersion": 1,
  "name": "test capabilities",
  "capabilities": [
    {
      "id": "alpha",
      "package": "internal/alpha",
      "name": "Alpha",
      "type": "domain-capability",
      "status": "stable",
      "summary": "alpha capability",
      "publicEntries": ["aicoding alpha list"],
      "architectureDoc": "docs/architecture/missing.md",
      "verification": []
    },
    {
      "id": "missing",
      "package": "internal/missing",
      "name": "Missing",
      "type": "internal-only",
      "status": "beta",
      "summary": "missing package",
      "publicEntries": []
    }
  ]
}`)
	catalog, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}

	result := Verify(repo, catalog, VerifyOptions{PublicEntryExists: func(string) bool { return false }})
	if result.OK {
		t.Fatal("invalid repository should fail verification")
	}
	assertCapabilityTestValue(t, result.Unregistered, "internal/orphan")
	assertCapabilityTestValue(t, result.MissingPackages, "internal/missing")
	assertCapabilityTestValue(t, result.MissingDocuments, "alpha: docs/architecture/missing.md")
	assertCapabilityTestValue(t, result.InvalidPublicEntries, "alpha: aicoding alpha list")
	assertCapabilityTestValue(t, result.StableWithoutVerification, "alpha")
}

func TestVerifyRejectsStablePublicCapabilityWithoutUsageClosure(t *testing.T) {
	repo := t.TempDir()
	writeCapabilityTestFile(t, repo, "internal/alpha/alpha.go", "package alpha\n")
	writeCapabilityTestFile(t, repo, "docs/architecture/alpha.md", "# Alpha\n")
	writeCapabilityTestFile(t, repo, CatalogPath, `{
  "schemaVersion": 1,
  "name": "test capabilities",
  "capabilities": [{
    "id": "alpha",
    "package": "internal/alpha",
    "name": "Alpha",
    "type": "domain-capability",
    "status": "stable",
    "summary": "alpha capability",
    "publicEntries": ["aicoding alpha list"],
    "architectureDoc": "docs/architecture/alpha.md",
    "verification": ["go test ./internal/alpha/..."]
  }]
}`)
	catalog, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}

	result := Verify(repo, catalog, VerifyOptions{PublicEntryExists: func(string) bool { return true }})
	if result.OK {
		t.Fatal("stable public capability without quickstart and activation must fail")
	}
	assertCapabilityTestValue(t, result.StablePublicWithoutQuickstart, "alpha")
	assertCapabilityTestValue(t, result.StablePublicWithoutActivation, "alpha")
	if !containsCapabilityTestSubstring(result.Errors, "stable public capability has no quickstart: alpha") ||
		!containsCapabilityTestSubstring(result.Errors, "stable public capability has no activation: alpha") {
		t.Fatalf("usage closure errors = %#v", result.Errors)
	}
}

func TestRenderIndexAndGeneratedVerification(t *testing.T) {
	repo := t.TempDir()
	writeCapabilityTestFile(t, repo, "internal/alpha/alpha.go", "package alpha\n")
	writeCapabilityTestFile(t, repo, "docs/architecture/alpha.md", "# Alpha\n")
	writeCapabilityTestFile(t, repo, CatalogPath, `{
  "schemaVersion": 1,
  "name": "test capabilities",
  "capabilities": [
    {
      "id": "alpha",
      "package": "internal/alpha",
      "name": "Alpha",
      "type": "product-workflow",
      "status": "stable",
      "summary": "alpha capability",
      "publicEntries": ["aicoding alpha"],
      "architectureDoc": "docs/architecture/alpha.md",
      "quickstart": {"steps": ["aicoding alpha --json"]},
      "activation": {
        "kind": "cli-entry",
        "note": "already available",
        "agentUsage": "aicoding alpha --json"
      },
      "verification": ["go test ./internal/alpha/..."]
    }
  ]
}`)
	readme := "# Test\n\n" + readmeBeginMarker + "\nold\n" + readmeEndMarker + "\n"
	writeCapabilityTestFile(t, repo, readmePath, readme)
	catalog, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}

	first, err := RenderIndex(catalog, readme)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderIndex(catalog, first.README)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("rendering the generated index should be deterministic")
	}
	for _, want := range []string{"docs/CAPABILITIES.md#capability-alpha", "aicoding alpha --json", "cli-entry", "capability describe --id alpha"} {
		if !strings.Contains(first.README+first.Document, want) {
			t.Fatalf("generated usage closure missing %q", want)
		}
	}
	writeCapabilityTestFile(t, repo, readmePath, first.README)
	writeCapabilityTestFile(t, repo, CapabilitiesPath, first.Document)
	result := Verify(repo, catalog, VerifyOptions{PublicEntryExists: func(entry string) bool { return entry == "aicoding alpha" }, CheckGenerated: true})
	if !result.OK || !result.READMEUpToDate || !result.DocumentUpToDate {
		t.Fatalf("generated verification = %#v", result)
	}

	writeCapabilityTestFile(t, repo, readmePath, strings.Replace(first.README, "alpha capability", "hand edited", 1))
	stale := Verify(repo, catalog, VerifyOptions{PublicEntryExists: func(string) bool { return true }, CheckGenerated: true})
	if stale.OK || stale.READMEUpToDate || !containsCapabilityTestSubstring(stale.Errors, "README capability index is stale") {
		t.Fatalf("hand-edited README was not rejected: %#v", stale)
	}
}

func TestListDoesNotScanAndVerifyReadsInternalDirectoryOnce(t *testing.T) {
	repo := t.TempDir()
	writeCapabilityTestFile(t, repo, "internal/alpha/alpha.go", "package alpha\n")
	catalog := Catalog{Capabilities: []Capability{{ID: "alpha", Package: "internal/alpha", Status: "beta"}}}
	original := readInternalDirectory
	defer func() { readInternalDirectory = original }()
	calls := 0
	readInternalDirectory = func(path string) ([]os.DirEntry, error) {
		calls++
		return os.ReadDir(path)
	}

	if _, err := List(catalog, "", ""); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("capability list scanned internal/: calls = %d", calls)
	}
	result := Verify(repo, catalog, VerifyOptions{})
	if !result.OK || calls != 1 {
		t.Fatalf("verify result = %#v, directory reads = %d", result, calls)
	}
}

func TestRenderIndexPreservesREADMEOutsideGeneratedBlock(t *testing.T) {
	catalog := Catalog{Digest: "sha256:test", Capabilities: []Capability{}}
	readme := "prefix\r\n" + readmeBeginMarker + "\r\nhand edited\r\n" + readmeEndMarker + "\r\nsuffix\r\n"
	rendered, err := RenderIndex(catalog, readme)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rendered.README, "prefix\r\n"+readmeBeginMarker+"\r\n") || !strings.HasSuffix(rendered.README, readmeEndMarker+"\r\nsuffix\r\n") {
		t.Fatalf("README content outside generated block changed: %q", rendered.README)
	}
}

func writeCapabilityTestFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertCapabilityTestValue(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%q not found in %#v", want, values)
}

func containsCapabilityTestSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
