package kit

import "encoding/json"

type registry struct {
	SchemaVersion int           `json:"schemaVersion"`
	Name          string        `json:"name"`
	DefaultMode   string        `json:"defaultMode"`
	Kits          []RegistryKit `json:"kits"`
}

type RegistryKit struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	Order    int    `json:"order"`
	Manifest string `json:"manifest"`
}

type Manifest struct {
	SchemaVersion int                               `json:"schemaVersion"`
	ID            string                            `json:"id"`
	Name          string                            `json:"name"`
	Version       string                            `json:"version"`
	Kind          []string                          `json:"kind"`
	Mode          string                            `json:"mode"`
	Description   string                            `json:"description"`
	Paths         map[string]string                 `json:"paths"`
	Commands      map[string]CommandDef             `json:"commands"`
	Skills        map[string]json.RawMessage        `json:"skills"`
	Hooks         map[string]json.RawMessage        `json:"hooks"`
	State         map[string]string                 `json:"state"`
	Trust         map[string]interface{}            `json:"trust"`
	Profiles      map[string]map[string]interface{} `json:"profiles"`
}

type CommandDef struct {
	Type           string   `json:"type"`
	Path           string   `json:"path"`
	Executable     string   `json:"executable"`
	Args           []string `json:"args"`
	Steps          []string `json:"steps"`
	RequiredPaths  []string `json:"requiredPaths"`
	SupportsJSON   *bool    `json:"supportsJson"`
	SupportsDryRun bool     `json:"supportsDryRun"`
	Reason         string   `json:"reason"`
}

type View struct {
	ID       string   `json:"id"`
	Enabled  bool     `json:"enabled"`
	Order    int      `json:"order"`
	Name     string   `json:"name,omitempty"`
	Version  string   `json:"version,omitempty"`
	Kind     []string `json:"kind,omitempty"`
	Mode     string   `json:"mode,omitempty"`
	Manifest string   `json:"manifest"`
}

type SmokeResult struct {
	ID       string   `json:"id"`
	OK       bool     `json:"ok"`
	Status   string   `json:"status"`
	Manifest string   `json:"manifest"`
	Errors   []string `json:"errors"`
}
