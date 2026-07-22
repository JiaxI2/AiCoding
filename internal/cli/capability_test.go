package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	capabilitydomain "github.com/JiaxI2/AiCoding/internal/capability"
)

func TestCatalogHasPublicEntryUsesTypedHelpForms(t *testing.T) {
	for _, entry := range []string{
		"aicoding --help",
		"aicoding test",
		"aicoding lifecycle status",
		"aicoding capability list",
		"aicoding governance capabilities",
		"aicoding work record",
	} {
		if !catalogHasPublicEntry(entry) {
			t.Errorf("expected typed catalog to contain %q", entry)
		}
	}
	for _, entry := range []string{"aicoding", "other capability list", "aicoding nonexistent", "aicoding capability nonexistent"} {
		if catalogHasPublicEntry(entry) {
			t.Errorf("typed catalog unexpectedly contains %q", entry)
		}
	}
}

func TestCapabilityCLIListsDescribesAndGeneratesIndexes(t *testing.T) {
	repo := capabilityCLIFixture(t, "aicoding capability list")
	start := time.Now()

	listed, err := runCapability([]string{"list", "--type", "domain-capability", "--status", "beta", "--repo-root", repo, "--json"}, start)
	if err != nil || !listed.OK || listed.InputDigest == "" {
		t.Fatalf("capability list = %#v, err = %v", listed, err)
	}
	items, ok := listed.Data.([]capabilitydomain.Capability)
	if !ok || len(items) != 1 || items[0].ID != "alpha" {
		t.Fatalf("list data = %#v", listed.Data)
	}

	described, err := runCapability([]string{"describe", "--id", "alpha", "--repo-root", repo, "--json"}, start)
	if err != nil || !described.OK {
		t.Fatalf("capability describe = %#v, err = %v", described, err)
	}
	detail, ok := described.Data.(capabilitydomain.Capability)
	if !ok || detail.ID != "alpha" || detail.Quickstart == nil || detail.Activation == nil {
		t.Fatalf("capability describe = %#v", described)
	}
	if detail.Quickstart.Steps[0] != "aicoding capability list --json" || detail.Activation.Kind != "cli-entry" {
		t.Fatalf("capability describe usage closure = %#v", detail)
	}
	invalid, err := runCapability([]string{"list", "--type", "unknown", "--repo-root", repo}, start)
	if err == nil || invalid.ErrorKind != "validation" {
		t.Fatalf("unknown filter = %#v, err = %v", invalid, err)
	}
	if _, err := runCapability([]string{"index", "--repo-root", repo}, start); !isUsageError(err) {
		t.Fatalf("index without --write error = %v", err)
	}

	generated, err := runCapability([]string{"index", "--write", "--repo-root", repo, "--json"}, start)
	if err != nil || !generated.OK {
		t.Fatalf("capability index = %#v, err = %v", generated, err)
	}
	changed := generated.Data.(capabilityIndexWrite).Changed
	if len(changed) != 2 || changed[0] != "README.md" || changed[1] != capabilitydomain.CapabilitiesPath {
		t.Fatalf("changed paths = %#v", changed)
	}
	if raw, readErr := os.ReadFile(filepath.Join(repo, "README.md")); readErr != nil || !strings.Contains(string(raw), "完整的 1 项能力") || strings.Contains(string(raw), "\nold\n") {
		t.Fatalf("generated README = %q, err = %v", raw, readErr)
	}
	if raw, readErr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(capabilitydomain.CapabilitiesPath))); readErr != nil || !strings.Contains(string(raw), "alpha capability") {
		t.Fatalf("generated capability document = %q, err = %v", raw, readErr)
	}
	second, err := runCapability([]string{"index", "--write", "--repo-root", repo}, start)
	if err != nil || len(second.Data.(capabilityIndexWrite).Changed) != 0 {
		t.Fatalf("second capability index = %#v, err = %v", second, err)
	}
}

func TestGovernanceCapabilitiesRejectsUnknownPublicEntry(t *testing.T) {
	repo := capabilityCLIFixture(t, "aicoding capability list")
	if _, err := runCapability([]string{"index", "--write", "--repo-root", repo}, time.Now()); err != nil {
		t.Fatal(err)
	}
	result, err := runGovernance([]string{"capabilities", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !result.OK {
		t.Fatalf("governance capabilities = %#v, err = %v", result, err)
	}

	writeCapabilityCLIFile(t, repo, capabilitydomain.CatalogPath, capabilityCLICatalog("aicoding capability nonexistent"))
	result, err = runGovernance([]string{"capabilities", "--repo-root", repo}, time.Now())
	if err == nil || result.OK || !containsCapabilityCLIError(result.Errors, "public entry is absent from typed command catalog") {
		t.Fatalf("unknown public entry was not rejected: %#v, err = %v", result, err)
	}
}

func capabilityCLIFixture(t *testing.T, publicEntry string) string {
	t.Helper()
	repo := t.TempDir()
	writeCapabilityCLIFile(t, repo, "internal/alpha/alpha.go", "package alpha\n")
	writeCapabilityCLIFile(t, repo, "docs/architecture/alpha.md", "# Alpha\n")
	writeCapabilityCLIFile(t, repo, "README.md", "# Fixture\n\n<!-- BEGIN GENERATED: CAPABILITIES -->\n<!-- END GENERATED: CAPABILITIES -->\n")
	writeCapabilityCLIFile(t, repo, capabilitydomain.CatalogPath, capabilityCLICatalog(publicEntry))
	return repo
}

func capabilityCLICatalog(publicEntry string) string {
	return `{
  "schemaVersion": 1,
  "name": "fixture capabilities",
  "capabilities": [
    {
      "id": "alpha",
      "package": "internal/alpha",
      "name": "Alpha",
      "type": "domain-capability",
      "status": "beta",
      "summary": "alpha capability",
      "publicEntries": ["` + publicEntry + `"],
      "architectureDoc": "docs/architecture/alpha.md",
      "quickstart": {"steps": ["aicoding capability list --json"]},
      "activation": {
        "kind": "cli-entry",
        "note": "already available",
        "agentUsage": "aicoding capability list --json"
      },
      "verification": ["go test ./internal/alpha/..."]
    }
  ]
}`
}

func writeCapabilityCLIFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsCapabilityCLIError(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
