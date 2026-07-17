package asset

import "time"

type Type string

const (
	TypeKit      Type = "kit"
	TypeSkill    Type = "skill"
	TypeMCP      Type = "mcp"
	TypeTemplate Type = "template"
	TypeRuleset  Type = "ruleset"
	TypeProfile  Type = "profile"
)

type InstallMode string

const (
	ModeManaged  InstallMode = "managed"
	ModeEditable InstallMode = "editable"
)

type Dependency struct {
	ID         string `json:"id"`
	Constraint string `json:"constraint,omitempty"`
	Optional   bool   `json:"optional,omitempty"`
}

type Entrypoints struct {
	Install   string `json:"install,omitempty"`
	Update    string `json:"update,omitempty"`
	Uninstall string `json:"uninstall,omitempty"`
	Verify    string `json:"verify,omitempty"`
	Doctor    string `json:"doctor,omitempty"`
	Run       string `json:"run,omitempty"`
}

type Paths struct {
	Payload  string `json:"payload"`
	Defaults string `json:"defaults,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Tests    string `json:"tests,omitempty"`
}

type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	ID            string         `json:"id"`
	Type          Type           `json:"type"`
	Version       string         `json:"version"`
	Name          string         `json:"name,omitempty"`
	Description   string         `json:"description,omitempty"`
	Platforms     []string       `json:"platforms,omitempty"`
	Capabilities  []string       `json:"capabilities,omitempty"`
	Dependencies  []Dependency   `json:"dependencies,omitempty"`
	Entrypoints   Entrypoints    `json:"entrypoints,omitempty"`
	Paths         Paths          `json:"paths"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type LockEntry struct {
	ID          string      `json:"id"`
	Type        Type        `json:"type"`
	Version     string      `json:"version"`
	Mode        InstallMode `json:"mode"`
	Source      string      `json:"source"`
	Digest      string      `json:"digest"`
	InstalledAt time.Time   `json:"installedAt"`
	Files       []string    `json:"files"`
}

type Lockfile struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Assets        map[string]LockEntry `json:"assets"`
}

type ConfigLayers struct {
	Defaults   map[string]any
	Repository map[string]any
	User       map[string]any
	Local      map[string]any
	CLI        map[string]any
}

type Result struct {
	OK       bool           `json:"ok"`
	Action   string         `json:"action"`
	AssetID  string         `json:"assetId,omitempty"`
	Message  string         `json:"message,omitempty"`
	Changed  []string       `json:"changed,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}
