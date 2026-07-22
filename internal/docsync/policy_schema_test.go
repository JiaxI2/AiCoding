package docsync

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryPolicySchemaClosureIsThirtyFiveOfThirtyFive(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	checked, errs := CheckPolicySchemas(repo)
	if !checked || len(errs) != 0 {
		t.Fatalf("repository policy schemas did not close: checked=%v errors=%#v", checked, errs)
	}
	if len(policySchemaBindings) != 35 {
		t.Fatalf("policy schema bindings = %d, want 35", len(policySchemaBindings))
	}
	if len(standaloneSchemaBindings) != 4 {
		t.Fatalf("standalone schema bindings = %d, want 4", len(standaloneSchemaBindings))
	}
}

func TestPolicySchemaClosureRejectsUnknownImpactField(t *testing.T) {
	repo := t.TempDir()
	for _, binding := range policySchemaBindings {
		writeDocSyncTestFile(t, repo, binding.Config, "{}\n")
		writeDocSyncTestFile(t, repo, binding.Schema, "{}\n")
	}
	for _, binding := range standaloneSchemaBindings {
		writeDocSyncTestFile(t, repo, binding.Schema, "{}\n")
	}
	writeDocSyncTestFile(t, repo, "config/impact-policy.json", `{"schemaVersion":1,"illegal":true}`+"\n")
	writeDocSyncTestFile(t, repo, "config/schemas/impact-policy.schema.json", `{
  "type": "object",
  "required": ["schemaVersion"],
  "additionalProperties": false,
  "properties": {"schemaVersion": {"const": 1}}
}`+"\n")

	checked, errs := CheckPolicySchemas(repo)
	if !checked || len(errs) != 1 || !strings.Contains(errs[0], `additional property "illegal" is not allowed`) {
		t.Fatalf("unknown field was not rejected: checked=%v errors=%#v", checked, errs)
	}
}

func TestJSONSchemaCommentIsIgnored(t *testing.T) {
	schema := map[string]any{
		"$comment":             "metadata only",
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"value": map[string]any{"type": "string", "$comment": "metadata only"},
		},
	}
	value := map[string]any{"value": "ok"}
	if errs := validateJSONSchema(schema, schema, value, "$"); len(errs) != 0 {
		t.Fatalf("$comment changed validation: %#v", errs)
	}
}

func TestConfigSchemaCompletenessAcceptsRegisteredInventory(t *testing.T) {
	files, directories := completePolicySchemaInventory()
	exclusions := schemaClosureExclusions{
		SchemaVersion: 1,
		Exclusions: []schemaClosureExclusion{{
			Path:   "config/schemas/**",
			Reason: "schemas are reverse-registered",
		}},
	}
	if errs := checkConfigSchemaCompleteness(files, directories, exclusions); len(errs) != 0 {
		t.Fatalf("registered inventory did not close: %#v", errs)
	}
}

func TestConfigSchemaCompletenessRejectsRogueConfigAndSchema(t *testing.T) {
	files, directories := completePolicySchemaInventory()
	files = append(files, "config/rogue.json", "config/schemas/ghost.schema.json")
	exclusions := schemaClosureExclusions{
		SchemaVersion: 1,
		Exclusions: []schemaClosureExclusion{{
			Path:   "config/schemas/**",
			Reason: "schemas are reverse-registered",
		}},
	}
	errs := checkConfigSchemaCompleteness(files, directories, exclusions)
	if !containsPolicySchemaError(errs, "config JSON is not registered or excluded: config/rogue.json") ||
		!containsPolicySchemaError(errs, "schema JSON is not reverse-registered: config/schemas/ghost.schema.json") {
		t.Fatalf("rogue inventory was not rejected precisely: %#v", errs)
	}
}

func TestConfigSchemaCompletenessRejectsGhostAndFuzzyExclusions(t *testing.T) {
	files, directories := completePolicySchemaInventory()
	exclusions := schemaClosureExclusions{
		SchemaVersion: 1,
		Exclusions: []schemaClosureExclusion{
			{Path: "config/missing.json", Reason: "ghost"},
			{Path: "config/*.json", Reason: "fuzzy"},
			{Path: "config/schemas/**", Reason: "schemas are reverse-registered"},
		},
	}
	errs := checkConfigSchemaCompleteness(files, directories, exclusions)
	if !containsPolicySchemaError(errs, "schema closure exclusion file does not exist: config/missing.json") ||
		!containsPolicySchemaError(errs, "only directory /** is allowed") {
		t.Fatalf("invalid exclusions were not rejected precisely: %#v", errs)
	}
}

func completePolicySchemaInventory() ([]string, []string) {
	files := []string{}
	seen := map[string]bool{}
	for _, binding := range policySchemaBindings {
		for _, path := range []string{binding.Config, binding.Schema} {
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
		}
	}
	for _, binding := range standaloneSchemaBindings {
		if !seen[binding.Schema] {
			seen[binding.Schema] = true
			files = append(files, binding.Schema)
		}
	}
	return files, []string{"config", "config/schemas"}
}

func containsPolicySchemaError(errs []string, fragment string) bool {
	for _, err := range errs {
		if strings.Contains(err, fragment) {
			return true
		}
	}
	return false
}
