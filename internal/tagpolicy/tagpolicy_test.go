package tagpolicy

import "testing"

func TestClassifyTagNamespaces(t *testing.T) {
	policy := DefaultPolicy()
	cases := map[string]string{
		"v0.2.0":                          "platform",
		"kit/powershell-skill-kit/v1.3.0": "kit",
		"milestone/2026.07.03-fast-path":  "milestone",
		"v2026.07.03-fast-path-v1":        "legacy-historical",
		"v1.3.0-powershell-skill-kit":     "legacy-component",
		"bad/tag":                         "unknown",
	}
	for tag, want := range cases {
		if got := Classify(tag, policy); got != want {
			t.Fatalf("Classify(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestAuditTagsReturnsWarningsForLegacyTags(t *testing.T) {
	policy := DefaultPolicy()
	audit := AuditTags([]string{"v0.2.0", "v1.3.0-powershell-skill-kit"}, policy)
	if audit.Total != 2 || audit.Counts["platform"] != 1 || audit.Counts["legacy-component"] != 1 {
		t.Fatalf("unexpected audit counts: %#v", audit)
	}
	if len(audit.Warnings) == 0 {
		t.Fatalf("expected legacy warning")
	}
}
