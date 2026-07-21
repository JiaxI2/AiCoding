package plan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheckPathsAppliesExemptionBeforeSensitivePatterns(t *testing.T) {
	policy := Policy{
		SchemaVersion: 1,
		SensitivePaths: []SensitiveRule{
			{Pattern: "internal/cli/**", Reason: "frozen kernel"},
			{Pattern: "docs/**", Reason: "documentation"},
			{Pattern: "config/kit-registry.json", Reason: "kit activation"},
		},
		ExemptPaths: []string{"docs/todolist/**"},
	}
	check, err := CheckPaths(policy, []string{
		"README.md",
		"docs/todolist/0004.md",
		"internal/clix/not-a-match.go",
		"internal/cli/x/y.go",
		"config/kit-registry.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(check.Exempt, []string{"docs/todolist/0004.md"}) {
		t.Fatalf("exempt = %#v", check.Exempt)
	}
	want := []SensitiveMatch{
		{Path: "config/kit-registry.json", Pattern: "config/kit-registry.json", Reason: "kit activation"},
		{Path: "internal/cli/x/y.go", Pattern: "internal/cli/**", Reason: "frozen kernel"},
	}
	if !reflect.DeepEqual(check.Sensitive, want) {
		t.Fatalf("sensitive = %#v, want %#v", check.Sensitive, want)
	}
}

func TestLoadPolicyNormalizesSortsAndRejectsInvalidPatterns(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, filepath.FromSlash(PolicyPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`{
  "schemaVersion": 1,
  "sensitivePaths": [
    {"pattern":"z/**","reason":"z"},
    {"pattern":"a/**","reason":"a"},
    {"pattern":"z/**","reason":"z"}
  ],
  "exemptPaths": ["docs/spec/**", "docs/spec/**"]
}`)
	policy, err := LoadPolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.SensitivePaths) != 2 || policy.SensitivePaths[0].Pattern != "a/**" || !reflect.DeepEqual(policy.ExemptPaths, []string{"docs/spec/**"}) {
		t.Fatalf("policy was not normalized: %#v", policy)
	}

	write(`{"schemaVersion":1,"sensitivePaths":[{"pattern":"/internal/**","reason":"bad"}],"exemptPaths":[]}`)
	if _, err := LoadPolicy(repo); err == nil {
		t.Fatal("LoadPolicy accepted an absolute pattern")
	}
	write(`{"schemaVersion":1,"sensitivePaths":[{"pattern":"a/**","reason":"a"}],"exemptPaths":[],"unknown":true}`)
	if _, err := LoadPolicy(repo); err == nil {
		t.Fatal("LoadPolicy accepted an unknown schema field")
	}
}

func TestCheckPathsRejectsNonRepositoryRelativeInput(t *testing.T) {
	policy := Policy{SchemaVersion: 1, SensitivePaths: []SensitiveRule{{Pattern: "internal/**", Reason: "kernel"}}}
	if _, err := CheckPaths(policy, []string{"../outside.go"}); err == nil {
		t.Fatal("CheckPaths accepted traversal")
	}
}

func TestMatchPatternUsesPlanGlobDialect(t *testing.T) {
	matched, err := MatchPattern("internal/**/*.go", "internal/plan/plan.go")
	if err != nil || !matched {
		t.Fatalf("double-star pattern did not match: matched=%v err=%v", matched, err)
	}
	matched, err = MatchPattern("docs/*.md", "docs/architecture/README.md")
	if err != nil || matched {
		t.Fatalf("single-star crossed a directory: matched=%v err=%v", matched, err)
	}
	if _, err := MatchPattern("../**", "internal/plan/plan.go"); err == nil {
		t.Fatal("traversal pattern was accepted")
	}
}
