package transition

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/loopkit/controlmode"
	"github.com/JiaxI2/AiCoding/internal/loopkit/gateref"
	"github.com/JiaxI2/AiCoding/internal/loopkit/workspec"
	"github.com/JiaxI2/AiCoding/internal/report/tokenusage"
)

var baseTime = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func validSpec() workspec.Spec {
	return workspec.Spec{
		SchemaVersion: 1,
		ID:            "loop-test",
		Domain:        workspec.DomainProjectDevelopment,
		Goal:          "finish bounded work",
		Acceptance:    []string{"full gate passes"},
		Control: controlmode.Control{
			Trigger: controlmode.TriggerExplicit,
			Stop: controlmode.Stop{
				MaxAttempts:       5,
				MaxElapsedSeconds: 300,
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

func attempt(number int, tree string, totalTokens int64, contextPercent float64, started time.Time) Attempt {
	return Attempt{
		Number:         number,
		SubjectTreeOID: tree,
		TokenUsage: tokenusage.Usage{
			TotalTokens:        totalTokens,
			ContextUsedPercent: contextPercent,
		},
		StartedAt: started,
		EndedAt:   started.Add(10 * time.Second),
	}
}

func gate(state GateState, tree string) GateStatus {
	ref := gateref.GateRef{Profile: "full"}
	if state == GateSatisfied {
		ref.ValidationIdentity = "sha256:validation"
		ref.ReceiptID = "receipt-1"
	}
	return GateStatus{Ref: ref, SubjectTreeOID: tree, State: state}
}

func TestDecideNamedStates(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*workspec.Spec)
		history []Attempt
		gates   []GateStatus
		now     time.Time
		want    State
	}{
		{name: "continue", history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}, gates: []GateStatus{gate(GatePending, "tree-a")}, now: baseTime.Add(20 * time.Second), want: Continue},
		{name: "satisfied", history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}, gates: []GateStatus{gate(GateSatisfied, "tree-a")}, now: baseTime.Add(20 * time.Second), want: StopSatisfied},
		{name: "attempt budget", mutate: func(s *workspec.Spec) { s.Control.Stop.MaxAttempts = 1 }, history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}, now: baseTime.Add(20 * time.Second), want: StopBudget},
		{name: "elapsed budget", mutate: func(s *workspec.Spec) { s.Control.Stop.MaxElapsedSeconds = 10 }, history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}, now: baseTime.Add(20 * time.Second), want: StopBudget},
		{name: "token budget", mutate: func(s *workspec.Spec) { s.Control.Stop.MaxTotalTokens = 10 }, history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}, now: baseTime.Add(20 * time.Second), want: StopBudget},
		{name: "stalled", history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime), attempt(2, "tree-a", 10, 20, baseTime.Add(20*time.Second))}, now: baseTime.Add(40 * time.Second), want: StopStalled},
		{name: "violation", history: []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}, gates: []GateStatus{gate(GateViolation, "tree-a")}, now: baseTime.Add(20 * time.Second), want: StopViolation},
		{name: "context checkpoint", history: []Attempt{attempt(1, "tree-a", 10, 81, baseTime)}, gates: []GateStatus{gate(GateSatisfied, "tree-a")}, now: baseTime.Add(20 * time.Second), want: Checkpoint},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := validSpec()
			if tt.mutate != nil {
				tt.mutate(&spec)
			}
			got, err := Decide(spec, tt.history, tt.gates, tt.now)
			if err != nil {
				t.Fatal(err)
			}
			if got.State != tt.want {
				t.Fatalf("got %s (%s), want %s", got.State, got.Reason, tt.want)
			}
		})
	}
}

func TestDecideUsesCurrentTreeEvidenceOnly(t *testing.T) {
	history := []Attempt{attempt(1, "tree-new", 10, 20, baseTime)}
	got, err := Decide(validSpec(), history, []GateStatus{gate(GateSatisfied, "tree-old")}, baseTime.Add(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if got.State != Continue {
		t.Fatalf("got %s", got.State)
	}
}

func TestDecideIsByteDeterministic(t *testing.T) {
	history := []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}
	gates := []GateStatus{gate(GatePending, "tree-a")}
	a, err := Decide(validSpec(), history, gates, baseTime.Add(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Decide(validSpec(), history, gates, baseTime.Add(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	if string(aJSON) != string(bJSON) {
		t.Fatalf("decision drift:\n%s\n%s", aJSON, bJSON)
	}
}

func TestBudgetRulePrecedesSatisfiedGate(t *testing.T) {
	spec := validSpec()
	spec.Control.Stop.MaxAttempts = 1
	history := []Attempt{attempt(1, "tree-a", 10, 20, baseTime)}
	got, err := Decide(spec, history, []GateStatus{gate(GateSatisfied, "tree-a")}, baseTime.Add(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StopBudget {
		t.Fatalf("got %s", got.State)
	}
}
