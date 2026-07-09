package tagpolicy

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type Policy struct {
	SchemaVersion            int      `json:"schemaVersion"`
	Name                     string   `json:"name"`
	PlatformTagPattern       string   `json:"platformTagPattern"`
	KitTagPattern            string   `json:"kitTagPattern"`
	MilestoneTagPattern      string   `json:"milestoneTagPattern"`
	NonCurrentDateTagPattern string   `json:"nonCurrentDateTagPattern"`
	Rules                    []string `json:"rules"`
}

type TagRecord struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Audit struct {
	PolicyPath string         `json:"policyPath"`
	Total      int            `json:"total"`
	Counts     map[string]int `json:"counts"`
	Tags       []TagRecord    `json:"tags"`
	Warnings   []string       `json:"warnings,omitempty"`
}

func DefaultPolicy() Policy {
	return Policy{
		SchemaVersion:            1,
		Name:                     "AiCoding Tagging Policy",
		PlatformTagPattern:       `^v(?![0-9]{4}\.)[0-9]+\.[0-9]+\.[0-9]+$`,
		KitTagPattern:            `^kit/[A-Za-z0-9._-]+/v(?![0-9]{4}\.)[0-9]+\.[0-9]+\.[0-9]+$`,
		MilestoneTagPattern:      `^milestone/[0-9]{4}\.[0-9]{2}\.[0-9]{2}-[A-Za-z0-9._-]+$`,
		NonCurrentDateTagPattern: `^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}(-[A-Za-z0-9._-]+)?$`,
	}
}

func LoadPolicy(repo string) (Policy, error) {
	p := platform.RepoPath(repo, "config/tagging-policy.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := json.Unmarshal(b, &policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func AuditRepo(repo string) (Audit, []string) {
	policy, err := LoadPolicy(repo)
	if err != nil {
		return Audit{PolicyPath: "config/tagging-policy.json"}, []string{err.Error()}
	}
	out, err := gitx.Run(repo, "tag", "--list")
	if err != nil {
		return Audit{PolicyPath: "config/tagging-policy.json"}, []string{err.Error()}
	}
	tags := []string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return AuditTags(tags, policy), nil
}

func AuditTags(tags []string, policy Policy) Audit {
	audit := Audit{PolicyPath: "config/tagging-policy.json", Counts: map[string]int{}}
	sort.Strings(tags)
	for _, tag := range tags {
		category := Classify(tag, policy)
		audit.Tags = append(audit.Tags, TagRecord{Name: tag, Category: category})
		audit.Counts[category]++
		audit.Total++
		if strings.HasPrefix(category, "noncurrent") {
			audit.Warnings = append(audit.Warnings, tag+": "+category)
		}
	}
	return audit
}

func Classify(tag string, policy Policy) string {
	switch {
	case isPlatformTag(tag):
		return "platform"
	case isKitTag(tag):
		return "kit"
	case isMilestoneTag(tag):
		return "milestone"
	case isNonCurrentDateTag(tag) || match(policy.NonCurrentDateTagPattern, tag):
		return "noncurrent-date"
	case regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-[A-Za-z0-9._-]+$`).MatchString(tag):
		return "noncurrent-component"
	default:
		return "unknown"
	}
}

func isPlatformTag(tag string) bool {
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tag) {
		return false
	}
	return !regexp.MustCompile(`^v[0-9]{4}\.`).MatchString(tag)
}

func isKitTag(tag string) bool {
	parts := strings.Split(tag, "/")
	if len(parts) != 3 || parts[0] != "kit" || parts[1] == "" {
		return false
	}
	return isPlatformTag(parts[2])
}

func isMilestoneTag(tag string) bool {
	return regexp.MustCompile(`^milestone/[0-9]{4}\.[0-9]{2}\.[0-9]{2}-[A-Za-z0-9._-]+$`).MatchString(tag)
}

func isNonCurrentDateTag(tag string) bool {
	return regexp.MustCompile(`^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}(-[A-Za-z0-9._-]+)?$`).MatchString(tag)
}

func match(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	return err == nil && re.MatchString(value)
}
