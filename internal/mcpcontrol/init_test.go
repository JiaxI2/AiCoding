package mcpcontrol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPInitPreviewDryRunAndManifestWrite(t *testing.T) {
	repo := t.TempDir()
	writeMCPInitRegistry(t, repo)
	preview, err := InitComponentScaffold(repo, "demo-mcp", InitOptions{DryRun: true})
	if err != nil || !preview.OK || preview.OutputMode != "preview" || !strings.Contains(preview.ComponentContent, `"ownsWorkflowPrompts": false`) {
		t.Fatalf("MCP preview failed: report=%#v err=%v", preview, err)
	}
	if preview.RegistryEntry.ID != "demo-mcp" || preview.RegistryEntry.Enabled || preview.RegistryEntry.Order != 30 || preview.RegistryEntry.Manifest != "config/mcp/components/demo-mcp.json" {
		t.Fatalf("MCP registry suggestion is invalid: %#v", preview.RegistryEntry)
	}

	out := t.TempDir()
	dryRun, err := InitComponentScaffold(repo, "demo-mcp", InitOptions{Out: out, DryRun: true})
	target := filepath.Join(out, "demo-mcp.json")
	if err != nil || !dryRun.OK || dryRun.Files[0].Action != "planned-create" {
		t.Fatalf("MCP output dry-run failed: report=%#v err=%v", dryRun, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("MCP dry-run wrote a file: %v", err)
	}

	created, err := InitComponentScaffold(repo, "demo-mcp", InitOptions{Out: out})
	if err != nil || !created.OK || created.Files[0].Action != "created" {
		t.Fatalf("MCP output failed: report=%#v err=%v", created, err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateMCPInitContent(content, "demo-mcp"); err != nil {
		t.Fatalf("generated MCP manifest is invalid: %v", err)
	}
	before := string(content)
	duplicate, duplicateErr := InitComponentScaffold(repo, "demo-mcp", InitOptions{Out: out})
	if duplicateErr == nil || duplicate.OK || !strings.Contains(duplicateErr.Error(), "will not be overwritten") {
		t.Fatalf("duplicate MCP init was not fail-closed: report=%#v err=%v", duplicate, duplicateErr)
	}
	after, err := os.ReadFile(target)
	if err != nil || string(after) != before {
		t.Fatalf("duplicate MCP init changed the target: err=%v", err)
	}
}

func TestMCPInitRejectsRegisteredReservedAndInvalidIDs(t *testing.T) {
	repo := t.TempDir()
	writeMCPInitRegistry(t, repo)
	for _, id := range []string{"visio-mcp", "aicoding-demo", "Demo_MCP"} {
		report, err := InitComponentScaffold(repo, id, InitOptions{})
		if err == nil || report.OK {
			t.Fatalf("MCP id %q was accepted: report=%#v err=%v", id, report, err)
		}
	}
}

func writeMCPInitRegistry(t *testing.T, repo string) {
	t.Helper()
	writeTestFile(t, filepath.Join(repo, "config", "mcp-registry.json"), `{
  "schemaVersion": 1,
  "name": "test",
  "components": [
    {"id":"visio-mcp","enabled":true,"order":10,"manifest":"config/mcp/components/visio-mcp.json"},
    {"id":"ppt-mcp","enabled":true,"order":20,"manifest":"config/mcp/components/ppt-mcp.json"}
  ]
}
`)
}
