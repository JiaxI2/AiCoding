package repohealth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

const (
	pwshBudgetConfigPath = "config/pwsh-budget.json"
	pwshBudgetScope      = "tools/specialty/*.ps1"
)

// PwshBudgetRatchet is the machine-readable PWSH-002 baseline comparison.
type PwshBudgetRatchet struct {
	OK                       bool     `json:"ok"`
	ConfigPath               string   `json:"configPath"`
	Scope                    string   `json:"scope"`
	BaselineRemainingScripts int      `json:"baselineRemainingScripts"`
	CurrentRemainingScripts  int      `json:"currentRemainingScripts"`
	BaselineUnspecified      int      `json:"baselineUnspecified"`
	CurrentUnspecified       int      `json:"currentUnspecified"`
	BaselineCommit           string   `json:"baselineCommit"`
	Evidence                 string   `json:"evidence"`
	UnexpectedScripts        []string `json:"unexpectedScripts"`
	MissingScripts           []string `json:"missingScripts"`
}

type pwshBudgetConfig struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Scope         string               `json:"scope"`
	Baselines     []pwshBudgetBaseline `json:"baselines"`
}

type pwshBudgetBaseline struct {
	RemainingScripts int      `json:"remainingScripts"`
	Unspecified      int      `json:"unspecified"`
	Scripts          []string `json:"scripts"`
	ObservedCommit   string   `json:"observedCommit"`
	Evidence         string   `json:"evidence"`
}

type pwshBudgetEvidence struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	OK            bool   `json:"ok"`
	Data          struct {
		Retirement struct {
			Scope            string `json:"scope"`
			RemainingScripts int    `json:"remainingScripts"`
			Unspecified      int    `json:"unspecified"`
		} `json:"retirement"`
	} `json:"data"`
}

func evaluatePwshBudgetRatchet(repo string) (PwshBudgetRatchet, []string) {
	ratchet := PwshBudgetRatchet{
		ConfigPath:        pwshBudgetConfigPath,
		Scope:             pwshBudgetScope,
		UnexpectedScripts: []string{},
		MissingScripts:    []string{},
	}
	errs := []string{}

	currentScripts, scanErrs := topLevelPwshScripts(repo)
	errs = append(errs, scanErrs...)
	retirement, retirementErrs := inspectPwshRetirement(repo)
	errs = append(errs, retirementErrs...)
	ratchet.CurrentRemainingScripts = retirement.RemainingScripts
	ratchet.CurrentUnspecified = retirement.Unspecified
	if len(currentScripts) != retirement.RemainingScripts {
		errs = append(errs, fmt.Sprintf("PowerShell inventory disagreement: paths=%d remainingScripts=%d", len(currentScripts), retirement.RemainingScripts))
	}

	config, configErrs := loadPwshBudgetConfig(repo)
	errs = append(errs, configErrs...)
	if len(config.Baselines) == 0 {
		ratchet.OK = false
		return ratchet, errs
	}
	latest := config.Baselines[len(config.Baselines)-1]
	ratchet.Scope = config.Scope
	ratchet.BaselineRemainingScripts = latest.RemainingScripts
	ratchet.BaselineUnspecified = latest.Unspecified
	ratchet.BaselineCommit = latest.ObservedCommit
	ratchet.Evidence = latest.Evidence

	baselineSet := make(map[string]struct{}, len(latest.Scripts))
	for _, path := range latest.Scripts {
		baselineSet[path] = struct{}{}
	}
	currentSet := make(map[string]struct{}, len(currentScripts))
	for _, path := range currentScripts {
		currentSet[path] = struct{}{}
		if _, ok := baselineSet[path]; !ok {
			ratchet.UnexpectedScripts = append(ratchet.UnexpectedScripts, path)
			errs = append(errs, "PowerShell script exceeds ratchet baseline: "+path)
			content, readErr := os.ReadFile(platform.RepoPath(repo, path))
			if readErr != nil {
				errs = append(errs, "cannot inspect new PowerShell script "+path+": "+readErr.Error())
			} else if pwshRetirementTrigger(string(content)) == "unspecified" {
				errs = append(errs, "PowerShell script is missing # RETIRE-AFTER: "+path)
			}
		}
	}
	for _, path := range latest.Scripts {
		if _, ok := currentSet[path]; !ok {
			ratchet.MissingScripts = append(ratchet.MissingScripts, path)
			errs = append(errs, "PowerShell baseline must be lowered in the same change after deleting: "+path)
		}
	}
	if retirement.RemainingScripts != latest.RemainingScripts {
		errs = append(errs, fmt.Sprintf("PowerShell remainingScripts=%d does not equal baseline=%d", retirement.RemainingScripts, latest.RemainingScripts))
	}
	if retirement.Unspecified > latest.Unspecified {
		for _, script := range retirement.Scripts {
			if script.RetirementTrigger == "unspecified" {
				errs = append(errs, "PowerShell retirement trigger became unspecified: "+script.Path)
			}
		}
		if retirement.Unspecified > 0 && len(retirement.Scripts) == 0 {
			errs = append(errs, fmt.Sprintf("PowerShell unspecified=%d exceeds baseline=%d", retirement.Unspecified, latest.Unspecified))
		}
	}

	ratchet.OK = len(errs) == 0
	return ratchet, errs
}

func loadPwshBudgetConfig(repo string) (pwshBudgetConfig, []string) {
	config := pwshBudgetConfig{}
	path := platform.RepoPath(repo, pwshBudgetConfigPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		return config, []string{"PowerShell budget config is missing or unreadable: " + err.Error()}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return pwshBudgetConfig{}, []string{"PowerShell budget config is invalid: " + err.Error()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return pwshBudgetConfig{}, []string{"PowerShell budget config contains trailing JSON content"}
	}

	errs := validatePwshBudgetConfig(repo, config)
	return config, errs
}

func validatePwshBudgetConfig(repo string, config pwshBudgetConfig) []string {
	errs := []string{}
	if config.SchemaVersion != 1 {
		errs = append(errs, fmt.Sprintf("PowerShell budget schemaVersion=%d, want 1", config.SchemaVersion))
	}
	if config.Scope != pwshBudgetScope {
		errs = append(errs, fmt.Sprintf("PowerShell budget scope=%q, want %q", config.Scope, pwshBudgetScope))
	}
	if len(config.Baselines) == 0 {
		return append(errs, "PowerShell budget baselines must not be empty")
	}

	var previous map[string]struct{}
	for index, baseline := range config.Baselines {
		label := fmt.Sprintf("PowerShell budget baseline[%d]", index)
		if baseline.RemainingScripts != len(baseline.Scripts) {
			errs = append(errs, fmt.Sprintf("%s remainingScripts=%d does not match scripts=%d", label, baseline.RemainingScripts, len(baseline.Scripts)))
		}
		if baseline.Unspecified != 0 {
			errs = append(errs, fmt.Sprintf("%s unspecified=%d, want 0", label, baseline.Unspecified))
		}
		if !isLowerHexCommit(baseline.ObservedCommit) {
			errs = append(errs, label+" observedCommit must be 40 lowercase hex")
		}
		if !safeRepoEvidencePath(baseline.Evidence) {
			errs = append(errs, label+" evidence must be a repository-relative JSON path under docs/operations/evidence")
		}

		sortedScripts := append([]string(nil), baseline.Scripts...)
		sort.Strings(sortedScripts)
		if !equalStrings(sortedScripts, baseline.Scripts) {
			errs = append(errs, label+" scripts must be sorted")
		}
		current := make(map[string]struct{}, len(baseline.Scripts))
		for _, script := range baseline.Scripts {
			if !validTopLevelPwshPath(script) {
				errs = append(errs, label+" has invalid script path: "+script)
			}
			if _, exists := current[script]; exists {
				errs = append(errs, label+" has duplicate script path: "+script)
			}
			current[script] = struct{}{}
			if previous != nil {
				if _, existed := previous[script]; !existed {
					errs = append(errs, label+" is not a strict subset; added path: "+script)
				}
			}
		}
		if previous != nil && len(current) >= len(previous) {
			errs = append(errs, fmt.Sprintf("%s must lower remainingScripts below %d", label, len(previous)))
		}
		if evidenceErr := validatePwshBudgetEvidence(repo, baseline, label); evidenceErr != "" {
			errs = append(errs, evidenceErr)
		}
		previous = current
	}
	return errs
}

func validatePwshBudgetEvidence(repo string, baseline pwshBudgetBaseline, label string) string {
	if !safeRepoEvidencePath(baseline.Evidence) {
		return ""
	}
	raw, err := os.ReadFile(platform.RepoPath(repo, baseline.Evidence))
	if err != nil {
		return label + " evidence is missing or unreadable: " + err.Error()
	}
	var evidence pwshBudgetEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		return label + " evidence is invalid JSON: " + err.Error()
	}
	retirement := evidence.Data.Retirement
	if evidence.SchemaVersion != 1 || evidence.Command != "doctor pwsh" || !evidence.OK || retirement.Scope != pwshBudgetScope ||
		retirement.RemainingScripts != baseline.RemainingScripts || retirement.Unspecified != baseline.Unspecified {
		return fmt.Sprintf("%s evidence does not prove remainingScripts=%d unspecified=%d", label, baseline.RemainingScripts, baseline.Unspecified)
	}
	return ""
}

func topLevelPwshScripts(repo string) ([]string, []string) {
	root := platform.RepoPath(repo, "tools/specialty")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return []string{}, []string{"cannot read tools/specialty: " + err.Error()}
	}
	paths := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".ps1") {
			continue
		}
		paths = append(paths, "tools/specialty/"+entry.Name())
	}
	sort.Strings(paths)
	return paths, nil
}

func validTopLevelPwshPath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && strings.HasPrefix(path, "tools/specialty/") &&
		!strings.Contains(strings.TrimPrefix(path, "tools/specialty/"), "/") &&
		strings.EqualFold(filepath.Ext(path), ".ps1")
}

func safeRepoEvidencePath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && strings.HasPrefix(path, "docs/operations/evidence/") &&
		strings.HasSuffix(strings.ToLower(path), ".json") && !filepath.IsAbs(filepath.FromSlash(path))
}

func isLowerHexCommit(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
