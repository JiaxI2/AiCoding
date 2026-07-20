package controlmode

import "testing"

func TestNormalizedDefaultsContextPressure(t *testing.T) {
	c := Control{Stop: Stop{}}
	if got := c.Normalized().Stop.ContextPressureThreshold; got != DefaultContextPressureThreshold {
		t.Fatalf("got %v", got)
	}
}

func TestValidateRejectsDuplicateGate(t *testing.T) {
	c := Control{
		Trigger: TriggerExplicit,
		Stop:    Stop{MaxAttempts: 1, MaxElapsedSeconds: 1, MaxTotalTokens: 1, StallThreshold: 2},
		Authority: Authority{
			WriteScope:    WriteScope{Allow: []string{"internal/**"}},
			RequiredGates: []string{"full", "full"},
		},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected duplicate required gate to fail")
	}
}
