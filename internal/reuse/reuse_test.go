package reuse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyAcceptsIndependentPilot(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, "reimplemented", "")

	report := Verify(repo)
	if !report.OK {
		t.Fatalf("expected valid governance report, got %#v", report.Errors)
	}
	if report.Summary.Pilot != 1 || report.Summary.Reimplemented != 1 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
	if len(report.Modules) != 1 || len(report.Modules[0].Evidence) != 6 {
		t.Fatalf("unexpected evidence output: %#v", report.Modules)
	}
	for _, proof := range report.Modules[0].Evidence {
		if !proof.OK {
			t.Fatalf("expected passing evidence, got %#v", proof)
		}
	}
}

func TestVerifyRejectsCopiedModuleWithoutNotice(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, "direct-reuse", "")

	report := Verify(repo)
	if report.OK || !strings.Contains(strings.Join(report.Errors, "\n"), "copied content requires an attribution notice path") {
		t.Fatalf("expected attribution failure, got %#v", report.Errors)
	}
}

func writeFixture(t *testing.T, repo, classification, attributionNotice string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, "internal", "cli", "cli.go"), "case \"reuse\":\nID: \"reuse governance\"\n")
	writeFile(t, filepath.Join(repo, "internal", "cli", "cli_ext.go"), "reuse.Verify(repo)\nID: \"reuse governance\"\n")
	writeFile(t, filepath.Join(repo, "docs", "operations", "THIRD_PARTY_REUSE_GOVERNANCE.md"), "DocSync\n")
	writeFile(t, filepath.Join(repo, "config", "kit-registry.json"), "reuse-governance\n")

	content := `{
  "schemaVersion": 1,
  "policy": {
    "requireAttributionForCopiedContent": true,
    "requireIndependentRuntime": true,
    "requireRollback": true,
    "requireNoPublicAPI": true
  },
  "modules": [
    {
      "id": "evidence-gate",
      "classification": "` + classification + `",
      "state": "pilot",
      "literalExternalContent": false,
      "runtimeDependency": false,
      "publicAPI": false,
      "attributionNotice": "` + attributionNotice + `",
      "integrations": ["go-cli", "skill-verify", "hook", "ci", "docsync", "lifecycle"],
      "requiredPaths": ["config/reuse-governance.json"],
      "evidence": [
        {"integration": "go-cli", "path": "internal/cli/cli.go", "contains": "case \"reuse\""},
        {"integration": "skill-verify", "path": "internal/cli/cli_ext.go", "contains": "reuse.Verify(repo)"},
        {"integration": "hook", "path": "internal/cli/cli.go", "contains": "ID: \"reuse governance\""},
        {"integration": "ci", "path": "internal/cli/cli_ext.go", "contains": "ID: \"reuse governance\""},
        {"integration": "docsync", "path": "docs/operations/THIRD_PARTY_REUSE_GOVERNANCE.md", "contains": "DocSync"},
        {"integration": "lifecycle", "path": "config/kit-registry.json", "contains": "reuse-governance"}
      ],
      "rollback": {"strategy": "remove", "statePath": ".aicoding/state/kits/reuse-governance"}
    }
  ]
}
`
	writeFile(t, filepath.Join(repo, configPath), content)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
