package cstyle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultSkillID      = "c99-standard-c"
	skillConfigRootPath = "config/skills"
)

type SkillConfig struct {
	SchemaVersion       int                    `json:"schemaVersion"`
	ID                  string                 `json:"id"`
	Title               string                 `json:"title"`
	Language            string                 `json:"language"`
	Standard            string                 `json:"standard"`
	Formatter           SkillFormatterConfig   `json:"formatter"`
	CommentTemplates    string                 `json:"commentTemplates"`
	Rules               string                 `json:"rules"`
	ExcludedDirectories []string               `json:"excludedDirectories"`
	Extra               map[string]interface{} `json:"-"`
}

type SkillFormatterConfig struct {
	ID     string `json:"id"`
	Config string `json:"config"`
}

type SkillStatusReport struct {
	SkillID                string     `json:"skillId"`
	SkillConfig            string     `json:"skillConfig"`
	Language               string     `json:"language"`
	Standard               string     `json:"standard"`
	Formatter              string     `json:"formatter"`
	FormatterConfig        string     `json:"formatterConfig"`
	FormatterConfigExists  bool       `json:"formatterConfigExists"`
	CommentTemplates       string     `json:"commentTemplates"`
	CommentTemplatesExists bool       `json:"commentTemplatesExists"`
	Rules                  string     `json:"rules"`
	RulesExists            bool       `json:"rulesExists"`
	ExcludedDirectories    []string   `json:"excludedDirectories"`
	ClangFormat            ToolStatus `json:"clangFormat"`
}

func LoadSkillConfig(repoRoot, skillID string) (SkillConfig, error) {
	root, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return SkillConfig{}, err
	}

	id := normalizeSkillID(skillID)
	path := filepath.Join(root, filepath.FromSlash(skillConfigPath(id)))
	raw, err := os.ReadFile(path)
	if err != nil {
		return SkillConfig{}, err
	}

	var cfg SkillConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return SkillConfig{}, err
	}
	if strings.TrimSpace(cfg.ID) == "" {
		return SkillConfig{}, fmt.Errorf("skill config id is required: %s", skillConfigPath(id))
	}
	if cfg.ID != id {
		return SkillConfig{}, fmt.Errorf("skill config id mismatch: want %s, got %s", id, cfg.ID)
	}
	if cfg.Language != "c" {
		return SkillConfig{}, fmt.Errorf("skill %s language must be c: %s", id, cfg.Language)
	}
	if strings.TrimSpace(cfg.Formatter.ID) == "" {
		return SkillConfig{}, fmt.Errorf("skill %s formatter id is required", id)
	}
	if strings.TrimSpace(cfg.Formatter.Config) == "" {
		return SkillConfig{}, fmt.Errorf("skill %s formatter config is required", id)
	}
	if strings.TrimSpace(cfg.CommentTemplates) == "" {
		return SkillConfig{}, fmt.Errorf("skill %s comment templates path is required", id)
	}
	if strings.TrimSpace(cfg.Rules) == "" {
		return SkillConfig{}, fmt.Errorf("skill %s rules path is required", id)
	}
	return cfg, nil
}

func ResolveFormatterConfig(repoRoot string, cfg SkillConfig) (string, error) {
	return resolveSkillRelativePath(repoRoot, cfg.ID, cfg.Formatter.Config)
}

func ResolveCommentTemplatesPath(repoRoot string, cfg SkillConfig) (string, error) {
	return resolveSkillRelativePath(repoRoot, cfg.ID, cfg.CommentTemplates)
}

func ResolveRulesPath(repoRoot string, cfg SkillConfig) (string, error) {
	return resolveSkillRelativePath(repoRoot, cfg.ID, cfg.Rules)
}

func SkillStatus(repoRoot, skillID string) (SkillStatusReport, error) {
	root, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return SkillStatusReport{}, err
	}

	cfg, err := LoadSkillConfig(root, skillID)
	if err != nil {
		return SkillStatusReport{}, err
	}

	formatterConfig, formatterErr := ResolveFormatterConfig(root, cfg)
	templatesPath, templatesErr := ResolveCommentTemplatesPath(root, cfg)
	rulesPath, rulesErr := ResolveRulesPath(root, cfg)

	report := SkillStatusReport{
		SkillID:             cfg.ID,
		SkillConfig:         skillConfigPath(cfg.ID),
		Language:            cfg.Language,
		Standard:            cfg.Standard,
		Formatter:           cfg.Formatter.ID,
		ExcludedDirectories: cfg.ExcludedDirectories,
		ClangFormat:         Status(),
	}
	if formatterErr == nil {
		report.FormatterConfig = relativeRepoPath(root, formatterConfig)
		report.FormatterConfigExists = fileExists(formatterConfig)
	}
	if templatesErr == nil {
		report.CommentTemplates = relativeRepoPath(root, templatesPath)
		report.CommentTemplatesExists = fileExists(templatesPath)
	}
	if rulesErr == nil {
		report.Rules = relativeRepoPath(root, rulesPath)
		report.RulesExists = fileExists(rulesPath)
	}

	errs := []string{}
	for _, item := range []struct {
		name string
		err  error
	}{
		{name: "formatter config", err: formatterErr},
		{name: "comment templates", err: templatesErr},
		{name: "rules", err: rulesErr},
	} {
		if item.err != nil {
			errs = append(errs, item.name+": "+item.err.Error())
		}
	}
	if len(errs) > 0 {
		return report, errors.New(strings.Join(errs, "; "))
	}
	return report, nil
}

func skillConfigPath(skillID string) string {
	return filepath.ToSlash(filepath.Join(skillConfigRootPath, normalizeSkillID(skillID), "skill.json"))
}

func resolveSkillRelativePath(repoRoot, skillID, rel string) (string, error) {
	root, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return "", err
	}
	cleanRel := strings.TrimSpace(rel)
	if cleanRel == "" {
		return "", fmt.Errorf("empty skill-relative path")
	}
	if filepath.IsAbs(cleanRel) {
		return cleanRel, nil
	}
	return filepath.Join(root, filepath.FromSlash(skillConfigRootPath), normalizeSkillID(skillID), filepath.FromSlash(cleanRel)), nil
}

func normalizeSkillID(skillID string) string {
	id := strings.TrimSpace(skillID)
	if id == "" {
		return DefaultSkillID
	}
	return id
}

func relativeRepoPath(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
