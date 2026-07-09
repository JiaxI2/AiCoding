package tagpolicy

import "testing"

func TestClassifyTagNamespaces(t *testing.T) {
	policy := DefaultPolicy()
	cases := map[string]string{
		"v2.4.6":                        "platform",
		"kit/example-kit/v2.3.4":        "kit",
		"milestone/2099.12.31-baseline": "milestone",
		"v2099.12.31-baseline":          "noncurrent-date",
		"v2.3.4-example-kit":            "noncurrent-component",
		"bad/tag":                       "unknown",
	}
	for tag, want := range cases {
		if got := Classify(tag, policy); got != want {
			t.Fatalf("Classify(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestAuditTagsReturnsWarningsForNonCurrentTags(t *testing.T) {
	policy := DefaultPolicy()
	audit := AuditTags([]string{"v2.4.6", "v2.3.4-example-kit"}, policy)
	if audit.Total != 2 || audit.Counts["platform"] != 1 || audit.Counts["noncurrent-component"] != 1 {
		t.Fatalf("unexpected audit counts: %#v", audit)
	}
	if len(audit.Warnings) == 0 {
		t.Fatalf("expected non-current warning")
	}
}
