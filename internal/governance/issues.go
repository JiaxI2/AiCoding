package governance

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type issueLabelManifest struct {
	SchemaVersion int                    `json:"schema_version"`
	Labels        []issueLabelDefinition `json:"labels"`
}

type issueLabelDefinition struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func lintIssueGovernance(repo, governanceConfig string, fail func(string)) {
	for _, required := range []string{
		".github/ISSUE_TEMPLATE/bug.yml",
		".github/ISSUE_TEMPLATE/feature.yml",
		".github/ISSUE_TEMPLATE/governance.yml",
		".github/ISSUE_TEMPLATE/config.yml",
		".github/issue-labels.json",
		".github/workflows/issue-governance.yml",
	} {
		if !platform.IsFile(platform.RepoPath(repo, required)) {
			fail("required Issue governance file missing: " + required)
		}
	}

	standard := tomlSection(governanceConfig, "governance_standard")
	for _, item := range []struct {
		needle string
		msg    string
	}{
		{`id = "aicoding-git-governance"`, "repository-governance.toml must declare the canonical governance standard id"},
		{`version = "2026.07.16"`, "repository-governance.toml must use governance standard version 2026.07.16"},
		{`source_url = "https://github.com/JiaxI2/Codex-Skills/blob/main/platform/aicoding-git-governance/references/aicoding-git-governance-standard.md"`, "repository-governance.toml must declare the canonical governance standard URL"},
		{`sync_policy = "track-canonical-url"`, "repository-governance.toml must track the canonical governance URL"},
	} {
		mustContain(standard, item.needle, item.msg, fail)
	}

	issues := tomlSection(governanceConfig, "issues")
	for _, item := range []struct {
		needle string
		msg    string
	}{
		{`enabled = true`, "issues.enabled must be true"},
		{`profile = "managed-lifecycle"`, "issues.profile must be managed-lifecycle"},
		{`templates_directory = ".github/ISSUE_TEMPLATE"`, "issues.templates_directory must point to .github/ISSUE_TEMPLATE"},
		{`label_manifest = ".github/issue-labels.json"`, "issues.label_manifest must point to .github/issue-labels.json"},
		{`workflow = ".github/workflows/issue-governance.yml"`, "issues.workflow must point to the lifecycle workflow"},
		{`allow_blank = false`, "issues.allow_blank must be false"},
		{`required_label_axes = ["type", "area", "priority", "status"]`, "issues.required_label_axes must include type, area, priority, and status"},
		{`closure_requires_resolution = true`, "Issue closure must require a resolution"},
		{`closure_requires_summary = true`, "Issue closure must require an outcome summary"},
		{`auto_close_stale = false`, "Issue governance must not auto-close stale Issues"},
	} {
		mustContain(issues, item.needle, item.msg, fail)
	}

	checkIssueForm(repo, ".github/ISSUE_TEMPLATE/bug.yml", "type:bug", []string{"existing", "current_behavior", "expected_behavior", "reproduction", "impact", "environment", "done_condition"}, fail)
	checkIssueForm(repo, ".github/ISSUE_TEMPLATE/feature.yml", "type:feature", []string{"existing", "problem", "outcome", "scope", "acceptance", "alternatives", "traceability"}, fail)
	checkIssueForm(repo, ".github/ISSUE_TEMPLATE/governance.yml", "type:governance", []string{"existing", "gap", "proposed_rule", "lifecycle_impact", "verification", "compatibility", "rollback"}, fail)

	chooser, _ := platform.ReadText(platform.RepoPath(repo, ".github/ISSUE_TEMPLATE/config.yml"))
	mustContain(chooser, "blank_issues_enabled: false", "Issue template chooser must disable contributor blank Issues", fail)

	manifestText, err := platform.ReadText(platform.RepoPath(repo, ".github/issue-labels.json"))
	if err != nil {
		return
	}
	var manifest issueLabelManifest
	if err := json.Unmarshal([]byte(manifestText), &manifest); err != nil {
		fail("cannot parse .github/issue-labels.json: " + err.Error())
		return
	}
	if manifest.SchemaVersion != 1 {
		fail("Issue label manifest schema_version must be 1")
	}
	validateIssueLabels(manifest.Labels, fail)

	workflow, _ := platform.ReadText(platform.RepoPath(repo, ".github/workflows/issue-governance.yml"))
	for _, token := range []string{"actions/github-script@v9", "issues: write", "opened", "reopened", "labeled", "closed", ".github/issue-labels.json"} {
		mustContain(workflow, token, "Issue governance workflow missing required token: "+token, fail)
	}
	if strings.Contains(workflow, "{{ISSUE_LABEL_MANIFEST}}") {
		fail("Issue governance workflow contains unresolved ISSUE_LABEL_MANIFEST placeholder")
	}
}

func checkIssueForm(repo, relativePath, typeLabel string, fieldIDs []string, fail func(string)) {
	content, err := platform.ReadText(platform.RepoPath(repo, relativePath))
	if err != nil {
		return
	}
	for _, token := range []string{"name:", "description:", "body:", `"` + typeLabel + `"`, `"status:needs-triage"`} {
		mustContain(content, token, relativePath+" missing required token: "+token, fail)
	}
	for _, id := range fieldIDs {
		matched, _ := regexp.MatchString(`(?m)^\s*(?:-\s*)?id:\s*`+regexp.QuoteMeta(id)+`\s*$`, content)
		if !matched {
			fail(relativePath + " missing required field id: " + id)
		}
	}
}

func validateIssueLabels(labels []issueLabelDefinition, fail func(string)) {
	required := []string{
		"type:bug", "type:feature", "type:governance",
		"priority:p0", "priority:p1", "priority:p2", "priority:p3",
		"status:needs-triage", "status:needs-info", "status:ready", "status:in-progress", "status:blocked",
		"resolution:completed", "resolution:duplicate", "resolution:not-planned", "resolution:invalid",
	}
	seen := map[string]bool{}
	hasArea := false
	colorPattern := regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
	for _, label := range labels {
		if label.Name == "" {
			fail("Issue label name must not be empty")
			continue
		}
		if seen[label.Name] {
			fail("Issue label manifest contains duplicate name: " + label.Name)
		}
		seen[label.Name] = true
		hasArea = hasArea || strings.HasPrefix(label.Name, "area:")
		if !colorPattern.MatchString(label.Color) {
			fail("Issue label must use a six-digit hex color without #: " + label.Name)
		}
		if strings.TrimSpace(label.Description) == "" {
			fail("Issue label must have a description: " + label.Name)
		}
	}
	for _, name := range required {
		if !seen[name] {
			fail("Issue label manifest missing required label: " + name)
		}
	}
	if !hasArea {
		fail("Issue label manifest must define at least one area:* label")
	}
}

func tomlSection(content, name string) string {
	pattern := `(?ms)^\[` + regexp.QuoteMeta(name) + `\]\s*(.*?)(?:^\[|\z)`
	match := regexp.MustCompile(pattern).FindStringSubmatch(content)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}
