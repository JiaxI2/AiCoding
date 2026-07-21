package report

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAggregateChecksUsesSharedStatusContract(t *testing.T) {
	checks := []Check{
		NewCheck("pass", "TEST", time.Now(), nil, nil, nil),
		NewCheck("warn", "TEST", time.Now(), nil, []string{"review"}, nil),
		NewCheck("fail", "TEST", time.Now(), nil, nil, []string{"broken"}),
	}
	summary, warnings, errorsFound := AggregateChecks(checks)
	if summary["total"] != 3 || summary["pass"] != 1 || summary["warn"] != 1 || summary["fail"] != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(warnings) != 1 || warnings[0] != "warn: review" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(errorsFound) != 1 || errorsFound[0] != "fail: broken" {
		t.Fatalf("unexpected errors: %#v", errorsFound)
	}
}

func TestValidationErrorClassification(t *testing.T) {
	err := BoolErr([]string{"invalid state"})
	if err == nil || !IsValidationError(err) || err.Error() != "invalid state" {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if IsValidationError(errors.New("execution failure")) {
		t.Fatal("plain errors must not be classified as validation errors")
	}
}

func TestResultDecisionCategoriesAreClosedAndCanonical(t *testing.T) {
	valid := []Category{
		CategoryNone, CategoryUsage, CategoryValidation, CategoryTransient,
		CategoryToolchain, CategoryEvidenceMissing, CategoryConflict, CategoryInternal,
	}
	for _, category := range valid {
		if !ValidCategory(category) {
			t.Fatalf("declared category is not valid: %q", category)
		}
	}
	if ValidCategory(Category("free-text")) {
		t.Fatal("free-text category was accepted")
	}

	usage := FinalizeDecision(Result{SchemaVersion: 1, Command: "kit verify", ErrorKind: ErrorKindUsage})
	if usage.Category != CategoryUsage || usage.Retryable || usage.NextAction == "" {
		t.Fatalf("usage decision is incomplete: %#v", usage)
	}
	transient := FinalizeDecision(WithDecision(Result{SchemaVersion: 1, Command: "fresh-clone"}, CategoryTransient, "aicoding fresh-clone --profile Release --json"))
	if transient.Category != CategoryTransient || !transient.Retryable {
		t.Fatalf("transient decision is not retryable: %#v", transient)
	}
	invalid := FinalizeDecision(Result{SchemaVersion: 1, Command: "broken", Category: Category("free-text")})
	if invalid.OK || invalid.Category != CategoryInternal || invalid.Retryable || len(invalid.Errors) == 0 {
		t.Fatalf("invalid decision did not fail closed: %#v", invalid)
	}
}

func TestCLIReportSchemaAndGoTypesStayAligned(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "config", "schemas", "cli-report.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("invalid CLI report schema: %v", err)
	}
	definitions, ok := schema["$defs"].(map[string]interface{})
	if !ok || definitions["standardReport"] == nil || definitions["check"] == nil {
		t.Fatalf("CLI report schema is missing shared definitions: %#v", definitions)
	}
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok || properties["inputDigest"] == nil || properties["planDigest"] == nil {
		t.Fatalf("CLI report schema is missing digest evidence fields: %#v", properties)
	}

	sample := Result{
		SchemaVersion: SchemaVersion,
		Command:       "verify --profile Smoke",
		OK:            true,
		Category:      CategoryNone,
		Retryable:     false,
		InputDigest:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PlanDigest:    "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Data: StandardReport{
			SchemaVersion: SchemaVersion,
			Status:        "PASS",
			Summary:       map[string]interface{}{"total": 1},
			Findings:      []Finding{},
			Command:       "verify --profile Smoke",
			Profile:       "Smoke",
			DurationMS:    1,
			Logs:          []LogRef{},
			Details:       []Check{},
		},
		ElapsedMS: 1,
	}
	encoded, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]interface{}
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	if object["schemaVersion"] != float64(SchemaVersion) || object["command"] == "" || object["elapsedMs"] == nil ||
		object["inputDigest"] == nil || object["planDigest"] == nil || object["category"] != string(CategoryNone) || object["retryable"] != false {
		t.Fatalf("result JSON does not match schema-required fields: %s", encoded)
	}
	standard, ok := object["data"].(map[string]interface{})
	if !ok || standard["schemaVersion"] != float64(SchemaVersion) || standard["duration_ms"] == nil {
		t.Fatalf("standard report JSON does not match schema-required fields: %s", encoded)
	}
}
