package mcpcontrol

import "encoding/json"

const ProtocolVersion = "2025-11-25"

type Registry struct {
	SchemaVersion int             `json:"schemaVersion"`
	Name          string          `json:"name"`
	Components    []RegistryEntry `json:"components"`
}

type RegistryEntry struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	Order    int    `json:"order"`
	Manifest string `json:"manifest"`
}

type Component struct {
	SchemaVersion int                    `json:"schemaVersion"`
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Transport     string                 `json:"transport"`
	Platforms     []string               `json:"platforms"`
	Runtime       PythonRuntime          `json:"runtime"`
	Codex         CodexRegistration      `json:"codex"`
	Doctor        CommandSpec            `json:"doctor"`
	Verify        map[string][][]string  `json:"verify"`
	Security      map[string]interface{} `json:"security"`
	Outputs       []string               `json:"outputs"`
}

type PythonRuntime struct {
	Kind           string            `json:"kind"`
	Root           string            `json:"root"`
	Requirements   string            `json:"requirements"`
	MinimumPython  string            `json:"minimumPython"`
	PythonEnvVar   string            `json:"pythonEnvVar"`
	Module         string            `json:"module"`
	PackageInstall []string          `json:"packageInstall"`
	ServerArgs     []string          `json:"serverArgs"`
	Env            map[string]string `json:"env"`
}

type CodexRegistration struct {
	ServerName        string `json:"serverName"`
	StartupTimeoutSec int    `json:"startupTimeoutSec"`
	ToolTimeoutSec    int    `json:"toolTimeoutSec"`
}

type CommandSpec struct {
	Args []string `json:"args"`
}

type ManagedView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Enabled     bool     `json:"enabled"`
	Order       int      `json:"order"`
	Manifest    string   `json:"manifest"`
	Transport   string   `json:"transport"`
	Platforms   []string `json:"platforms,omitempty"`
	RuntimeRoot string   `json:"runtimeRoot"`
}

type ConfiguredView struct {
	ID                string   `json:"id"`
	Enabled           bool     `json:"enabled"`
	Transport         string   `json:"transport"`
	Command           string   `json:"command,omitempty"`
	Args              []string `json:"args,omitempty"`
	URL               string   `json:"url,omitempty"`
	Cwd               string   `json:"cwd,omitempty"`
	EnvKeys           []string `json:"envKeys,omitempty"`
	BearerTokenEnvVar string   `json:"bearerTokenEnvVar,omitempty"`
	StartupTimeoutSec int      `json:"startupTimeoutSec,omitempty"`
	ToolTimeoutSec    int      `json:"toolTimeoutSec,omitempty"`
}

type Inventory struct {
	RegistryPath    string           `json:"registryPath"`
	RegistryDigest  string           `json:"registryDigest"`
	CodexConfigPath string           `json:"codexConfigPath"`
	Managed         []ManagedView    `json:"managed"`
	Configured      []ConfiguredView `json:"configured"`
	Warnings        []string         `json:"warnings,omitempty"`
}

type Endpoint struct {
	ID                string
	Transport         string
	Command           string
	Args              []string
	URL               string
	Cwd               string
	Env               map[string]string
	BearerTokenEnvVar string
	StartupTimeoutSec int
	ToolTimeoutSec    int
}

type CapabilitySummary struct {
	Tools     bool `json:"tools"`
	Resources bool `json:"resources"`
	Prompts   bool `json:"prompts"`
	Logging   bool `json:"logging"`
}

type ProbeResult struct {
	ID              string            `json:"id"`
	Transport       string            `json:"transport"`
	OK              bool              `json:"ok"`
	ProtocolVersion string            `json:"protocolVersion,omitempty"`
	Capabilities    CapabilitySummary `json:"capabilities"`
	ToolCount       int               `json:"toolCount,omitempty"`
	Tools           []string          `json:"tools,omitempty"`
	ResourceCount   int               `json:"resourceCount,omitempty"`
	PromptCount     int               `json:"promptCount"`
	Warnings        []string          `json:"warnings,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
}

type StatusResult struct {
	ID                 string   `json:"id"`
	OK                 bool     `json:"ok"`
	Root               string   `json:"root"`
	RootExists         bool     `json:"rootExists"`
	VenvPython         string   `json:"venvPython"`
	Installed          bool     `json:"installed"`
	CodexConfigPath    string   `json:"codexConfigPath"`
	Registered         bool     `json:"registered"`
	UnmanagedCollision bool     `json:"unmanagedCollision"`
	StatePath          string   `json:"statePath"`
	StateExists        bool     `json:"stateExists"`
	Warnings           []string `json:"warnings,omitempty"`
	Errors             []string `json:"errors,omitempty"`
}

type CommandResult struct {
	Command   []string        `json:"command"`
	OK        bool            `json:"ok"`
	Output    json.RawMessage `json:"output,omitempty"`
	RawOutput string          `json:"rawOutput,omitempty"`
	Errors    []string        `json:"errors,omitempty"`
	ElapsedMS int64           `json:"elapsedMs"`
}

type ComponentVerifyResult struct {
	ID       string          `json:"id"`
	Profile  string          `json:"profile"`
	OK       bool            `json:"ok"`
	Steps    []CommandResult `json:"steps"`
	Errors   []string        `json:"errors,omitempty"`
	Warnings []string        `json:"warnings,omitempty"`
}

type VerifyReport struct {
	Profile    string                  `json:"profile"`
	OK         bool                    `json:"ok"`
	Managed    []ComponentVerifyResult `json:"managed"`
	Configured []ProbeResult           `json:"configured"`
	Warnings   []string                `json:"warnings,omitempty"`
	Errors     []string                `json:"errors,omitempty"`
}

type LifecycleResult struct {
	ID              string   `json:"id"`
	Action          string   `json:"action"`
	DryRun          bool     `json:"dryRun"`
	OK              bool     `json:"ok"`
	Status          string   `json:"status"`
	Python          string   `json:"python,omitempty"`
	VenvPython      string   `json:"venvPython,omitempty"`
	CodexConfigPath string   `json:"codexConfigPath,omitempty"`
	BackupPath      string   `json:"backupPath,omitempty"`
	StatePath       string   `json:"statePath,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}
