package testengine

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

// ChangeImpactRule maps one repository path pattern to a test profile.
type ChangeImpactRule struct {
	Pattern string `json:"pattern"`
	Profile string `json:"profile"`
	Reason  string `json:"reason"`
}

// ChangeImpactPolicy is the changeVerify section of impact-policy.json.
type ChangeImpactPolicy struct {
	DefaultProfile string             `json:"defaultProfile"`
	Rules          []ChangeImpactRule `json:"rules"`
}

// ChangeImpactMatch records one matched path rule.
type ChangeImpactMatch struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
	Profile string `json:"profile"`
	Reason  string `json:"reason"`
}

type changeImpactPolicyFile struct {
	SchemaVersion int                `json:"schemaVersion"`
	RaceScope     json.RawMessage    `json:"raceScope"`
	ChangeVerify  ChangeImpactPolicy `json:"changeVerify"`
}

// LoadChangeImpactPolicy strictly loads and normalizes changeVerify rules.
func LoadChangeImpactPolicy(repo string) (ChangeImpactPolicy, error) {
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(impactPolicyPath)))
	if err != nil {
		return ChangeImpactPolicy{}, fmt.Errorf("read %s: %w", impactPolicyPath, err)
	}
	var file changeImpactPolicyFile
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return ChangeImpactPolicy{}, fmt.Errorf("parse %s: %w", impactPolicyPath, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return ChangeImpactPolicy{}, fmt.Errorf("parse %s: multiple JSON values", impactPolicyPath)
		}
		return ChangeImpactPolicy{}, fmt.Errorf("parse %s: %w", impactPolicyPath, err)
	}
	if file.SchemaVersion != 1 {
		return ChangeImpactPolicy{}, errors.New("impact policy schemaVersion must be 1")
	}
	policy := file.ChangeVerify
	defaultProfile, err := normalizeImpactProfile(policy.DefaultProfile)
	if err != nil {
		return ChangeImpactPolicy{}, fmt.Errorf("changeVerify.defaultProfile: %w", err)
	}
	policy.DefaultProfile = defaultProfile
	if len(policy.Rules) == 0 {
		return ChangeImpactPolicy{}, errors.New("impact policy requires changeVerify.rules")
	}
	seen := map[string]bool{}
	for index := range policy.Rules {
		rule := &policy.Rules[index]
		compiled, err := pathpolicy.Compile([]string{rule.Pattern})
		if err != nil {
			return ChangeImpactPolicy{}, fmt.Errorf("changeVerify.rules[%d].pattern: %w", index, err)
		}
		rule.Pattern = compiled[0].Value
		if seen[rule.Pattern] {
			return ChangeImpactPolicy{}, fmt.Errorf("changeVerify.rules[%d] duplicates pattern %q", index, rule.Pattern)
		}
		seen[rule.Pattern] = true
		rule.Profile, err = normalizeImpactProfile(rule.Profile)
		if err != nil {
			return ChangeImpactPolicy{}, fmt.Errorf("changeVerify.rules[%d].profile: %w", index, err)
		}
		rule.Reason = strings.TrimSpace(rule.Reason)
		if rule.Reason == "" {
			return ChangeImpactPolicy{}, fmt.Errorf("changeVerify.rules[%d].reason is required", index)
		}
	}
	return policy, nil
}

// SelectChangeProfile deterministically selects the highest matched profile.
func SelectChangeProfile(policy ChangeImpactPolicy, paths []string) (string, []ChangeImpactMatch, error) {
	patterns := make([]string, 0, len(policy.Rules))
	rules := make(map[string]ChangeImpactRule, len(policy.Rules))
	for _, rule := range policy.Rules {
		patterns = append(patterns, rule.Pattern)
		rules[rule.Pattern] = rule
	}
	compiled, err := pathpolicy.Compile(patterns)
	if err != nil {
		return "", nil, err
	}

	profile := ""
	matches := []ChangeImpactMatch{}
	for _, repoPath := range paths {
		pathProfile := ""
		for _, pattern := range compiled {
			matched, err := pathpolicy.Match(pattern, repoPath)
			if err != nil {
				return "", nil, err
			}
			if !matched {
				continue
			}
			rule := rules[pattern.Value]
			matches = append(matches, ChangeImpactMatch{Path: repoPath, Pattern: rule.Pattern, Profile: displayImpactProfile(rule.Profile), Reason: rule.Reason})
			if impactProfileRank(rule.Profile) > impactProfileRank(pathProfile) {
				pathProfile = rule.Profile
			}
		}
		if pathProfile == "" {
			pathProfile = policy.DefaultProfile
		}
		if impactProfileRank(pathProfile) > impactProfileRank(profile) {
			profile = pathProfile
		}
	}
	if profile == "" {
		profile = policy.DefaultProfile
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Path != matches[j].Path {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Pattern < matches[j].Pattern
	})
	return profile, matches, nil
}

func normalizeImpactProfile(profile string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case ProfileSmoke:
		return ProfileSmoke, nil
	case ProfileFull:
		return ProfileFull, nil
	case ProfileRelease:
		return ProfileRelease, nil
	default:
		return "", fmt.Errorf("unsupported profile %q", profile)
	}
}

func displayImpactProfile(profile string) string {
	switch profile {
	case ProfileSmoke:
		return "Smoke"
	case ProfileFull:
		return "Full"
	case ProfileRelease:
		return "Release"
	default:
		return profile
	}
}

func impactProfileRank(profile string) int {
	switch profile {
	case ProfileSmoke:
		return 1
	case ProfileFull:
		return 2
	case ProfileRelease:
		return 3
	default:
		return 0
	}
}
