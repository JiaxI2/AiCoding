package mcpcontrol

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/JiaxI2/AiCoding/internal/platform"
	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

type codexConfig struct {
	MCPServers map[string]codexServer `toml:"mcp_servers"`
}

type codexServer struct {
	Command           string            `toml:"command"`
	Args              []string          `toml:"args"`
	URL               string            `toml:"url"`
	Cwd               string            `toml:"cwd"`
	Env               map[string]string `toml:"env"`
	BearerTokenEnvVar string            `toml:"bearer_token_env_var"`
	Enabled           *bool             `toml:"enabled"`
	StartupTimeoutSec int               `toml:"startup_timeout_sec"`
	ToolTimeoutSec    int               `toml:"tool_timeout_sec"`
}

type RegistrySnapshot struct {
	object   registryobject.Snapshot
	registry Registry
}

func LoadRegistrySnapshot(repo string) (RegistrySnapshot, error) {
	path := platform.RepoPath(repo, "config/mcp-registry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return RegistrySnapshot{}, err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return RegistrySnapshot{}, err
	}
	sort.SliceStable(registry.Components, func(i, j int) bool {
		return registry.Components[i].Order < registry.Components[j].Order
	})
	object, err := registryobject.NewSnapshot("mcp-registry", registry)
	if err != nil {
		return RegistrySnapshot{}, err
	}
	registry.Components = cloneRegistryEntries(registry.Components)
	return RegistrySnapshot{object: object, registry: registry}, nil
}

func LoadRegistry(repo string) (Registry, error) {
	snapshot, err := LoadRegistrySnapshot(repo)
	if err != nil {
		return Registry{}, err
	}
	return snapshot.Registry(), nil
}

func (s RegistrySnapshot) Digest() string {
	return s.object.Digest()
}

func (s RegistrySnapshot) Object() registryobject.Snapshot {
	return s.object
}

func (s RegistrySnapshot) Registry() Registry {
	registry := s.registry
	registry.Components = cloneRegistryEntries(s.registry.Components)
	return registry
}

func cloneRegistryEntries(entries []RegistryEntry) []RegistryEntry {
	out := make([]RegistryEntry, len(entries))
	copy(out, entries)
	return out
}

func LoadComponent(repo, manifest string) (Component, error) {
	data, err := os.ReadFile(platform.RepoPath(repo, manifest))
	if err != nil {
		return Component{}, err
	}
	var component Component
	if err := json.Unmarshal(data, &component); err != nil {
		return Component{}, err
	}
	return component, nil
}

func SelectComponents(repo, id string, all bool) ([]RegistryEntry, error) {
	if all && id != "" {
		return nil, errors.New("use either --all or a component id, not both")
	}
	if !all && id == "" {
		return nil, errors.New("component selection requires --all or a component id")
	}
	registry, err := LoadRegistry(repo)
	if err != nil {
		return nil, err
	}
	selected := []RegistryEntry{}
	for _, entry := range registry.Components {
		if all && entry.Enabled {
			selected = append(selected, entry)
		}
		if id != "" && entry.ID == id {
			selected = append(selected, entry)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no MCP component matched")
	}
	return selected, nil
}

func ResolveCodexConfig(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return filepath.Join(home, "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func LoadConfigured(path string) ([]Endpoint, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Endpoint{}, nil
	}
	var config codexConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	endpoints := make([]Endpoint, 0, len(config.MCPServers))
	for id, server := range config.MCPServers {
		enabled := true
		if server.Enabled != nil {
			enabled = *server.Enabled
		}
		if !enabled {
			continue
		}
		transport := "stdio"
		if server.URL != "" {
			transport = "streamable-http"
		}
		timeout := server.StartupTimeoutSec
		if timeout <= 0 {
			timeout = 30
		}
		endpoints = append(endpoints, Endpoint{
			ID:                id,
			Transport:         transport,
			Command:           server.Command,
			Args:              append([]string{}, server.Args...),
			URL:               server.URL,
			Cwd:               server.Cwd,
			Env:               copyMap(server.Env),
			BearerTokenEnvVar: server.BearerTokenEnvVar,
			StartupTimeoutSec: timeout,
			ToolTimeoutSec:    server.ToolTimeoutSec,
		})
	}
	sort.Slice(endpoints, func(i, j int) bool {
		return strings.ToLower(endpoints[i].ID) < strings.ToLower(endpoints[j].ID)
	})
	return endpoints, nil
}

func ListInventory(repo, codexPath string) (Inventory, error) {
	snapshot, err := LoadRegistrySnapshot(repo)
	if err != nil {
		return Inventory{}, err
	}
	registry := snapshot.Registry()
	configPath, err := ResolveCodexConfig(codexPath)
	if err != nil {
		return Inventory{}, err
	}
	inventory := Inventory{
		RegistryPath:    platform.RepoPath(repo, "config/mcp-registry.json"),
		RegistryDigest:  snapshot.Digest(),
		CodexConfigPath: configPath,
		Managed:         []ManagedView{},
		Configured:      []ConfiguredView{},
	}
	for _, entry := range registry.Components {
		component, loadErr := LoadComponent(repo, entry.Manifest)
		if loadErr != nil {
			inventory.Warnings = append(inventory.Warnings, entry.ID+": "+loadErr.Error())
			continue
		}
		inventory.Managed = append(inventory.Managed, ManagedView{
			ID:          entry.ID,
			Name:        component.Name,
			Version:     component.Version,
			Enabled:     entry.Enabled,
			Order:       entry.Order,
			Manifest:    entry.Manifest,
			Transport:   component.Transport,
			Platforms:   append([]string{}, component.Platforms...),
			RuntimeRoot: component.Runtime.Root,
		})
	}
	configured, loadErr := LoadConfigured(configPath)
	if loadErr != nil {
		inventory.Warnings = append(inventory.Warnings, "Codex MCP config: "+loadErr.Error())
		return inventory, nil
	}
	for _, endpoint := range configured {
		keys := make([]string, 0, len(endpoint.Env))
		for key := range endpoint.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		inventory.Configured = append(inventory.Configured, ConfiguredView{
			ID:                endpoint.ID,
			Enabled:           true,
			Transport:         endpoint.Transport,
			Command:           endpoint.Command,
			Args:              append([]string{}, endpoint.Args...),
			URL:               endpoint.URL,
			Cwd:               endpoint.Cwd,
			EnvKeys:           keys,
			BearerTokenEnvVar: endpoint.BearerTokenEnvVar,
			StartupTimeoutSec: endpoint.StartupTimeoutSec,
			ToolTimeoutSec:    endpoint.ToolTimeoutSec,
		})
	}
	return inventory, nil
}

func DoctorRegistry(repo string) []string {
	errorsFound := []string{}
	registry, err := LoadRegistry(repo)
	if err != nil {
		return []string{err.Error()}
	}
	if registry.SchemaVersion != 1 {
		errorsFound = append(errorsFound, "mcp registry schemaVersion must be 1")
	}
	seen := map[string]bool{}
	for _, entry := range registry.Components {
		if entry.ID == "" || entry.Manifest == "" {
			errorsFound = append(errorsFound, "mcp registry entry requires id and manifest")
			continue
		}
		if seen[entry.ID] {
			errorsFound = append(errorsFound, "duplicate MCP component id: "+entry.ID)
		}
		seen[entry.ID] = true
		component, loadErr := LoadComponent(repo, entry.Manifest)
		if loadErr != nil {
			errorsFound = append(errorsFound, entry.ID+": "+loadErr.Error())
			continue
		}
		if component.SchemaVersion != 1 {
			errorsFound = append(errorsFound, entry.ID+": component schemaVersion must be 1")
		}
		if component.ID != entry.ID {
			errorsFound = append(errorsFound, entry.ID+": manifest id mismatch")
		}
		if component.Transport != "stdio" && component.Transport != "streamable-http" {
			errorsFound = append(errorsFound, entry.ID+": unsupported transport "+component.Transport)
		}
		if component.Runtime.PythonEnvVar == "" {
			errorsFound = append(errorsFound, entry.ID+": runtime pythonEnvVar is required")
		}
		if len(component.Runtime.PackageInstall) == 0 {
			errorsFound = append(errorsFound, entry.ID+": runtime packageInstall is required")
		}
		for _, rel := range []string{
			component.Runtime.Root,
			filepath.ToSlash(filepath.Join(component.Runtime.Root, component.Runtime.Requirements)),
			entry.Manifest,
		} {
			if rel == "" || !platform.Exists(platform.RepoPath(repo, rel)) {
				errorsFound = append(errorsFound, entry.ID+": missing "+rel)
			}
		}
		if len(component.Verify["Smoke"]) == 0 || len(component.Verify["Full"]) == 0 || len(component.Verify["Release"]) == 0 {
			errorsFound = append(errorsFound, entry.ID+": Smoke, Full and Release verify steps are required")
		}
	}
	for _, rel := range []string{
		"config/schemas/mcp-registry.schema.json",
		"config/schemas/mcp-component.schema.json",
	} {
		data, readErr := os.ReadFile(platform.RepoPath(repo, rel))
		if readErr != nil {
			errorsFound = append(errorsFound, readErr.Error())
			continue
		}
		var value interface{}
		if json.Unmarshal(data, &value) != nil {
			errorsFound = append(errorsFound, rel+": invalid JSON")
		}
	}
	return errorsFound
}

func copyMap(input map[string]string) map[string]string {
	output := map[string]string{}
	for key, value := range input {
		output[key] = value
	}
	return output
}
