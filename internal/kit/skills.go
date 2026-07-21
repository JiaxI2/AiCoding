package kit

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/runner"
)

type SkillVerifyReport struct {
	SchemaVersion int                `json:"schemaVersion"`
	Profile       string             `json:"profile"`
	OK            bool               `json:"ok"`
	Summary       SkillVerifySummary `json:"summary"`
	Kits          []SkillKitResult   `json:"kits"`
	Errors        []string           `json:"errors,omitempty"`
	Warnings      []string           `json:"warnings,omitempty"`
}

type SkillVerifySummary struct {
	Kits     int `json:"kits"`
	Skills   int `json:"skills"`
	OK       int `json:"ok"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
}

type SkillKitResult struct {
	ID       string       `json:"id"`
	OK       bool         `json:"ok"`
	Skills   []SkillEntry `json:"skills"`
	Errors   []string     `json:"errors,omitempty"`
	Warnings []string     `json:"warnings,omitempty"`
}

type SkillEntry struct {
	ID          string   `json:"id"`
	Role        string   `json:"role"`
	Kind        string   `json:"kind"`
	Path        string   `json:"path"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type rawSkills struct {
	Umbrella *rawSkill  `json:"umbrella"`
	Members  []rawSkill `json:"members"`
}
type rawSkill struct {
	ID          string   `json:"id"`
	Role        string   `json:"role"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func VerifySkills(repo string, entries []RegistryKit, profile string) SkillVerifyReport {
	inputs := make([]lifecycleInput, 0, len(entries))
	for _, entry := range entries {
		inputs = append(inputs, lifecycleInput{entry: entry})
	}
	return verifySkills(repo, inputs, profile)
}

func VerifyCatalogSkills(repo string, snapshots []ManifestSnapshot, profile string) SkillVerifyReport {
	inputs := make([]lifecycleInput, 0, len(snapshots))
	for _, snapshot := range snapshots {
		manifest, err := snapshot.Manifest()
		inputs = append(inputs, lifecycleInput{entry: snapshot.Entry(), manifest: manifest, err: err, resolved: true})
	}
	return verifySkills(repo, inputs, profile)
}

func verifySkills(repo string, inputs []lifecycleInput, profile string) SkillVerifyReport {
	profile = normalizeKitProfile(profile)
	report := SkillVerifyReport{SchemaVersion: 1, Profile: profile, OK: true}
	tasks := make([]runner.Task, 0, len(inputs))
	for _, input := range inputs {
		input := input
		tasks = append(tasks, runner.Task{
			ID:    input.entry.ID,
			Group: "skill-verify",
			Run: func(context.Context) runner.TaskResult {
				return runner.TaskResult{ID: input.entry.ID, OK: true, Data: verifyKitSkills(repo, input, profile)}
			},
		})
	}
	for _, taskResult := range runner.Run(context.Background(), tasks, runner.Options{}) {
		kitResult, ok := taskResult.Data.(SkillKitResult)
		if !ok {
			kitResult = SkillKitResult{ID: taskResult.ID, OK: false, Errors: []string{"invalid skill verify result"}}
		}
		report.Kits = append(report.Kits, kitResult)
		report.Summary.Kits++
		report.Summary.Skills += len(kitResult.Skills)
		report.Summary.Warnings += len(kitResult.Warnings)
		if kitResult.OK {
			report.Summary.OK++
		} else {
			report.Summary.Failed++
			report.OK = false
		}
		for _, err := range kitResult.Errors {
			report.Errors = append(report.Errors, kitResult.ID+": "+err)
		}
		for _, warning := range kitResult.Warnings {
			report.Warnings = append(report.Warnings, kitResult.ID+": "+warning)
		}
	}
	return report
}

func normalizeKitProfile(profile string) string {
	normalized := strings.ToLower(strings.TrimSpace(profile))
	switch normalized {
	case "", "smoke":
		return "Smoke"
	case "full":
		return "Full"
	case "release":
		return "Release"
	default:
		return strings.ToUpper(normalized[:1]) + normalized[1:]
	}
}

func verifyKitSkills(repo string, input lifecycleInput, profile string) SkillKitResult {
	entry := input.entry
	result := SkillKitResult{ID: entry.ID, OK: true}
	manifest, err := lifecycleManifest(repo, input)
	if err != nil {
		result.OK = false
		result.Errors = append(result.Errors, "cannot load manifest: "+err.Error())
		return result
	}
	skills, errs := parseSkillEntries(manifest)
	result.Skills = skills
	result.Errors = append(result.Errors, errs...)
	seen := map[string]bool{}
	umbrella := 0
	for _, skill := range skills {
		if skill.ID == "" {
			result.Errors = append(result.Errors, "empty skill id")
			continue
		}
		if seen[skill.ID] {
			result.Errors = append(result.Errors, "duplicate skill id: "+skill.ID)
		}
		seen[skill.ID] = true
		if skill.Kind == "umbrella" {
			umbrella++
			if skill.Role != "router" && skill.Role != "umbrella" {
				result.Errors = append(result.Errors, "invalid umbrella role: "+skill.ID+" -> "+skill.Role)
			}
		}
		if skill.Kind == "member" && skill.Role != "subskill" {
			result.Errors = append(result.Errors, "invalid member role: "+skill.ID+" -> "+skill.Role)
		}
		if skill.Path == "" {
			result.Errors = append(result.Errors, "missing skill path: "+skill.ID)
			continue
		}
		full := platform.RepoPath(repo, skill.Path)
		document, ferrs := readSkillDocument(full)
		front := document.Frontmatter
		for _, e := range ferrs {
			result.Errors = append(result.Errors, skill.ID+": "+e)
		}
		if name := front["name"]; name == "" {
			result.Errors = append(result.Errors, skill.ID+": missing frontmatter.name")
		}
		if desc := front["description"]; desc == "" {
			result.Errors = append(result.Errors, skill.ID+": missing frontmatter.description")
		}
		if profile == "Full" || profile == "Release" {
			if len(document.Sections) == 0 {
				result.Warnings = append(result.Warnings, skill.ID+": SKILL.md has no section heading")
			}
			dir := filepath.Dir(full)
			if platform.IsDir(filepath.Join(dir, "references")) {
				result.Warnings = append(result.Warnings, skill.ID+": references directory present; content not deeply validated by Go")
			}
		}
	}
	if umbrella > 1 {
		result.Errors = append(result.Errors, "more than one umbrella skill")
	}
	result.OK = len(result.Errors) == 0
	return result
}

func parseSkillEntries(manifest Manifest) ([]SkillEntry, []string) {
	if len(manifest.Skills) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(manifest.Skills)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var raw rawSkills
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, []string{err.Error()}
	}
	entries := []SkillEntry{}
	if raw.Umbrella != nil {
		entries = append(entries, skillEntry(*raw.Umbrella, "umbrella"))
	}
	for _, member := range raw.Members {
		entries = append(entries, skillEntry(member, "member"))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries, nil
}

func skillEntry(raw rawSkill, kind string) SkillEntry {
	return SkillEntry{ID: raw.ID, Role: raw.Role, Kind: kind, Path: filepath.ToSlash(raw.Path), Description: raw.Description, Tags: raw.Tags}
}

type skillDocument struct {
	Frontmatter map[string]string
	Sections    []string
}

func readSkillDocument(path string) (skillDocument, []string) {
	file, err := os.Open(path)
	if err != nil {
		return skillDocument{Frontmatter: map[string]string{}}, []string{"missing SKILL.md"}
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return skillDocument{Frontmatter: map[string]string{}}, []string{"missing frontmatter"}
	}
	document := skillDocument{Frontmatter: map[string]string{}, Sections: []string{}}
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if !closed {
			if strings.TrimSpace(line) == "---" {
				closed = true
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if key != "" {
				document.Frontmatter[key] = value
			}
			continue
		}
		if strings.HasPrefix(line, "## ") {
			section := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if section != "" {
				document.Sections = append(document.Sections, section)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return document, []string{err.Error()}
	}
	if !closed {
		return document, []string{"unterminated frontmatter"}
	}
	return document, nil
}
