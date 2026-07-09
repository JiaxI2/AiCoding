package cstyle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const CommentTemplatesPath = "config/skills/c99-standard-c/templates/comment-templates.json"

type TemplateConfig struct {
	SchemaVersion int               `json:"schemaVersion"`
	Description   string            `json:"description,omitempty"`
	Templates     []CommentTemplate `json:"templates"`
}

type CommentTemplate struct {
	ID          string                      `json:"id"`
	Title       string                      `json:"title"`
	Description string                      `json:"description"`
	Language    string                      `json:"language"`
	Kind        string                      `json:"kind"`
	Body        []string                    `json:"body"`
	Variables   map[string]TemplateVariable `json:"variables"`
}

type TemplateVariable struct {
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type TemplateSummary struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	BodyLine int    `json:"bodyLine"`
}

type TemplateValidation struct {
	Path      string            `json:"path"`
	Valid     bool              `json:"valid"`
	Count     int               `json:"count"`
	Templates []TemplateSummary `json:"templates"`
	Errors    []string          `json:"errors,omitempty"`
}

func ValidateTemplates(repoRoot string) (TemplateValidation, error) {
	return ValidateCommentTemplates(repoRoot, DefaultSkillID)
}

func ValidateCommentTemplates(repoRoot string, skillID string) (TemplateValidation, error) {
	root, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return TemplateValidation{Path: CommentTemplatesPath, Valid: false, Errors: []string{err.Error()}}, err
	}

	cfg, err := LoadSkillConfig(root, skillID)
	if err != nil {
		res := TemplateValidation{Path: skillConfigPath(skillID), Valid: false, Errors: []string{err.Error()}}
		return res, err
	}

	fullPath, err := ResolveCommentTemplatesPath(root, cfg)
	if err != nil {
		res := TemplateValidation{Path: cfg.CommentTemplates, Valid: false, Errors: []string{err.Error()}}
		return res, err
	}
	pathLabel := relativeRepoPath(root, fullPath)

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		res := TemplateValidation{Path: pathLabel, Valid: false, Errors: []string{err.Error()}}
		return res, err
	}

	var cfgTemplates TemplateConfig
	if err := json.Unmarshal(raw, &cfgTemplates); err != nil {
		res := TemplateValidation{Path: pathLabel, Valid: false, Errors: []string{err.Error()}}
		return res, err
	}

	res := TemplateValidation{Path: pathLabel}
	seen := map[string]int{}

	if len(cfgTemplates.Templates) == 0 {
		res.Errors = append(res.Errors, "templates must not be empty")
	}

	for i, tmpl := range cfgTemplates.Templates {
		idx := i + 1
		id := strings.TrimSpace(tmpl.ID)
		kind := strings.TrimSpace(tmpl.Kind)
		language := strings.TrimSpace(tmpl.Language)

		if id == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("template[%d] id is required", idx))
		} else if prev, ok := seen[id]; ok {
			res.Errors = append(res.Errors, fmt.Sprintf("template[%d] id duplicates template[%d]: %s", idx, prev, id))
		} else {
			seen[id] = idx
		}

		if len(tmpl.Body) == 0 {
			res.Errors = append(res.Errors, fmt.Sprintf("template[%d] body must not be empty", idx))
		}
		if language != "c" {
			res.Errors = append(res.Errors, fmt.Sprintf("template[%d] language must be c: %s", idx, language))
		}
		if kind == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("template[%d] kind is required", idx))
		}

		res.Templates = append(res.Templates, TemplateSummary{
			ID:       id,
			Title:    tmpl.Title,
			Kind:     kind,
			BodyLine: len(tmpl.Body),
		})
	}

	sort.Slice(res.Templates, func(i, j int) bool {
		return res.Templates[i].ID < res.Templates[j].ID
	})
	res.Count = len(res.Templates)
	res.Valid = len(res.Errors) == 0

	if !res.Valid {
		return res, errors.New(strings.Join(res.Errors, "; "))
	}
	return res, nil
}
