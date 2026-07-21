package mcpcontrol

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	mcptemplates "github.com/JiaxI2/AiCoding/config/templates/mcp"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

var mcpInitIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
var mcpInitEnvPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

type InitOptions struct {
	Out    string
	DryRun bool
}

type InitReport struct {
	SchemaVersion    int           `json:"schemaVersion"`
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	DryRun           bool          `json:"dryRun"`
	OutputMode       string        `json:"outputMode"`
	OutputRoot       string        `json:"outputRoot,omitempty"`
	ManifestPath     string        `json:"manifestPath"`
	ComponentContent string        `json:"componentContent,omitempty"`
	RegistryEntry    RegistryEntry `json:"registryEntry"`
	OK               bool          `json:"ok"`
	Files            []InitFile    `json:"files"`
	Errors           []string      `json:"errors,omitempty"`
}

type InitFile struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Digest string `json:"digest"`
	Bytes  int    `json:"bytes"`
}

type mcpInitTemplateData struct {
	ID           string
	Name         string
	Module       string
	PythonEnvVar string
}

func InitComponentScaffold(repo, id string, opts InitOptions) (InitReport, error) {
	id = strings.TrimSpace(id)
	report := InitReport{
		SchemaVersion: 1, ID: id, DryRun: opts.DryRun,
		OutputMode: "preview", Files: []InitFile{},
	}
	if !mcpInitIDPattern.MatchString(id) || len(id) > 64 {
		return failMCPInit(report, fmt.Errorf("MCP component id must be lowercase hyphen-case with 2-64 characters: %s", id))
	}
	if strings.HasPrefix(id, "aicoding-") {
		return failMCPInit(report, fmt.Errorf("reusable MCP component id must not use the reserved aicoding- namespace: %s", id))
	}

	registryContent, err := os.ReadFile(platform.RepoPath(repo, "config/mcp-registry.json"))
	if err != nil {
		return failMCPInit(report, fmt.Errorf("read config/mcp-registry.json: %w", err))
	}
	var registry Registry
	if err := decodeMCPInitJSON(registryContent, &registry); err != nil {
		return failMCPInit(report, fmt.Errorf("parse config/mcp-registry.json: %w", err))
	}
	if registry.SchemaVersion != 1 || strings.TrimSpace(registry.Name) == "" {
		return failMCPInit(report, fmt.Errorf("config/mcp-registry.json identity is incomplete"))
	}
	maxOrder := 0
	manifestPath := "config/mcp/components/" + id + ".json"
	for _, entry := range registry.Components {
		if entry.ID == id || filepath.ToSlash(entry.Manifest) == manifestPath {
			return failMCPInit(report, fmt.Errorf("MCP component is already registered and no files were changed: %s", id))
		}
		if entry.Order > maxOrder {
			maxOrder = entry.Order
		}
	}

	report.Name = mcpInitDisplayName(id)
	report.ManifestPath = manifestPath
	report.RegistryEntry = RegistryEntry{ID: id, Enabled: false, Order: maxOrder + 10, Manifest: manifestPath}
	data := mcpInitTemplateData{
		ID: id, Name: report.Name,
		Module:       strings.ReplaceAll(id, "-", "_"),
		PythonEnvVar: strings.ToUpper(strings.ReplaceAll(id, "-", "_")) + "_PYTHON",
	}
	content, err := renderMCPInitTemplate(data)
	if err != nil {
		return failMCPInit(report, err)
	}
	if err := validateMCPInitContent(content, id); err != nil {
		return failMCPInit(report, fmt.Errorf("generated MCP component scaffold is invalid: %w", err))
	}

	out := strings.TrimSpace(opts.Out)
	if out == "" {
		report.ComponentContent = string(content)
		report.Files = append(report.Files, InitFile{
			Path: "stdout:" + id + ".json", Action: "preview",
			Digest: mcpInitContentDigest(content), Bytes: len(content),
		})
		report.OK = true
		return report, nil
	}
	outputRoot := filepath.FromSlash(out)
	if !filepath.IsAbs(outputRoot) {
		outputRoot = filepath.Join(repo, outputRoot)
	}
	outputRoot, err = filepath.Abs(outputRoot)
	if err != nil {
		return failMCPInit(report, fmt.Errorf("resolve MCP output directory: %w", err))
	}
	target := filepath.Join(outputRoot, id+".json")
	report.OutputMode = "directory"
	report.OutputRoot = outputRoot
	report.Files = append(report.Files, InitFile{
		Path: filepath.ToSlash(target), Action: "planned-create",
		Digest: mcpInitContentDigest(content), Bytes: len(content),
	})
	if _, statErr := os.Lstat(target); statErr == nil {
		return failMCPInit(report, fmt.Errorf("target already exists and will not be overwritten: %s", target))
	} else if !os.IsNotExist(statErr) {
		return failMCPInit(report, fmt.Errorf("inspect %s: %w", target, statErr))
	}
	if opts.DryRun {
		report.OK = true
		return report, nil
	}
	if err := writeMCPInitNewFile(target, content); err != nil {
		return failMCPInit(report, fmt.Errorf("create %s: %w", target, err))
	}
	report.Files[0].Action = "created"
	report.OK = true
	return report, nil
}

func failMCPInit(report InitReport, err error) (InitReport, error) {
	report.OK = false
	report.Errors = append(report.Errors, err.Error())
	return report, err
}

func renderMCPInitTemplate(data mcpInitTemplateData) ([]byte, error) {
	raw, err := mcptemplates.Files.ReadFile("component.tmpl.json")
	if err != nil {
		return nil, fmt.Errorf("read MCP init template: %w", err)
	}
	tmpl, err := template.New("component.tmpl.json").Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse MCP init template: %w", err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render MCP init template: %w", err)
	}
	return rendered.Bytes(), nil
}

func validateMCPInitContent(content []byte, expectedID string) error {
	var component Component
	if err := decodeMCPInitJSON(content, &component); err != nil {
		return err
	}
	if component.SchemaVersion != 1 || component.ID != expectedID || strings.TrimSpace(component.Name) == "" || component.Version != "0.1.0" {
		return fmt.Errorf("component identity is incomplete")
	}
	if component.Transport != "stdio" || component.Runtime.Kind != "python-venv" {
		return fmt.Errorf("component transport or runtime kind is invalid")
	}
	if strings.TrimSpace(component.Description) == "" || component.Runtime.Root == "" || component.Runtime.Requirements == "" || component.Runtime.Module == "" {
		return fmt.Errorf("component runtime description is incomplete")
	}
	if !mcpInitEnvPattern.MatchString(component.Runtime.PythonEnvVar) || len(component.Runtime.PackageInstall) == 0 || len(component.Runtime.ServerArgs) == 0 {
		return fmt.Errorf("component runtime execution contract is incomplete")
	}
	if component.Codex.ServerName != expectedID || component.Codex.StartupTimeoutSec < 1 || component.Codex.ToolTimeoutSec < 1 || len(component.Doctor.Args) == 0 {
		return fmt.Errorf("component Codex or doctor contract is incomplete")
	}
	for _, profile := range []string{"Smoke", "Full", "Release"} {
		if len(component.Verify[profile]) == 0 {
			return fmt.Errorf("component verify profile is missing: %s", profile)
		}
	}
	if ownsPrompts, ok := component.Security["ownsWorkflowPrompts"].(bool); !ok || ownsPrompts {
		return fmt.Errorf("capability MCP scaffold must explicitly reject workflow prompt ownership")
	}
	return nil
}

func decodeMCPInitJSON(content []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func mcpInitDisplayName(id string) string {
	parts := strings.Split(id, "-")
	for index, part := range parts {
		if part != "" {
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func mcpInitContentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeMCPInitNewFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}
