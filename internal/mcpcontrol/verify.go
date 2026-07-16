package mcpcontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func DoctorComponents(repo string, entries []RegistryEntry) []CommandResult {
	return DoctorComponentsContext(context.Background(), repo, entries)
}

func DoctorComponentsContext(ctx context.Context, repo string, entries []RegistryEntry) []CommandResult {
	results := make([]CommandResult, 0, len(entries))
	for _, entry := range entries {
		component, err := LoadComponent(repo, entry.Manifest)
		if err != nil {
			results = append(results, CommandResult{OK: false, Errors: []string{err.Error()}})
			continue
		}
		root := componentRoot(repo, component)
		python := venvPython(root)
		results = append(results, runPythonStep(ctx, root, python, component.Doctor.Args))
	}
	return results
}

func Verify(
	ctx context.Context,
	repo string,
	codexPath string,
	entries []RegistryEntry,
	profile string,
	includeConfigured bool,
) VerifyReport {
	normalized := normalizeProfile(profile)
	report := VerifyReport{
		Profile:    normalized,
		OK:         true,
		Managed:    []ComponentVerifyResult{},
		Configured: []ProbeResult{},
	}
	for _, entry := range entries {
		component, err := LoadComponent(repo, entry.Manifest)
		if err != nil {
			item := ComponentVerifyResult{ID: entry.ID, Profile: normalized, OK: false, Errors: []string{err.Error()}}
			report.Managed = append(report.Managed, item)
			report.Errors = append(report.Errors, entry.ID+": "+err.Error())
			report.OK = false
			continue
		}
		item := verifyComponent(ctx, repo, component, normalized)
		report.Managed = append(report.Managed, item)
		if !item.OK {
			report.OK = false
			for _, issue := range item.Errors {
				report.Errors = append(report.Errors, component.ID+": "+issue)
			}
		}
	}
	if includeConfigured {
		configPath, err := ResolveCodexConfig(codexPath)
		if err != nil {
			report.OK = false
			report.Errors = append(report.Errors, err.Error())
			return report
		}
		endpoints, err := LoadConfigured(configPath)
		if err != nil {
			report.OK = false
			report.Errors = append(report.Errors, err.Error())
			return report
		}
		report.Configured = ProbeAll(ctx, endpoints, 4)
		for _, probe := range report.Configured {
			if !probe.OK {
				report.OK = false
				for _, issue := range probe.Errors {
					report.Errors = append(report.Errors, probe.ID+": "+issue)
				}
			}
		}
		if len(endpoints) == 0 {
			report.Warnings = append(report.Warnings, "no configured Codex MCP servers were found")
		}
	}
	return report
}

func verifyComponent(ctx context.Context, repo string, component Component, profile string) ComponentVerifyResult {
	result := ComponentVerifyResult{
		ID:      component.ID,
		Profile: profile,
		OK:      true,
		Steps:   []CommandResult{},
	}
	steps := component.Verify[profile]
	if len(steps) == 0 {
		result.OK = false
		result.Errors = []string{"verify profile is not defined: " + profile}
		return result
	}
	root := componentRoot(repo, component)
	python := venvPython(root)
	for _, step := range steps {
		commandResult := runPythonStep(ctx, root, python, step)
		result.Steps = append(result.Steps, commandResult)
		if !commandResult.OK {
			result.OK = false
			result.Errors = append(result.Errors, commandResult.Errors...)
			break
		}
	}
	return result
}

func runPythonStep(ctx context.Context, root, python string, args []string) CommandResult {
	started := time.Now()
	result := CommandResult{
		Command: append([]string{python}, args...),
		OK:      false,
	}
	if python == "" {
		result.Errors = []string{"venv Python path is empty"}
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result
	}
	output, err := runNativeContext(ctx, root, python, args...)
	result.ElapsedMS = time.Since(started).Milliseconds()
	if err != nil {
		result.Errors = []string{err.Error()}
		result.RawOutput = output
		return result
	}
	result.OK = true
	if json.Valid([]byte(output)) {
		result.Output = json.RawMessage(output)
	} else {
		result.RawOutput = output
	}
	return result
}

func normalizeProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "smoke", "":
		return "Smoke"
	case "full":
		return "Full"
	case "release":
		return "Release"
	default:
		return profile
	}
}

func VerifyErrors(report VerifyReport) error {
	if report.OK {
		return nil
	}
	return fmt.Errorf("MCP verification failed with %d errors", len(report.Errors))
}
