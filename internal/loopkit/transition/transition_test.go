package transition

import (
	"github.com/JiaxI2/AiCoding/internal/loopkit/evidence"
	"testing"
)

func TestRequiredFailureCannotVerify(t *testing.T) {
	d := Evaluate(Input{Attempt: 1, MaxAttempts: 3, RequiredGateIDs: []string{"test"}, Evidence: evidence.Receipt{Checks: []evidence.Check{{ID: "test", Status: "FAIL"}}}})
	if d.Next == Verified {
		t.Fatal("failed required gate must not verify")
	}
}

func TestBudgetExhaustion(t *testing.T) {
	d := Evaluate(Input{Attempt: 3, MaxAttempts: 3, RequiredGateIDs: []string{"test"}, Evidence: evidence.Receipt{Checks: []evidence.Check{{ID: "test", Status: "FAIL"}}}})
	if d.Next != Exhausted {
		t.Fatalf("got %s", d.Next)
	}
}
