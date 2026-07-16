package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
)

func TestRunMCPList(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "config", "mcp-registry.json"), `{
  "schemaVersion":1,
  "name":"test",
  "components":[{"id":"visio-mcp","enabled":true,"order":10,"manifest":"config/mcp/components/visio-mcp.json"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "mcp", "components", "visio-mcp.json"), `{
  "schemaVersion":1,
  "id":"visio-mcp",
  "name":"Visio",
  "version":"0.1.0",
  "transport":"stdio",
  "runtime":{"kind":"python-venv","root":"asset","requirements":"requirements.txt","minimumPython":"3.10","pythonEnvVar":"VISIO_MCP_PYTHON","module":"visio_mcp","packageInstall":["-e","."],"serverArgs":["-m","visio_mcp","server"],"env":{}},
  "codex":{"serverName":"visio-mcp","startupTimeoutSec":30,"toolTimeoutSec":120},
  "doctor":{"args":["-m","visio_mcp","doctor","--json"]},
  "verify":{"Smoke":[["-m","pytest"]],"Full":[["-m","pytest"]],"Release":[["-m","pytest"]]}
}`)
	mustWrite(t, filepath.Join(repo, "asset", "requirements.txt"), "example\n")
	config := filepath.Join(repo, "config.toml")
	mustWrite(t, config, "[mcp_servers.remote]\nurl = \"https://example.com/mcp\"\n")

	result, err := runMCP([]string{"list", "--repo-root", repo, "--codex-config", config, "--json"}, time.Now())
	if err != nil || !result.OK {
		t.Fatalf("mcp list failed: %v %#v", err, result)
	}
	inventory, ok := result.Data.(mcpcontrol.Inventory)
	if !ok {
		t.Fatalf("unexpected data type: %T", result.Data)
	}
	if len(inventory.Managed) != 1 || len(inventory.Configured) != 1 {
		t.Fatalf("unexpected inventory: %#v", inventory)
	}
}

func TestRunMCPLifecycleDryRun(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "config", "mcp-registry.json"), `{
  "schemaVersion":1,
  "name":"test",
  "components":[{"id":"visio-mcp","enabled":true,"order":10,"manifest":"config/mcp/components/visio-mcp.json"}]
}`)
	mustWrite(t, filepath.Join(repo, "config", "mcp", "components", "visio-mcp.json"), `{
  "schemaVersion":1,
  "id":"visio-mcp",
  "name":"Visio",
  "version":"0.1.0",
  "transport":"stdio",
  "runtime":{"kind":"python-venv","root":"asset","requirements":"requirements.txt","minimumPython":"3.10","pythonEnvVar":"VISIO_MCP_PYTHON","module":"visio_mcp","packageInstall":["-e","."],"serverArgs":["-m","visio_mcp","server"],"env":{}},
  "codex":{"serverName":"visio-mcp","startupTimeoutSec":30,"toolTimeoutSec":120},
  "doctor":{"args":["-m","visio_mcp","doctor","--json"]},
  "verify":{"Smoke":[["-m","pytest"]],"Full":[["-m","pytest"]],"Release":[["-m","pytest"]]}
}`)
	mustWrite(t, filepath.Join(repo, "asset", "requirements.txt"), "example\n")
	config := filepath.Join(repo, "config.toml")
	if err := os.WriteFile(config, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := runMCP(
		[]string{"install", "visio-mcp", "--dry-run", "--repo-root", repo, "--codex-config", config, "--json"},
		time.Now(),
	)
	if err != nil || !result.OK {
		t.Fatalf("mcp install dry-run failed: %v %#v", err, result)
	}
}
