package workspec

import (
	"testing"

	"github.com/JiaxI2/AiCoding/internal/loopkit/controlmode"
)

func validSpec() Spec {
	return Spec{
		SchemaVersion: 1,
		ID:            "w1",
		Domain:        DomainProjectDevelopment,
		Goal:          "implement",
		Acceptance:    []string{"tests pass"},
		Control: controlmode.Control{
			Trigger: controlmode.TriggerExplicit,
			Stop: controlmode.Stop{
				MaxAttempts:       3,
				MaxElapsedSeconds: 60,
				MaxTotalTokens:    1000,
				StallThreshold:    2,
			},
			Authority: controlmode.Authority{
				WriteScope:    controlmode.WriteScope{Allow: []string{"internal/**"}},
				RequiredGates: []string{"full"},
			},
		},
	}
}

func TestDigestIsStableAndNormalizesDefaults(t *testing.T) {
	s := validSpec()
	a, err := s.Digest()
	if err != nil {
		t.Fatal(err)
	}
	s.Control.Stop.ContextPressureThreshold = controlmode.DefaultContextPressureThreshold
	b, err := s.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("digest drift: %s != %s", a, b)
	}
}

func TestRejectsUnsupportedTrigger(t *testing.T) {
	s := validSpec()
	s.Control.Trigger = "forever"
	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRejectsMissingRequiredGates(t *testing.T) {
	s := validSpec()
	s.Control.Authority.RequiredGates = nil
	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
