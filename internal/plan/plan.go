// Package plan decides whether repository paths require Plan Mode artifacts.
package plan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/pathpolicy"
)

const PolicyPath = "config/plan-policy.json"

type SensitiveRule struct {
	Pattern string `json:"pattern"`
	Reason  string `json:"reason"`
}

type Policy struct {
	SchemaVersion  int             `json:"schemaVersion"`
	SensitivePaths []SensitiveRule `json:"sensitivePaths"`
	ExemptPaths    []string        `json:"exemptPaths"`
}

type SensitiveMatch struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
	Reason  string `json:"reason"`
}

type Check struct {
	PolicyPath     string           `json:"policyPath"`
	Paths          []string         `json:"paths"`
	Sensitive      []SensitiveMatch `json:"sensitive"`
	Exempt         []string         `json:"exempt"`
	ApprovedPlans  []string         `json:"approvedPlans"`
	Uncovered      []SensitiveMatch `json:"uncovered"`
	RequiredAction string           `json:"requiredAction,omitempty"`
}

// LoadPolicy strictly decodes and validates the plan-policy schema, then
// returns a deterministic pattern order with exact duplicates removed.
func LoadPolicy(repo string) (Policy, error) {
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(PolicyPath)))
	if err != nil {
		return Policy{}, fmt.Errorf("read %s: %w", PolicyPath, err)
	}
	var policy Policy
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("parse %s: %w", PolicyPath, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Policy{}, fmt.Errorf("parse %s: multiple JSON values", PolicyPath)
		}
		return Policy{}, fmt.Errorf("parse %s: %w", PolicyPath, err)
	}
	return normalizePolicy(policy)
}

// CheckPaths is pure: the same policy and repository-relative paths produce
// the same sorted sensitive and exempt projections.
func CheckPaths(policy Policy, paths []string) (Check, error) {
	policy, err := normalizePolicy(policy)
	if err != nil {
		return Check{}, err
	}
	normalizedPaths, err := normalizePaths(paths)
	if err != nil {
		return Check{}, err
	}
	check := Check{
		PolicyPath:    PolicyPath,
		Paths:         normalizedPaths,
		Sensitive:     []SensitiveMatch{},
		Exempt:        []string{},
		ApprovedPlans: []string{},
		Uncovered:     []SensitiveMatch{},
	}
	exemptPatterns, err := pathpolicy.Compile(policy.ExemptPaths)
	if err != nil {
		return Check{}, err
	}
	sensitiveValues := make([]string, 0, len(policy.SensitivePaths))
	for _, rule := range policy.SensitivePaths {
		sensitiveValues = append(sensitiveValues, rule.Pattern)
	}
	sensitivePatterns, err := pathpolicy.Compile(sensitiveValues)
	if err != nil {
		return Check{}, err
	}
	for _, path := range normalizedPaths {
		exempt, err := matchesAny(exemptPatterns, path)
		if err != nil {
			return Check{}, err
		}
		if exempt {
			check.Exempt = append(check.Exempt, path)
			continue
		}
		for index, rule := range policy.SensitivePaths {
			matched, err := pathpolicy.Match(sensitivePatterns[index], path)
			if err != nil {
				return Check{}, err
			}
			if matched {
				check.Sensitive = append(check.Sensitive, SensitiveMatch{Path: path, Pattern: rule.Pattern, Reason: rule.Reason})
				break
			}
		}
	}
	return check, nil
}

func normalizePolicy(policy Policy) (Policy, error) {
	if policy.SchemaVersion != 1 {
		return Policy{}, errors.New("plan policy schemaVersion must be 1")
	}
	if len(policy.SensitivePaths) == 0 {
		return Policy{}, errors.New("plan policy requires sensitivePaths")
	}
	rules := make(map[string]string, len(policy.SensitivePaths))
	for index, rule := range policy.SensitivePaths {
		compiled, err := pathpolicy.Compile([]string{rule.Pattern})
		if err != nil {
			return Policy{}, fmt.Errorf("sensitivePaths[%d]: %w", index, err)
		}
		pattern := compiled[0].Value
		reason := strings.TrimSpace(rule.Reason)
		if reason == "" {
			return Policy{}, fmt.Errorf("sensitivePaths[%d].reason is required", index)
		}
		if previous, exists := rules[pattern]; exists && previous != reason {
			return Policy{}, fmt.Errorf("sensitive pattern %q has conflicting reasons", pattern)
		}
		rules[pattern] = reason
	}
	patterns := make([]string, 0, len(rules))
	for pattern := range rules {
		patterns = append(patterns, pattern)
	}
	compiledRules, err := pathpolicy.Compile(patterns)
	if err != nil {
		return Policy{}, err
	}
	policy.SensitivePaths = make([]SensitiveRule, 0, len(compiledRules))
	for _, pattern := range compiledRules {
		policy.SensitivePaths = append(policy.SensitivePaths, SensitiveRule{Pattern: pattern.Value, Reason: rules[pattern.Value]})
	}

	exempt := make(map[string]struct{}, len(policy.ExemptPaths))
	for index, raw := range policy.ExemptPaths {
		compiled, err := pathpolicy.Compile([]string{raw})
		if err != nil {
			return Policy{}, fmt.Errorf("exemptPaths[%d]: %w", index, err)
		}
		pattern := compiled[0].Value
		exempt[pattern] = struct{}{}
	}
	exemptValues := make([]string, 0, len(exempt))
	for pattern := range exempt {
		exemptValues = append(exemptValues, pattern)
	}
	compiledExempt, err := pathpolicy.Compile(exemptValues)
	if err != nil {
		return Policy{}, err
	}
	policy.ExemptPaths = make([]string, 0, len(compiledExempt))
	for _, pattern := range compiledExempt {
		policy.ExemptPaths = append(policy.ExemptPaths, pattern.Value)
	}
	return policy, nil
}

func normalizePaths(paths []string) ([]string, error) {
	set := make(map[string]struct{}, len(paths))
	for index, raw := range paths {
		path := filepath.ToSlash(strings.TrimSpace(raw))
		if err := validateRelativePath(path); err != nil {
			return nil, fmt.Errorf("paths[%d]: %w", index, err)
		}
		set[path] = struct{}{}
	}
	normalized := make([]string, 0, len(set))
	for path := range set {
		normalized = append(normalized, path)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func validateRelativePath(value string) error {
	if value == "" {
		return errors.New("repository-relative path is required")
	}
	if strings.HasPrefix(value, "/") || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return fmt.Errorf("path %q must be repository-relative and use forward slashes", value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("path %q contains an invalid segment", value)
		}
	}
	return nil
}

func matchesAny(patterns []pathpolicy.Pattern, path string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := pathpolicy.Match(pattern, path)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
