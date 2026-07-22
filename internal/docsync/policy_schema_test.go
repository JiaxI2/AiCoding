package docsync

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryPolicySchemaClosureIsSixOfSix(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	checked, errs := CheckPolicySchemas(repo)
	if !checked || len(errs) != 0 {
		t.Fatalf("repository policy schemas did not close: checked=%v errors=%#v", checked, errs)
	}
	if len(policySchemaBindings) != 6 {
		t.Fatalf("policy schema bindings = %d, want 6", len(policySchemaBindings))
	}
}

func TestPolicySchemaClosureRejectsUnknownImpactField(t *testing.T) {
	repo := t.TempDir()
	for _, binding := range policySchemaBindings {
		writeDocSyncTestFile(t, repo, binding.Config, "{}\n")
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
