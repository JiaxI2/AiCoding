package mcpcontrol

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

func TestInventoryLoadsManagedAndConfiguredServers(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "config", "mcp-registry.json"), `{
  "schemaVersion": 1,
  "name": "test",
  "components": [
    {"id":"visio-mcp","enabled":true,"order":10,"manifest":"config/mcp/components/visio-mcp.json"}
  ]
}`)
	writeTestFile(t, filepath.Join(repo, "config", "mcp", "components", "visio-mcp.json"), `{
  "schemaVersion":1,
  "id":"visio-mcp",
  "name":"Visio",
  "version":"0.1.0",
  "transport":"stdio",
  "runtime":{"kind":"python-venv","root":"asset","requirements":"requirements.txt","minimumPython":"3.10","pythonEnvVar":"VISIO_MCP_PYTHON","module":"visio_mcp","packageInstall":["-e","."],"serverArgs":["-m","visio_mcp","server"],"env":{"VISIO_MCP_ROOT":"${componentRoot}"}},
  "codex":{"serverName":"visio-mcp","startupTimeoutSec":30,"toolTimeoutSec":120},
  "doctor":{"args":["-m","visio_mcp","doctor","--json"]},
  "verify":{"Smoke":[["-m","pytest"]],"Full":[["-m","pytest"]],"Release":[["-m","pytest"]]}
}`)
	writeTestFile(t, filepath.Join(repo, "asset", "requirements.txt"), "example\n")
	config := filepath.Join(repo, "config.toml")
	writeTestFile(t, config, `
[mcp_servers.local]
command = "helper.exe"
args = ["serve"]

[mcp_servers.local.env]
TOKEN = "secret"

[mcp_servers.remote]
url = "https://example.com/mcp"
bearer_token_env_var = "REMOTE_TOKEN"
`)

	inventory, err := ListInventory(repo, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Managed) != 1 || len(inventory.Configured) != 2 {
		t.Fatalf("unexpected inventory: %#v", inventory)
	}
	if len(inventory.Configured[0].EnvKeys) != 1 || inventory.Configured[0].EnvKeys[0] != "TOKEN" {
		t.Fatalf("environment keys were not redacted: %#v", inventory.Configured[0])
	}
}

func TestManagedCodexBlockBackupAndRemoval(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.toml")
	original := "personality = \"pragmatic\"\n"
	writeTestFile(t, config, original)
	component := testComponent()
	root := filepath.Join(t.TempDir(), "visio")
	python := filepath.Join(root, ".venv", "Scripts", "python.exe")

	backup, err := writeManagedBlock(config, component, python, root)
	if err != nil {
		t.Fatal(err)
	}
	if backup == "" {
		t.Fatal("expected timestamped backup")
	}
	if data, err := os.ReadFile(backup); err != nil || string(data) != original {
		t.Fatalf("unexpected backup: %v %q", err, data)
	}
	managedConfig, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(managedConfig), "[mcp_servers.visio-mcp.env]") || !strings.Contains(string(managedConfig), "VISIO_MCP_ROOT") {
		t.Fatalf("runtime environment was not rendered: %s", managedConfig)
	}
	managed, collision, err := managedBlockStatus(config, component.Codex.ServerName)
	if err != nil || !managed || collision {
		t.Fatalf("unexpected managed block status: managed=%v collision=%v err=%v", managed, collision, err)
	}
	if _, err := removeManagedBlock(config, component.Codex.ServerName); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(original) {
		t.Fatalf("unexpected config after removal: %q", data)
	}
}

func TestManagedCodexBlockRefusesCollision(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.toml")
	writeTestFile(t, config, "[mcp_servers.visio-mcp]\ncommand = \"user.exe\"\n")
	_, err := writeManagedBlock(config, testComponent(), "python.exe", "root")
	if err == nil || !strings.Contains(err.Error(), "unmanaged") {
		t.Fatalf("expected unmanaged collision, got %v", err)
	}
}

func TestStagedVenvCanBeRestoredAndRemoved(t *testing.T) {
	root := t.TempDir()
	venvFile := filepath.Join(root, ".venv", "Scripts", "python.exe")
	writeTestFile(t, venvFile, "python")

	staged, err := stageOwnedVenv(root)
	if err != nil {
		t.Fatal(err)
	}
	if platform.IsDir(filepath.Join(root, ".venv")) || !platform.IsDir(staged) {
		t.Fatalf("unexpected staged venv state: %q", staged)
	}
	if err := restoreStagedVenv(root, staged); err != nil {
		t.Fatal(err)
	}
	if !platform.IsFile(venvFile) {
		t.Fatal("restored venv is missing")
	}
	staged, err = stageOwnedVenv(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := removeStagedVenv(root, staged); err != nil {
		t.Fatal(err)
	}
	if platform.Exists(staged) || platform.Exists(filepath.Join(root, ".venv")) {
		t.Fatal("owned venv was not removed")
	}
}

func TestProbeStdio(t *testing.T) {
	endpoint := Endpoint{
		ID:                "helper",
		Transport:         "stdio",
		Command:           os.Args[0],
		Args:              []string{"-test.run=TestMCPHelperProcess"},
		Env:               map[string]string{"GO_WANT_MCP_HELPER": "1"},
		StartupTimeoutSec: 10,
	}
	result := Probe(context.Background(), endpoint)
	if !result.OK {
		t.Fatalf("stdio probe failed: %#v", result)
	}
	if result.ProtocolVersion != ProtocolVersion || result.ToolCount != 2 || result.ResourceCount != 1 || result.PromptCount != 1 {
		t.Fatalf("unexpected stdio probe: %#v", result)
	}
}

func TestProbeHTTPWithSSEDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		if request.Method == http.MethodDelete {
			writer.WriteHeader(http.StatusOK)
			return
		}
		var message struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(request.Body).Decode(&message); err != nil {
			t.Fatal(err)
		}
		if message.ID == 0 {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		var result interface{}
		switch message.Method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": ProtocolVersion,
				"capabilities": map[string]interface{}{
					"tools":     map[string]interface{}{},
					"resources": map[string]interface{}{},
					"prompts":   map[string]interface{}{},
				},
				"serverInfo": map[string]string{"name": "test", "version": "1"},
			}
		case "tools/list":
			result = map[string]interface{}{"tools": []map[string]string{{"name": "one"}}}
		case "resources/list":
			result = map[string]interface{}{"resources": []map[string]string{{"uri": "test://one"}}}
		case "prompts/list":
			result = map[string]interface{}{"prompts": []map[string]string{{"name": "one"}}}
		default:
			t.Fatalf("unexpected method: %s", message.Method)
		}
		response := map[string]interface{}{"jsonrpc": "2.0", "id": message.ID, "result": result}
		if message.Method == "tools/list" {
			writer.Header().Set("Content-Type", "text/event-stream")
			data, _ := json.Marshal(response)
			fmt.Fprintf(writer, "data: %s\n\n", data)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	result := Probe(context.Background(), Endpoint{
		ID:                "remote",
		Transport:         "streamable-http",
		URL:               server.URL,
		StartupTimeoutSec: 10,
	})
	if !result.OK || result.ToolCount != 1 || result.ResourceCount != 1 || result.PromptCount != 1 {
		t.Fatalf("unexpected HTTP probe: %#v", result)
	}
}

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var message struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &message) != nil || message.ID == 0 {
			continue
		}
		var result interface{}
		switch message.Method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": ProtocolVersion,
				"capabilities": map[string]interface{}{
					"tools":     map[string]interface{}{},
					"resources": map[string]interface{}{},
					"prompts":   map[string]interface{}{},
				},
				"serverInfo": map[string]string{"name": "helper", "version": "1"},
			}
		case "tools/list":
			result = map[string]interface{}{"tools": []map[string]string{{"name": "one"}, {"name": "two"}}}
		case "resources/list":
			result = map[string]interface{}{"resources": []map[string]string{{"uri": "test://one"}}}
		case "prompts/list":
			result = map[string]interface{}{"prompts": []map[string]string{{"name": "one"}}}
		default:
			_ = encoder.Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      message.ID,
				"error":   map[string]interface{}{"code": -32601, "message": "Method not found"},
			})
			continue
		}
		_ = encoder.Encode(map[string]interface{}{"jsonrpc": "2.0", "id": message.ID, "result": result})
	}
	os.Exit(0)
}

func testComponent() Component {
	return Component{
		ID:      "visio-mcp",
		Name:    "Visio",
		Version: "0.1.0",
		Runtime: PythonRuntime{
			Root:           "asset",
			PythonEnvVar:   "VISIO_MCP_PYTHON",
			PackageInstall: []string{"-e", "."},
			ServerArgs:     []string{"-m", "visio_mcp", "server"},
			Env:            map[string]string{"VISIO_MCP_ROOT": "${componentRoot}"},
		},
		Codex: CodexRegistration{
			ServerName:        "visio-mcp",
			StartupTimeoutSec: 30,
			ToolTimeoutSec:    120,
		},
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
