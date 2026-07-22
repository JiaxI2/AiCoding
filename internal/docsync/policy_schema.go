package docsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type policySchemaBinding struct {
	Config string
	Schema string
}

var policySchemaBindings = []policySchemaBinding{
	{Config: "config/agent-dev-kit-plan-mode.registry.json", Schema: "config/schemas/agent-dev-kit-plan-mode.registry.schema.json"},
	{Config: "config/codex-kit.json", Schema: "config/schemas/codex-kit.schema.json"},
	{Config: "config/common-registry.json", Schema: "config/schemas/common-registry.schema.json"},
	{Config: "config/dependency-governance.json", Schema: "config/schemas/dependency-governance.schema.json"},
	{Config: "config/docs-sync.policy.json", Schema: "config/schemas/docs-sync-policy.schema.json"},
	{Config: "config/docs-sync.semantic.json", Schema: "config/schemas/docs-sync-semantic.schema.json"},
	{Config: "config/hooks-registry.json", Schema: "config/schemas/hooks-registry.schema.json"},
	{Config: "config/impact-policy.json", Schema: "config/schemas/impact-policy.schema.json"},
	{Config: "config/internal-capabilities.json", Schema: "config/schemas/internal-capabilities.schema.json"},
	{Config: "config/kit-registry.json", Schema: "config/schemas/kit-registry.schema.json"},
	{Config: "config/kits/aicoding-platform.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/kits/c-userstyle-kit.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/kits/common-control-kit.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/kits/docsync-plus.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/kits/loop-engineering-kit.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/kits/release-governance-overlay-kit.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/kits/reuse-governance.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/mcp-registry.json", Schema: "config/schemas/mcp-registry.schema.json"},
	{Config: "config/mcp/components/ppt-mcp.json", Schema: "config/schemas/mcp-component.schema.json"},
	{Config: "config/mcp/components/visio-mcp.json", Schema: "config/schemas/mcp-component.schema.json"},
	{Config: "config/plan-policy.json", Schema: "config/schemas/plan-policy.schema.json"},
	{Config: "config/pwsh-budget.json", Schema: "config/schemas/pwsh-budget.schema.json"},
	{Config: "config/repository-layout.json", Schema: "config/schemas/repository-layout.schema.json"},
	{Config: "config/repository-navigation.json", Schema: "config/schemas/repository-navigation.schema.json"},
	{Config: "config/reuse-governance.json", Schema: "config/schemas/reuse-governance.schema.json"},
	{Config: "config/schema-closure-exclusions.json", Schema: "config/schemas/schema-closure-exclusions.schema.json"},
	{Config: "config/skill-sources.json", Schema: "config/schemas/skill-sources.schema.json"},
	{Config: "config/skills/c99-standard-c/skill.json", Schema: "config/schemas/c99-standard-c-skill.schema.json"},
	{Config: "config/skills/c99-standard-c/templates/comment-templates.json", Schema: "config/schemas/comment-templates.schema.json"},
	{Config: "config/tagging-policy.json", Schema: "config/schemas/tagging-policy.schema.json"},
	{Config: "config/templates/kit/manifest-external.tmpl.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/templates/kit/manifest.tmpl.json", Schema: "config/schemas/kit-manifest.schema.json"},
	{Config: "config/templates/kit/workspec-example.tmpl.json", Schema: "config/schemas/loop-work-spec.schema.json"},
	{Config: "config/templates/mcp/component.tmpl.json", Schema: "config/schemas/mcp-component.schema.json"},
	{Config: "config/validation-policy.json", Schema: "config/schemas/validation-policy.schema.json"},
}

type standaloneSchemaBinding struct {
	Schema      string
	Enforcement string
}

var standaloneSchemaBindings = []standaloneSchemaBinding{
	{Schema: "config/schemas/cli-report.schema.json", Enforcement: "internal/report contract and FREEZE-001"},
	{Schema: "config/schemas/loop-attempt.schema.json", Enforcement: "Loop Engineering attempt artifact contract"},
	{Schema: "config/schemas/loop-profile.schema.json", Enforcement: "Loop Engineering profile artifact contract"},
	{Schema: "config/schemas/plan-spec.schema.json", Enforcement: "internal/plan PLAN.md frontmatter contract"},
}

const schemaClosureExclusionsPath = "config/schema-closure-exclusions.json"

type schemaClosureExclusions struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Exclusions    []schemaClosureExclusion `json:"exclusions"`
}

type schemaClosureExclusion struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// CheckPolicySchemas validates AiCoding's complete policy surface against its
// checked-in schemas. Repositories without the plan-policy marker are outside
// this repository-specific closure gate.
func CheckPolicySchemas(repo string) (bool, []string) {
	if !policySchemaClosureEnabled(repo) {
		return false, nil
	}

	errs := []string{}
	for _, binding := range policySchemaBindings {
		if err := validatePolicySchemaBinding(repo, binding); err != nil {
			errs = append(errs, err.Error())
		}
	}
	for _, binding := range standaloneSchemaBindings {
		if _, err := readJSONObject(filepath.Join(repo, filepath.FromSlash(binding.Schema))); err != nil {
			errs = append(errs, fmt.Sprintf("standalone schema %s (%s): %v", binding.Schema, binding.Enforcement, err))
		}
	}
	sort.Strings(errs)
	return true, errs
}

// CheckConfigSchemaCompleteness verifies the bidirectional config/schema
// inventory. The caller supplies its existing repository inventory so this
// check does not introduce another filesystem walk.
func CheckConfigSchemaCompleteness(repo string, inventoryFiles, inventoryDirectories []string) (bool, []string) {
	if !policySchemaClosureEnabled(repo) {
		return false, nil
	}

	var exclusions schemaClosureExclusions
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(schemaClosureExclusionsPath)))
	if err != nil {
		return true, []string{schemaClosureExclusionsPath + ": " + err.Error()}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&exclusions); err != nil {
		return true, []string{schemaClosureExclusionsPath + ": " + err.Error()}
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return true, []string{schemaClosureExclusionsPath + ": " + err.Error()}
	}

	errs := checkConfigSchemaCompleteness(inventoryFiles, inventoryDirectories, exclusions)
	sort.Strings(errs)
	return true, errs
}

func policySchemaClosureEnabled(repo string) bool {
	markerConfig := filepath.Join(repo, filepath.FromSlash("config/plan-policy.json"))
	markerSchema := filepath.Join(repo, filepath.FromSlash("config/schemas/plan-policy.schema.json"))
	if _, configErr := os.Stat(markerConfig); os.IsNotExist(configErr) {
		if _, schemaErr := os.Stat(markerSchema); os.IsNotExist(schemaErr) {
			return false
		}
	}
	return true
}

func checkConfigSchemaCompleteness(inventoryFiles, inventoryDirectories []string, exclusions schemaClosureExclusions) []string {
	errs := []string{}
	files := make(map[string]bool, len(inventoryFiles))
	for _, path := range inventoryFiles {
		if normalized, err := normalizeSchemaClosurePath(path); err == nil {
			files[normalized] = true
		}
	}
	directories := make(map[string]bool, len(inventoryDirectories))
	for _, path := range inventoryDirectories {
		if normalized, err := normalizeSchemaClosurePath(path); err == nil {
			directories[normalized] = true
		}
	}

	boundConfigs := map[string]bool{}
	referencedSchemas := map[string]bool{}
	for _, binding := range policySchemaBindings {
		if boundConfigs[binding.Config] {
			errs = append(errs, "duplicate policy schema config binding: "+binding.Config)
		}
		boundConfigs[binding.Config] = true
		referencedSchemas[binding.Schema] = true
		if !files[binding.Config] {
			errs = append(errs, "registered config is missing: "+binding.Config)
		}
		if !files[binding.Schema] {
			errs = append(errs, "registered schema is missing: "+binding.Schema)
		}
	}
	for _, binding := range standaloneSchemaBindings {
		if referencedSchemas[binding.Schema] {
			errs = append(errs, "duplicate standalone schema registration: "+binding.Schema)
		}
		referencedSchemas[binding.Schema] = true
		if !files[binding.Schema] {
			errs = append(errs, "registered standalone schema is missing: "+binding.Schema)
		}
	}

	if exclusions.SchemaVersion != 1 {
		errs = append(errs, fmt.Sprintf("%s schemaVersion=%d, want 1", schemaClosureExclusionsPath, exclusions.SchemaVersion))
	}
	validatedExclusions := []schemaClosureExclusion{}
	seenExclusions := map[string]bool{}
	for _, exclusion := range exclusions.Exclusions {
		normalized, directory, err := validateSchemaClosureExclusion(exclusion, files, directories)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if seenExclusions[normalized] {
			errs = append(errs, "duplicate schema closure exclusion: "+normalized)
			continue
		}
		seenExclusions[normalized] = true
		if directory {
			normalized += "/**"
		}
		validatedExclusions = append(validatedExclusions, schemaClosureExclusion{Path: normalized, Reason: exclusion.Reason})
	}

	for path := range files {
		if !strings.HasPrefix(path, "config/") || filepath.Ext(path) != ".json" {
			continue
		}
		excluded := matchesSchemaClosureExclusion(path, validatedExclusions)
		if boundConfigs[path] {
			if excluded {
				errs = append(errs, "bound config must not also be excluded: "+path)
			}
			continue
		}
		if !excluded {
			errs = append(errs, "config JSON is not registered or excluded: "+path)
		}
		if strings.HasPrefix(path, "config/schemas/") && !referencedSchemas[path] {
			errs = append(errs, "schema JSON is not reverse-registered: "+path)
		}
	}
	return errs
}

func validateSchemaClosureExclusion(exclusion schemaClosureExclusion, files, directories map[string]bool) (string, bool, error) {
	if strings.TrimSpace(exclusion.Reason) == "" {
		return "", false, fmt.Errorf("schema closure exclusion %q has no reason", exclusion.Path)
	}
	directory := strings.HasSuffix(exclusion.Path, "/**")
	path := exclusion.Path
	if directory {
		path = strings.TrimSuffix(path, "/**")
		if strings.ContainsAny(path, "*?[]") {
			return "", false, fmt.Errorf("schema closure exclusion %q uses an unsupported wildcard", exclusion.Path)
		}
	} else if strings.ContainsAny(path, "*?[]") {
		return "", false, fmt.Errorf("schema closure exclusion %q uses an unsupported wildcard; only directory /** is allowed", exclusion.Path)
	}
	normalized, err := normalizeSchemaClosurePath(path)
	if err != nil || normalized != path || !strings.HasPrefix(normalized, "config/") {
		return "", false, fmt.Errorf("schema closure exclusion %q must be a normalized path under config/", exclusion.Path)
	}
	if directory {
		if !directories[normalized] {
			return "", false, fmt.Errorf("schema closure exclusion directory does not exist: %s", normalized)
		}
		return normalized, true, nil
	}
	if filepath.Ext(normalized) != ".json" {
		return "", false, fmt.Errorf("schema closure exclusion must name a JSON file: %s", normalized)
	}
	if !files[normalized] {
		return "", false, fmt.Errorf("schema closure exclusion file does not exist: %s", normalized)
	}
	return normalized, false, nil
}

func normalizeSchemaClosurePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("invalid path")
	}
	normalized := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("invalid path")
	}
	return normalized, nil
}

func matchesSchemaClosureExclusion(path string, exclusions []schemaClosureExclusion) bool {
	for _, exclusion := range exclusions {
		if strings.HasSuffix(exclusion.Path, "/**") {
			root := strings.TrimSuffix(exclusion.Path, "/**")
			if strings.HasPrefix(path, root+"/") {
				return true
			}
			continue
		}
		if path == exclusion.Path {
			return true
		}
	}
	return false
}

func validatePolicySchemaBinding(repo string, binding policySchemaBinding) error {
	schema, err := readJSONObject(filepath.Join(repo, filepath.FromSlash(binding.Schema)))
	if err != nil {
		return fmt.Errorf("policy schema %s: %w", binding.Schema, err)
	}
	value, err := readJSONValue(filepath.Join(repo, filepath.FromSlash(binding.Config)))
	if err != nil {
		return fmt.Errorf("policy config %s: %w", binding.Config, err)
	}
	errs := validateJSONSchema(schema, schema, value, "$")
	if len(errs) != 0 {
		return fmt.Errorf("policy config %s violates %s: %s", binding.Config, binding.Schema, strings.Join(errs, "; "))
	}
	return nil
}

func readJSONObject(path string) (map[string]any, error) {
	value, err := readJSONValue(path)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema root must be an object")
	}
	return object, nil
}

func readJSONValue(path string) (any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func validateJSONSchema(root, schema map[string]any, value any, path string) []string {
	if ref, ok := schema["$ref"].(string); ok {
		resolved, err := resolveLocalSchemaRef(root, ref)
		if err != nil {
			return []string{path + ": " + err.Error()}
		}
		return validateJSONSchema(root, resolved, value, path)
	}

	errs := []string{}
	if expected, ok := schema["type"].(string); ok && !jsonSchemaTypeMatches(expected, value) {
		return []string{fmt.Sprintf("%s: expected %s", path, expected)}
	}
	if expected, ok := schema["const"]; ok && !reflect.DeepEqual(expected, value) {
		errs = append(errs, fmt.Sprintf("%s: value does not match const", path))
	}
	if values, ok := schema["enum"].([]any); ok {
		matched := false
		for _, candidate := range values {
			if reflect.DeepEqual(candidate, value) {
				matched = true
				break
			}
		}
		if !matched {
			errs = append(errs, fmt.Sprintf("%s: value is not in enum", path))
		}
	}

	if object, ok := value.(map[string]any); ok {
		required, _ := schema["required"].([]any)
		for _, item := range required {
			name, _ := item.(string)
			if _, exists := object[name]; !exists {
				errs = append(errs, fmt.Sprintf("%s: required property %q is missing", path, name))
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			propertySchema, declared := properties[key]
			if declared {
				if typed, ok := propertySchema.(map[string]any); ok {
					errs = append(errs, validateJSONSchema(root, typed, object[key], jsonPath(path, key))...)
				}
				continue
			}
			switch additional := schema["additionalProperties"].(type) {
			case bool:
				if !additional {
					errs = append(errs, fmt.Sprintf("%s: additional property %q is not allowed", path, key))
				}
			case map[string]any:
				errs = append(errs, validateJSONSchema(root, additional, object[key], jsonPath(path, key))...)
			}
		}
	}

	if array, ok := value.([]any); ok {
		if minimum, ok := schemaInteger(schema["minItems"]); ok && len(array) < minimum {
			errs = append(errs, fmt.Sprintf("%s: expected at least %d items", path, minimum))
		}
		if unique, _ := schema["uniqueItems"].(bool); unique {
			seen := map[string]bool{}
			for index, item := range array {
				encoded, _ := json.Marshal(item)
				key := string(encoded)
				if seen[key] {
					errs = append(errs, fmt.Sprintf("%s[%d]: duplicate item", path, index))
				}
				seen[key] = true
			}
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for index, item := range array {
				errs = append(errs, validateJSONSchema(root, itemSchema, item, fmt.Sprintf("%s[%d]", path, index))...)
			}
		}
	}

	if text, ok := value.(string); ok {
		if minimum, ok := schemaInteger(schema["minLength"]); ok && utf8.RuneCountInString(text) < minimum {
			errs = append(errs, fmt.Sprintf("%s: string is shorter than %d", path, minimum))
		}
		if expression, ok := schema["pattern"].(string); ok {
			pattern, err := regexp.Compile(expression)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: schema pattern is invalid: %v", path, err))
			} else if !pattern.MatchString(text) {
				errs = append(errs, fmt.Sprintf("%s: string does not match pattern", path))
			}
		}
	}

	if branches, ok := schema["oneOf"].([]any); ok {
		matched := 0
		for _, branch := range branches {
			if typed, ok := branch.(map[string]any); ok && len(validateJSONSchema(root, typed, value, path)) == 0 {
				matched++
			}
		}
		if matched != 1 {
			errs = append(errs, fmt.Sprintf("%s: expected exactly one oneOf branch, matched %d", path, matched))
		}
	}
	return errs
}

func resolveLocalSchemaRef(root map[string]any, ref string) (map[string]any, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("unsupported schema ref %q", ref)
	}
	var current any = root
	for _, encoded := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		key := strings.ReplaceAll(strings.ReplaceAll(encoded, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schema ref %q does not resolve to an object", ref)
		}
		current, ok = object[key]
		if !ok {
			return nil, fmt.Errorf("schema ref %q is missing", ref)
		}
	}
	resolved, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema ref %q does not resolve to an object", ref)
	}
	return resolved, nil
}

func jsonSchemaTypeMatches(expected string, value any) bool {
	switch expected {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := value.(json.Number)
		return ok
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		parsed, err := strconv.ParseFloat(number.String(), 64)
		return err == nil && parsed == float64(int64(parsed))
	default:
		return false
	}
}

func schemaInteger(value any) (int, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(number.String())
	return parsed, err == nil
}

func jsonPath(parent, key string) string {
	if regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(key) {
		return parent + "." + key
	}
	return parent + "[" + strconv.Quote(key) + "]"
}
