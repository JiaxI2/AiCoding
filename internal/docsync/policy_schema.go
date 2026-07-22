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
	{Config: "config/docs-sync.policy.json", Schema: "config/schemas/docs-sync-policy.schema.json"},
	{Config: "config/docs-sync.semantic.json", Schema: "config/schemas/docs-sync-semantic.schema.json"},
	{Config: "config/impact-policy.json", Schema: "config/schemas/impact-policy.schema.json"},
	{Config: "config/plan-policy.json", Schema: "config/schemas/plan-policy.schema.json"},
	{Config: "config/tagging-policy.json", Schema: "config/schemas/tagging-policy.schema.json"},
	{Config: "config/validation-policy.json", Schema: "config/schemas/validation-policy.schema.json"},
}

// CheckPolicySchemas validates AiCoding's complete policy surface against its
// checked-in schemas. Repositories without the plan-policy marker are outside
// this repository-specific closure gate.
func CheckPolicySchemas(repo string) (bool, []string) {
	markerConfig := filepath.Join(repo, filepath.FromSlash("config/plan-policy.json"))
	markerSchema := filepath.Join(repo, filepath.FromSlash("config/schemas/plan-policy.schema.json"))
	if _, configErr := os.Stat(markerConfig); os.IsNotExist(configErr) {
		if _, schemaErr := os.Stat(markerSchema); os.IsNotExist(schemaErr) {
			return false, nil
		}
	}

	errs := []string{}
	for _, binding := range policySchemaBindings {
		if err := validatePolicySchemaBinding(repo, binding); err != nil {
			errs = append(errs, err.Error())
		}
	}
	sort.Strings(errs)
	return true, errs
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
