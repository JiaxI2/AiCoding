package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SpecsRoot = "docs/spec"

const (
	StatusDraft         = "draft"
	StatusNeedsDecision = "needs-decision"
	StatusApproved      = "approved"
	StatusImplemented   = "implemented"
	StatusArchived      = "archived"
)

type Gate struct {
	Profile string `json:"profile"`
}

// Spec is the machine-readable PLAN.md frontmatter plus its repository path.
type Spec struct {
	File         string   `json:"file"`
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	Scope        []string `json:"scope"`
	ApprovedTree string   `json:"approvedTree"`
	Decision     string   `json:"decision,omitempty"`
	Gates        []Gate   `json:"gates"`
}

type Verification struct {
	SchemaVersion int      `json:"schemaVersion"`
	OK            bool     `json:"ok"`
	Specs         []Spec   `json:"specs"`
	Warnings      []string `json:"warnings"`
	Errors        []string `json:"errors"`
}

// ListSpecs reads only docs/spec/*/PLAN.md and returns a deterministic ID order.
func ListSpecs(repo string) ([]Spec, error) {
	files, err := specFiles(repo)
	if err != nil {
		return nil, err
	}
	specs := make([]Spec, 0, len(files))
	for _, file := range files {
		spec, err := parseSpec(repo, file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", displayPath(repo, file), err)
		}
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	return specs, nil
}

// VerifySpecs validates frontmatter and cross-file constraints without reading
// Markdown body semantics.
func VerifySpecs(repo string) (Verification, error) {
	verification := Verification{SchemaVersion: 1, OK: true, Specs: []Spec{}, Warnings: []string{}, Errors: []string{}}
	files, err := specFiles(repo)
	if err != nil {
		return verification, err
	}
	for _, file := range files {
		spec, parseErr := parseSpec(repo, file)
		if parseErr != nil {
			verification.Errors = append(verification.Errors, displayPath(repo, file)+": "+parseErr.Error())
			continue
		}
		verification.Specs = append(verification.Specs, spec)
		errs, warnings := verifySpec(repo, spec)
		verification.Errors = append(verification.Errors, errs...)
		verification.Warnings = append(verification.Warnings, warnings...)
	}
	sort.Slice(verification.Specs, func(i, j int) bool { return verification.Specs[i].ID < verification.Specs[j].ID })
	sort.Strings(verification.Errors)
	sort.Strings(verification.Warnings)
	verification.OK = len(verification.Errors) == 0
	return verification, nil
}

func specFiles(repo string) ([]string, error) {
	root := filepath.Join(repo, filepath.FromSlash(SpecsRoot))
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", SpecsRoot, err)
	}
	files := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		file := filepath.Join(root, entry.Name(), "PLAN.md")
		if info, statErr := os.Stat(file); statErr == nil && !info.IsDir() {
			files = append(files, file)
			continue
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return nil, statErr
		}
		files = append(files, file)
	}
	sort.Strings(files)
	return files, nil
}

func parseSpec(repo, file string) (Spec, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return Spec{}, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return Spec{}, errors.New("PLAN.md frontmatter is missing")
	}
	end := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			end = index
			break
		}
	}
	if end < 0 {
		return Spec{}, errors.New("PLAN.md frontmatter is not closed")
	}
	spec := Spec{File: displayPath(repo, file), Scope: []string{}, Gates: []Gate{}}
	seen := map[string]bool{}
	section := ""
	for index := 1; index < end; index++ {
		rawLine := lines[index]
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			switch section {
			case "scope":
				spec.Scope = append(spec.Scope, trimYAMLScalar(value))
			case "gates":
				key, scalar, ok := strings.Cut(value, ":")
				if !ok || strings.TrimSpace(key) != "profile" || trimYAMLScalar(scalar) == "" {
					return Spec{}, fmt.Errorf("frontmatter line %d has an invalid gate", index+1)
				}
				spec.Gates = append(spec.Gates, Gate{Profile: strings.ToLower(trimYAMLScalar(scalar))})
			default:
				return Spec{}, fmt.Errorf("frontmatter line %d has a list item outside scope or gates", index+1)
			}
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return Spec{}, fmt.Errorf("frontmatter line %d is not a key-value", index+1)
		}
		key = strings.TrimSpace(key)
		value = trimYAMLScalar(value)
		if seen[key] {
			return Spec{}, fmt.Errorf("frontmatter key %q is duplicated", key)
		}
		seen[key] = true
		section = ""
		switch key {
		case "id":
			spec.ID = value
		case "status":
			spec.Status = value
		case "approvedTree":
			spec.ApprovedTree = value
		case "decision":
			spec.Decision = filepath.ToSlash(value)
		case "scope", "gates":
			if value != "" {
				return Spec{}, fmt.Errorf("frontmatter key %q must use a block list", key)
			}
			section = key
		default:
			return Spec{}, fmt.Errorf("frontmatter key %q is unsupported", key)
		}
	}
	for _, required := range []string{"id", "status", "scope", "approvedTree", "gates"} {
		if !seen[required] {
			return Spec{}, fmt.Errorf("frontmatter key %q is required", required)
		}
	}
	return spec, nil
}

func verifySpec(repo string, spec Spec) ([]string, []string) {
	errs := []string{}
	warnings := []string{}
	prefix := spec.File + ": "
	directoryID := filepath.Base(filepath.Dir(filepath.FromSlash(spec.File)))
	if !validPlanID(spec.ID) {
		errs = append(errs, prefix+"id must be lowercase kebab-case")
	}
	if spec.ID != directoryID {
		errs = append(errs, prefix+fmt.Sprintf("id %q does not match directory %q", spec.ID, directoryID))
	}
	switch spec.Status {
	case StatusDraft, StatusNeedsDecision, StatusApproved, StatusImplemented, StatusArchived:
	default:
		errs = append(errs, prefix+"unsupported status: "+spec.Status)
	}
	if len(spec.Scope) == 0 {
		errs = append(errs, prefix+"scope must not be empty")
	}
	seenScope := map[string]bool{}
	for _, pattern := range spec.Scope {
		normalized, err := normalizePattern(pattern)
		if err != nil {
			errs = append(errs, prefix+"invalid scope pattern: "+err.Error())
			continue
		}
		if seenScope[normalized] {
			errs = append(errs, prefix+"duplicate scope pattern: "+normalized)
		}
		seenScope[normalized] = true
	}
	if len(spec.Gates) == 0 {
		errs = append(errs, prefix+"gates must not be empty")
	}
	seenGates := map[string]bool{}
	for _, gate := range spec.Gates {
		if gate.Profile != "smoke" && gate.Profile != "full" && gate.Profile != "release" {
			errs = append(errs, prefix+"unsupported gate profile: "+gate.Profile)
		}
		if seenGates[gate.Profile] {
			errs = append(errs, prefix+"duplicate gate profile: "+gate.Profile)
		}
		seenGates[gate.Profile] = true
	}
	planDir := filepath.Dir(filepath.Join(repo, filepath.FromSlash(spec.File)))
	optionsExists := isRegularFile(filepath.Join(planDir, "OPTIONS.md"))
	if optionsExists && spec.Decision == "" {
		errs = append(errs, prefix+"OPTIONS.md requires decision frontmatter")
	}
	if spec.Decision != "" {
		expected := filepath.ToSlash(filepath.Join(SpecsRoot, spec.ID, "DECISION.md"))
		if spec.Decision != expected {
			errs = append(errs, prefix+fmt.Sprintf("decision must be %q", expected))
		} else if !isRegularFile(filepath.Join(repo, filepath.FromSlash(spec.Decision))) {
			errs = append(errs, prefix+"decision file is missing: "+spec.Decision)
		}
	}
	if spec.Status == StatusApproved && spec.ApprovedTree == "" {
		warnings = append(warnings, prefix+"approvedTree is empty until TODO 0006 approval binding lands")
	}
	return errs, warnings
}

func trimYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func validPlanID(id string) bool {
	if id == "" || id[0] == '-' || id[len(id)-1] == '-' {
		return false
	}
	previousDash := false
	for _, char := range id {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			previousDash = false
		case char == '-' && !previousDash:
			previousDash = true
		default:
			return false
		}
	}
	return true
}

func displayPath(repo, file string) string {
	rel, err := filepath.Rel(repo, file)
	if err != nil {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
