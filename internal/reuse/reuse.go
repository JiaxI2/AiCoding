package reuse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

const configPath = "config/reuse-governance.json"

var moduleIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

var expectedIntegrations = []string{
	"go-cli",
	"skill-verify",
	"hook",
	"ci",
	"docsync",
	"lifecycle",
}

type registry struct {
	SchemaVersion int      `json:"schemaVersion"`
	Policy        policy   `json:"policy"`
	Modules       []module `json:"modules"`
}

type policy struct {
	RequireAttributionForCopiedContent bool `json:"requireAttributionForCopiedContent"`
	RequireIndependentRuntime          bool `json:"requireIndependentRuntime"`
	RequireRollback                    bool `json:"requireRollback"`
	RequireNoPublicAPI                 bool `json:"requireNoPublicAPI"`
}

type module struct {
	ID                     string     `json:"id"`
	Classification         string     `json:"classification"`
	State                  string     `json:"state"`
	LiteralExternalContent bool       `json:"literalExternalContent"`
	RuntimeDependency      bool       `json:"runtimeDependency"`
	PublicAPI              bool       `json:"publicAPI"`
	AttributionNotice      string     `json:"attributionNotice,omitempty"`
	Integrations           []string   `json:"integrations"`
	RequiredPaths          []string   `json:"requiredPaths"`
	Evidence               []evidence `json:"evidence"`
	Rollback               rollback   `json:"rollback"`
}

type evidence struct {
	Integration string `json:"integration"`
	Path        string `json:"path"`
	Contains    string `json:"contains"`
}

type rollback struct {
	Strategy  string `json:"strategy"`
	StatePath string `json:"statePath"`
}

type Report struct {
	SchemaVersion int            `json:"schemaVersion"`
	OK            bool           `json:"ok"`
	Config        string         `json:"config"`
	Summary       Summary        `json:"summary"`
	Modules       []ModuleResult `json:"modules"`
	Warnings      []string       `json:"warnings"`
	Errors        []string       `json:"errors"`
}

type Summary struct {
	Modules       int `json:"modules"`
	Pilot         int `json:"pilot"`
	Direct        int `json:"direct"`
	Modified      int `json:"modified"`
	Reimplemented int `json:"reimplemented"`
	IdeaOnly      int `json:"ideaOnly"`
	NotAdopted    int `json:"notAdopted"`
}

type ModuleResult struct {
	ID             string           `json:"id"`
	Classification string           `json:"classification"`
	State          string           `json:"state"`
	OK             bool             `json:"ok"`
	Evidence       []EvidenceResult `json:"evidence"`
	Errors         []string         `json:"errors,omitempty"`
}

type EvidenceResult struct {
	Integration string `json:"integration"`
	Path        string `json:"path"`
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
}

func Verify(repo string) Report {
	report := Report{
		SchemaVersion: 1,
		OK:            true,
		Config:        configPath,
		Modules:       []ModuleResult{},
		Warnings:      []string{},
		Errors:        []string{},
	}

	raw, err := os.ReadFile(platform.RepoPath(repo, configPath))
	if err != nil {
		report.Errors = append(report.Errors, "cannot read reuse governance config: "+err.Error())
		report.OK = false
		return report
	}

	var reg registry
	if err := json.Unmarshal(raw, &reg); err != nil {
		report.Errors = append(report.Errors, "cannot parse reuse governance config: "+err.Error())
		report.OK = false
		return report
	}
	if reg.SchemaVersion != 1 {
		report.Errors = append(report.Errors, "unsupported reuse governance schemaVersion")
	}
	report.Errors = append(report.Errors, validatePolicy(reg.Policy)...)

	seen := map[string]bool{}
	for _, item := range reg.Modules {
		result := ModuleResult{ID: item.ID, Classification: item.Classification, State: item.State, OK: true, Evidence: []EvidenceResult{}}
		if item.ID == "" || !moduleIDPattern.MatchString(item.ID) {
			result.Errors = append(result.Errors, "invalid module id")
		} else if seen[item.ID] {
			result.Errors = append(result.Errors, "duplicate module id")
		}
		seen[item.ID] = true
		moduleErrors, evidence := validateModule(repo, item)
		result.Errors = append(result.Errors, moduleErrors...)
		result.Evidence = evidence
		result.Errors = uniqueSorted(result.Errors)
		result.OK = len(result.Errors) == 0
		report.Modules = append(report.Modules, result)
		report.Summary.Modules++
		if item.State == "pilot" {
			report.Summary.Pilot++
		}
		switch item.Classification {
		case "direct-reuse":
			report.Summary.Direct++
		case "modified-reuse":
			report.Summary.Modified++
		case "reimplemented":
			report.Summary.Reimplemented++
		case "idea-only":
			report.Summary.IdeaOnly++
		case "not-adopted":
			report.Summary.NotAdopted++
		}
		for _, err := range result.Errors {
			report.Errors = append(report.Errors, item.ID+": "+err)
		}
	}

	if len(reg.Modules) == 0 {
		report.Errors = append(report.Errors, "reuse governance config has no modules")
	}
	report.Errors = uniqueSorted(report.Errors)
	report.OK = len(report.Errors) == 0
	return report
}

func validatePolicy(p policy) []string {
	errs := []string{}
	if !p.RequireAttributionForCopiedContent {
		errs = append(errs, "policy must require attribution for copied content")
	}
	if !p.RequireIndependentRuntime {
		errs = append(errs, "policy must require an independent runtime")
	}
	if !p.RequireRollback {
		errs = append(errs, "policy must require rollback")
	}
	if !p.RequireNoPublicAPI {
		errs = append(errs, "policy must prohibit public API coupling")
	}
	return errs
}

func validateModule(repo string, item module) ([]string, []EvidenceResult) {
	errs := []string{}
	if !validClassification(item.Classification) {
		errs = append(errs, "invalid classification")
	}
	if item.State != "pilot" && item.State != "active" && item.State != "retired" {
		errs = append(errs, "invalid module state")
	}
	if item.RuntimeDependency {
		errs = append(errs, "runtime dependency is not allowed")
	}
	if item.PublicAPI {
		errs = append(errs, "public API coupling is not allowed")
	}
	if item.Classification == "reimplemented" && item.LiteralExternalContent {
		errs = append(errs, "reimplemented module cannot contain copied content")
	}
	if (item.Classification == "direct-reuse" || item.Classification == "modified-reuse") && item.AttributionNotice == "" {
		errs = append(errs, "copied content requires an attribution notice path")
	}
	if item.AttributionNotice != "" && !pathExists(repo, item.AttributionNotice) {
		errs = append(errs, "attribution notice path is missing")
	}
	if item.Rollback.Strategy == "" || item.Rollback.StatePath == "" {
		errs = append(errs, "rollback strategy and state path are required")
	}
	for _, requiredPath := range item.RequiredPaths {
		if !pathExists(repo, requiredPath) {
			errs = append(errs, "required path is missing: "+requiredPath)
		}
	}
	evidenceErrors, evidenceResults := validateEvidence(repo, item)
	errs = append(errs, evidenceErrors...)
	return errs, evidenceResults
}

func validClassification(value string) bool {
	switch value {
	case "direct-reuse", "modified-reuse", "reimplemented", "idea-only", "not-adopted":
		return true
	default:
		return false
	}
}

func validateEvidence(repo string, item module) ([]string, []EvidenceResult) {
	errs := []string{}
	results := []EvidenceResult{}
	integrations := map[string]bool{}
	for _, integration := range item.Integrations {
		integrations[integration] = true
	}
	evidenceByIntegration := map[string]bool{}
	for _, proof := range item.Evidence {
		result := EvidenceResult{Integration: proof.Integration, Path: proof.Path}
		if proof.Integration == "" || proof.Path == "" || proof.Contains == "" {
			result.Error = "integration, path, and contains are required"
			errs = append(errs, "evidence requires integration, path, and contains")
			results = append(results, result)
			continue
		}
		evidenceByIntegration[proof.Integration] = true
		path, ok := safePath(repo, proof.Path)
		if !ok {
			result.Error = "path must be repository-relative"
			errs = append(errs, "evidence path must be repository-relative")
			results = append(results, result)
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			result.Error = "path is missing"
			errs = append(errs, "evidence path is missing: "+proof.Path)
			results = append(results, result)
			continue
		}
		if !strings.Contains(string(content), proof.Contains) {
			result.Error = "anchor is missing"
			errs = append(errs, "evidence anchor is missing: "+proof.Integration)
			results = append(results, result)
			continue
		}
		result.OK = true
		results = append(results, result)
	}
	for _, integration := range expectedIntegrations {
		if !integrations[integration] {
			errs = append(errs, "missing required integration: "+integration)
		}
		if !evidenceByIntegration[integration] {
			errs = append(errs, "missing evidence for integration: "+integration)
			results = append(results, EvidenceResult{Integration: integration, OK: false, Error: "evidence is missing"})
		}
	}
	return errs, results
}

func pathExists(repo, rel string) bool {
	path, ok := safePath(repo, rel)
	return ok && platform.Exists(path)
}

func safePath(repo, rel string) (string, bool) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", false
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return platform.RepoPath(repo, filepath.ToSlash(clean)), true
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
