package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func TestSkillInitCLIProducesPreviewAndExternalScaffold(t *testing.T) {
	repo := t.TempDir()
	preview, err := runSkill([]string{"init", "demo-skill", "--dry-run", "--repo-root", repo, "--json"}, time.Now())
	data, ok := preview.Data.(kit.SkillInitReport)
	if err != nil || !preview.OK || !ok || !strings.Contains(data.Content, "## Verification") {
		t.Fatalf("Skill init preview failed: result=%#v err=%v", preview, err)
	}

	out := t.TempDir()
	created, err := runSkill([]string{"init", "demo-skill", "--out", out, "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !created.OK {
		t.Fatalf("Skill init output failed: result=%#v err=%v", created, err)
	}
	if _, err := os.Stat(filepath.Join(out, "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	readOnly := filepath.Join(repo, "CodingKit", "agents", "skills", "demo-skill")
	rejected, rejectErr := runSkill([]string{"init", "demo-skill", "--out", readOnly, "--repo-root", repo, "--json"}, time.Now())
	if rejectErr == nil || rejected.OK || rejected.Category != report.CategoryValidation || !strings.Contains(strings.Join(rejected.Errors, " "), "read-only") {
		t.Fatalf("Skill read-only output was not rejected: result=%#v err=%v", rejected, rejectErr)
	}
}

func TestMCPInitCLIProducesCompliantManifestAndSuggestion(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "config", "mcp-registry.json"), `{"schemaVersion":1,"name":"test","components":[]}`+"\n")
	preview, err := runMCP([]string{"init", "demo-mcp", "--dry-run", "--repo-root", repo, "--json"}, time.Now())
	data, ok := preview.Data.(mcpcontrol.InitReport)
	if err != nil || !preview.OK || !ok || data.RegistryEntry.ID != "demo-mcp" || data.RegistryEntry.Enabled {
		t.Fatalf("MCP init preview failed: result=%#v err=%v", preview, err)
	}

	out := t.TempDir()
	created, err := runMCP([]string{"init", "demo-mcp", "--out", out, "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !created.OK {
		t.Fatalf("MCP init output failed: result=%#v err=%v", created, err)
	}
	if _, err := os.Stat(filepath.Join(out, "demo-mcp.json")); err != nil {
		t.Fatal(err)
	}
}
