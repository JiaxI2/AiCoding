package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/pathpolicy"
)

// BindingStatus is the deterministic projection of changes since approval.
type BindingStatus struct {
	ApprovedTree string   `json:"approvedTree"`
	CurrentTree  string   `json:"currentTree"`
	Changed      []string `json:"changed"`
	Drift        []string `json:"drift"`
	OutOfScope   []string `json:"outOfScope"`
	Exempt       []string `json:"exempt"`
	ScopeCovered bool     `json:"scopeCovered"`
}

// Approve binds one eligible plan to the caller-supplied Git tree and updates
// only its PLAN.md frontmatter. The CLI owns clean-worktree adjudication.
func Approve(repo, id, tree string) (Spec, error) {
	id = strings.TrimSpace(id)
	if !validPlanID(id) {
		return Spec{}, fmt.Errorf("plan id must be lowercase kebab-case")
	}
	tree = strings.TrimSpace(tree)
	if !validTreeOID(tree) {
		return Spec{}, fmt.Errorf("approval tree must be a Git tree object id")
	}
	file := filepath.Join(repo, filepath.FromSlash(SpecsRoot), id, "PLAN.md")
	spec, err := parseSpec(repo, file)
	if err != nil {
		return Spec{}, fmt.Errorf("%s: %w", displayPath(repo, file), err)
	}
	errs, _ := verifySpec(repo, spec)
	if len(errs) > 0 {
		return Spec{}, fmt.Errorf("plan is invalid: %s", strings.Join(errs, "; "))
	}
	if spec.Status != StatusDraft && spec.Status != StatusNeedsDecision {
		return Spec{}, fmt.Errorf("plan %s status %q cannot be approved", id, spec.Status)
	}
	if err := writeApproval(file, tree); err != nil {
		return Spec{}, err
	}
	approved, err := parseSpec(repo, file)
	if err != nil {
		return Spec{}, fmt.Errorf("read approved plan: %w", err)
	}
	return approved, nil
}

// EvaluateBinding classifies one precomputed tree diff against plan scope and
// the repository-wide exempt paths.
func EvaluateBinding(policy Policy, spec Spec, currentTree string, changed []string) (BindingStatus, error) {
	paths, err := normalizePaths(changed)
	if err != nil {
		return BindingStatus{}, err
	}
	status := BindingStatus{
		ApprovedTree: spec.ApprovedTree,
		CurrentTree:  strings.TrimSpace(currentTree),
		Changed:      paths,
		Drift:        []string{},
		OutOfScope:   []string{},
		Exempt:       []string{},
	}
	scopePatterns, err := pathpolicy.Compile(spec.Scope)
	if err != nil {
		return BindingStatus{}, err
	}
	exemptPatterns, err := pathpolicy.Compile(policy.ExemptPaths)
	if err != nil {
		return BindingStatus{}, err
	}
	covered := make([]bool, len(scopePatterns))
	for _, path := range paths {
		inScope := false
		for index, pattern := range scopePatterns {
			matched, matchErr := pathpolicy.Match(pattern, path)
			if matchErr != nil {
				return BindingStatus{}, matchErr
			}
			if matched {
				inScope = true
				covered[index] = true
			}
		}
		if inScope {
			status.Drift = append(status.Drift, path)
			continue
		}
		exempt, matchErr := matchesAny(exemptPatterns, path)
		if matchErr != nil {
			return BindingStatus{}, matchErr
		}
		if exempt {
			status.Exempt = append(status.Exempt, path)
		} else {
			status.OutOfScope = append(status.OutOfScope, path)
		}
	}
	status.ScopeCovered = len(covered) > 0
	for _, matched := range covered {
		status.ScopeCovered = status.ScopeCovered && matched
	}
	return status, nil
}

// ApprovedCoverage returns the approved plans used to cover sensitive paths
// and the sensitive paths that remain uncovered.
func ApprovedCoverage(specs []Spec, sensitive []SensitiveMatch) ([]string, []SensitiveMatch, error) {
	used := map[string]struct{}{}
	uncovered := make([]SensitiveMatch, 0)
	compiledScopes := make(map[string][]pathpolicy.Pattern, len(specs))
	for _, spec := range specs {
		if spec.Status != StatusApproved {
			continue
		}
		compiled, err := pathpolicy.Compile(spec.Scope)
		if err != nil {
			return nil, nil, err
		}
		compiledScopes[spec.ID] = compiled
	}
	for _, item := range sensitive {
		covered := false
		for _, spec := range specs {
			if spec.Status != StatusApproved {
				continue
			}
			matched, err := matchesAny(compiledScopes[spec.ID], item.Path)
			if err != nil {
				return nil, nil, err
			}
			if matched {
				covered = true
				used[spec.ID] = struct{}{}
			}
		}
		if !covered {
			uncovered = append(uncovered, item)
		}
	}
	ids := make([]string, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, uncovered, nil
}

func writeApproval(file, tree string) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	newline := "\n"
	if strings.Contains(string(raw), "\r\n") {
		newline = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	inFrontmatter := false
	frontmatterClosed := false
	statusUpdates := 0
	treeUpdates := 0
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if index == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && trimmed == "---" {
			frontmatterClosed = true
			break
		}
		if !inFrontmatter {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		switch {
		case strings.HasPrefix(trimmed, "status:"):
			lines[index] = indent + "status: " + StatusApproved
			statusUpdates++
		case strings.HasPrefix(trimmed, "approvedTree:"):
			lines[index] = indent + "approvedTree: \"" + tree + "\""
			treeUpdates++
		}
	}
	if !frontmatterClosed || statusUpdates != 1 || treeUpdates != 1 {
		return fmt.Errorf("PLAN.md approval fields are missing or duplicated")
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	return os.WriteFile(file, []byte(strings.Join(lines, newline)), info.Mode().Perm())
}
