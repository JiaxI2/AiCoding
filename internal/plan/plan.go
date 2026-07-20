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
	"regexp"
	"sort"
	"strings"
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
		PolicyPath: PolicyPath,
		Paths:      normalizedPaths,
		Sensitive:  []SensitiveMatch{},
		Exempt:     []string{},
	}
	for _, path := range normalizedPaths {
		exempt, err := matchesAny(policy.ExemptPaths, path)
		if err != nil {
			return Check{}, err
		}
		if exempt {
			check.Exempt = append(check.Exempt, path)
			continue
		}
		for _, rule := range policy.SensitivePaths {
			matched, err := matchPattern(rule.Pattern, path)
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
		pattern, err := normalizePattern(rule.Pattern)
		if err != nil {
			return Policy{}, fmt.Errorf("sensitivePaths[%d]: %w", index, err)
		}
		reason := strings.TrimSpace(rule.Reason)
		if reason == "" {
			return Policy{}, fmt.Errorf("sensitivePaths[%d].reason is required", index)
		}
		if previous, exists := rules[pattern]; exists && previous != reason {
			return Policy{}, fmt.Errorf("sensitive pattern %q has conflicting reasons", pattern)
		}
		rules[pattern] = reason
	}
	policy.SensitivePaths = make([]SensitiveRule, 0, len(rules))
	for pattern, reason := range rules {
		policy.SensitivePaths = append(policy.SensitivePaths, SensitiveRule{Pattern: pattern, Reason: reason})
	}
	sort.Slice(policy.SensitivePaths, func(i, j int) bool {
		return policy.SensitivePaths[i].Pattern < policy.SensitivePaths[j].Pattern
	})

	exempt := make(map[string]struct{}, len(policy.ExemptPaths))
	for index, raw := range policy.ExemptPaths {
		pattern, err := normalizePattern(raw)
		if err != nil {
			return Policy{}, fmt.Errorf("exemptPaths[%d]: %w", index, err)
		}
		exempt[pattern] = struct{}{}
	}
	policy.ExemptPaths = make([]string, 0, len(exempt))
	for pattern := range exempt {
		policy.ExemptPaths = append(policy.ExemptPaths, pattern)
	}
	sort.Strings(policy.ExemptPaths)
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

func normalizePattern(raw string) (string, error) {
	pattern := filepath.ToSlash(strings.TrimSpace(raw))
	if err := validateRelativePath(pattern); err != nil {
		return "", err
	}
	if strings.ContainsAny(pattern, "[]{}") {
		return "", errors.New("pattern supports only *, **, and ? wildcards")
	}
	if _, err := regexp.Compile(globRegex(pattern)); err != nil {
		return "", fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	return pattern, nil
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

func matchesAny(patterns []string, path string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := matchPattern(pattern, path)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func matchPattern(pattern, path string) (bool, error) {
	compiled, err := regexp.Compile(globRegex(pattern))
	if err != nil {
		return false, fmt.Errorf("compile plan pattern %q: %w", pattern, err)
	}
	return compiled.MatchString(path), nil
}

func globRegex(pattern string) string {
	var out strings.Builder
	out.WriteByte('^')
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '*':
			if index+1 < len(pattern) && pattern[index+1] == '*' {
				out.WriteString(".*")
				index++
			} else {
				out.WriteString("[^/]*")
			}
		case '?':
			out.WriteString("[^/]")
		default:
			out.WriteString(regexp.QuoteMeta(string(pattern[index])))
		}
	}
	out.WriteByte('$')
	return out.String()
}
