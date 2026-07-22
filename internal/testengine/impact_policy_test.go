package testengine

import (
	"reflect"
	"testing"
)

func TestSelectChangeProfileUsesSharedPathPolicyBoundary(t *testing.T) {
	policy := ChangeImpactPolicy{
		DefaultProfile: ProfileFull,
		Rules: []ChangeImpactRule{
			{Pattern: "internal/cli/**", Profile: ProfileSmoke, Reason: "boundary"},
			{Pattern: "docs/*.md", Profile: ProfileSmoke, Reason: "single segment"},
		},
	}
	profile, matches, err := SelectChangeProfile(policy, []string{"internal/cli/x/y.go", "internal/clix/y.go", "docs/architecture/README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if profile != ProfileFull {
		t.Fatalf("profile = %q, want %q", profile, ProfileFull)
	}
	want := []ChangeImpactMatch{{Path: "internal/cli/x/y.go", Pattern: "internal/cli/**", Profile: "Smoke", Reason: "boundary"}}
	if !reflect.DeepEqual(matches, want) {
		t.Fatalf("matches = %#v, want %#v", matches, want)
	}
}
