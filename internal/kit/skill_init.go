package kit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	skilltemplates "github.com/JiaxI2/AiCoding/config/templates/skill"
)

type SkillInitOptions struct {
	Out    string
	DryRun bool
}

type SkillInitReport struct {
	SchemaVersion int        `json:"schemaVersion"`
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	DryRun        bool       `json:"dryRun"`
	OutputMode    string     `json:"outputMode"`
	OutputRoot    string     `json:"outputRoot,omitempty"`
	Content       string     `json:"content,omitempty"`
	OK            bool       `json:"ok"`
	Files         []InitFile `json:"files"`
	Errors        []string   `json:"errors,omitempty"`
}

type skillInitTemplateData struct {
	ID   string
	Name string
}

func InitSkill(repo, id string, opts SkillInitOptions) (SkillInitReport, error) {
	id = strings.TrimSpace(id)
	report := SkillInitReport{
		SchemaVersion: 1, ID: id, DryRun: opts.DryRun,
		OutputMode: "preview", Files: []InitFile{},
	}
	if !kitInitIDPattern.MatchString(id) || len(id) > 64 {
		return failSkillInit(report, fmt.Errorf("skill id must be lowercase hyphen-case with 2-64 characters: %s", id))
	}
	report.Name = kitInitDisplayName(id)
	content, err := renderSkillInitTemplate(skillInitTemplateData{ID: id, Name: report.Name})
	if err != nil {
		return failSkillInit(report, err)
	}
	if errs := validateSkillInitContent(content, id); len(errs) != 0 {
		return failSkillInit(report, fmt.Errorf("generated Skill scaffold is invalid: %s", strings.Join(errs, "; ")))
	}

	out := strings.TrimSpace(opts.Out)
	if out == "" {
		report.Content = string(content)
		report.Files = append(report.Files, InitFile{
			Path: "stdout:SKILL.md", Action: "preview",
			Digest: initContentDigest(content), Bytes: len(content),
		})
		report.OK = true
		return report, nil
	}

	target, outputRoot, err := resolveSkillInitTarget(repo, out)
	if err != nil {
		return failSkillInit(report, err)
	}
	report.OutputMode = "directory"
	report.OutputRoot = outputRoot
	report.Files = append(report.Files, InitFile{
		Path: filepath.ToSlash(target), Action: "planned-create",
		Digest: initContentDigest(content), Bytes: len(content),
	})
	if _, statErr := os.Lstat(target); statErr == nil {
		return failSkillInit(report, fmt.Errorf("target already exists and will not be overwritten: %s", target))
	} else if !os.IsNotExist(statErr) {
		return failSkillInit(report, fmt.Errorf("inspect %s: %w", target, statErr))
	}
	if opts.DryRun {
		report.OK = true
		return report, nil
	}
	if err := writeKitInitNewFile(target, content); err != nil {
		return failSkillInit(report, fmt.Errorf("create %s: %w", target, err))
	}
	report.Files[0].Action = "created"
	report.OK = true
	return report, nil
}

func failSkillInit(report SkillInitReport, err error) (SkillInitReport, error) {
	report.OK = false
	report.Errors = append(report.Errors, err.Error())
	return report, err
}

func renderSkillInitTemplate(data skillInitTemplateData) ([]byte, error) {
	raw, err := skilltemplates.Files.ReadFile("SKILL.tmpl")
	if err != nil {
		return nil, fmt.Errorf("read Skill init template: %w", err)
	}
	tmpl, err := template.New("SKILL.tmpl").Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse Skill init template: %w", err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render Skill init template: %w", err)
	}
	return rendered.Bytes(), nil
}

func validateSkillInitContent(content []byte, expectedID string) []string {
	document, errs := parseSkillDocument(bytes.NewReader(content))
	if document.Frontmatter["name"] != expectedID {
		errs = append(errs, "frontmatter.name does not match the requested id")
	}
	if strings.TrimSpace(document.Frontmatter["description"]) == "" {
		errs = append(errs, "frontmatter.description is required")
	}
	sections := map[string]bool{}
	for _, section := range document.Sections {
		sections[strings.ToLower(section)] = true
	}
	for _, required := range []string{"Skill Type", "When to use", "Workflow", "Workflow Contract", "Verification", "Constraints", "Gate Rules", "Human Confirmation"} {
		if !sections[strings.ToLower(required)] {
			errs = append(errs, "missing section: "+required)
		}
	}
	return errs
}

func resolveSkillInitTarget(repo, out string) (string, string, error) {
	outputRoot := filepath.FromSlash(out)
	if !filepath.IsAbs(outputRoot) {
		outputRoot = filepath.Join(repo, outputRoot)
	}
	outputRoot, err := filepath.Abs(outputRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve Skill output directory: %w", err)
	}
	target := filepath.Join(outputRoot, "SKILL.md")
	canonicalTarget, err := canonicalProspectivePath(target)
	if err != nil {
		return "", "", fmt.Errorf("canonicalize Skill output: %w", err)
	}
	canonicalRepo, err := canonicalProspectivePath(repo)
	if err != nil {
		return "", "", fmt.Errorf("canonicalize repository root: %w", err)
	}
	canonicalSubmodule, err := canonicalProspectivePath(filepath.Join(repo, "CodingKit", "agents", "skills"))
	if err != nil {
		return "", "", fmt.Errorf("canonicalize read-only Skill submodule: %w", err)
	}
	if pathWithin(canonicalSubmodule, canonicalTarget) {
		return "", "", fmt.Errorf("skill output is inside read-only CodingKit/agents/skills; create it in a writable Codex-Skills worktree")
	}
	if pathWithin(canonicalRepo, canonicalTarget) {
		return "", "", fmt.Errorf("AiCoding does not own Skill source; choose an output directory outside the AiCoding repository")
	}
	return target, outputRoot, nil
}

func canonicalProspectivePath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	current := filepath.Clean(absolute)
	tail := []string{}
	for {
		if _, statErr := os.Lstat(current); statErr == nil {
			resolved, resolveErr := filepath.EvalSymlinks(current)
			if resolveErr != nil {
				return "", resolveErr
			}
			for index := len(tail) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, tail[index])
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing ancestor for %s", absolute)
		}
		tail = append(tail, filepath.Base(current))
		current = parent
	}
}

func pathWithin(root, target string) bool {
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
		target = strings.ToLower(target)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
