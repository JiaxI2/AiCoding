// Package pathpolicy compiles and matches deterministic repository-relative
// path patterns shared by policy consumers.
package pathpolicy

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Pattern is one normalized, compiled policy pattern.
type Pattern struct {
	Value string
	match *regexp.Regexp
}

// Compile validates, de-duplicates, and sorts policy patterns before compiling
// their frozen *, **, and ? glob dialect.
func Compile(patterns []string) ([]Pattern, error) {
	unique := make(map[string]struct{}, len(patterns))
	for _, raw := range patterns {
		pattern, err := normalizePattern(raw)
		if err != nil {
			return nil, err
		}
		unique[pattern] = struct{}{}
	}

	values := make([]string, 0, len(unique))
	for pattern := range unique {
		values = append(values, pattern)
	}
	sort.Strings(values)

	compiled := make([]Pattern, 0, len(values))
	for _, pattern := range values {
		matcher, err := regexp.Compile(globRegex(pattern))
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, Pattern{Value: pattern, match: matcher})
	}
	return compiled, nil
}

// Match validates path and reports whether one compiled pattern matches it.
func Match(compiled Pattern, path string) (bool, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if err := validateRelativePath(path); err != nil {
		return false, err
	}
	if compiled.match == nil || compiled.Value == "" {
		return false, errors.New("compiled path policy pattern is required")
	}
	return compiled.match.MatchString(path), nil
}

// Validate checks the path policy dialect without retaining compiled state.
func Validate(patterns []string) error {
	_, err := Compile(patterns)
	return err
}

func normalizePattern(raw string) (string, error) {
	pattern := filepath.ToSlash(strings.TrimSpace(raw))
	if err := validateRelativePath(pattern); err != nil {
		return "", err
	}
	if strings.ContainsAny(pattern, "[]{}") {
		return "", errors.New("pattern supports only *, **, and ? wildcards")
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
