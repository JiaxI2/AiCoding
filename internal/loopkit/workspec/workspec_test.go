package workspec

import "testing"

func validSpec() Spec {
	return Spec{SchemaVersion: 1, ID: "w1", Domain: "project-development", Control: Control{Mode: "goal", Trigger: map[string]interface{}{"type": "manual"}, StoppingRule: map[string]interface{}{"success": "verified"}}, Goal: "implement", Acceptance: []string{"tests pass"}, Policy: Policy{Workspace: "worktree", WriteScope: WriteScope{}, Verification: map[string]interface{}{}, Budget: Budget{MaxAttempts: 3, MaxElapsedSeconds: 60}}}
}

func TestDigestIsStable(t *testing.T) {
	s := validSpec()
	a, err := s.Digest()
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("digest drift: %s != %s", a, b)
	}
}

func TestRejectsUnknownMode(t *testing.T) {
	s := validSpec()
	s.Control.Mode = "forever"
	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
