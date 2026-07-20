package transition

import "github.com/JiaxI2/AiCoding/internal/loopkit/evidence"

type State string

const (
	Draft      State = "DRAFT"
	Ready      State = "READY"
	Running    State = "RUNNING"
	Verifying  State = "VERIFYING"
	Verified   State = "VERIFIED"
	Continue   State = "CONTINUE"
	Blocked    State = "BLOCKED"
	NeedsHuman State = "NEEDS_HUMAN"
	Exhausted  State = "EXHAUSTED"
)

type Input struct {
	Current         State
	Attempt         int
	MaxAttempts     int
	RequiredGateIDs []string
	Evidence        evidence.Receipt
	HardBlocked     bool
	HumanRequired   bool
}

type Decision struct {
	Next   State  `json:"next"`
	Reason string `json:"reason"`
}

func Evaluate(in Input) Decision {
	if in.HardBlocked {
		return Decision{Next: Blocked, Reason: "hard block reported"}
	}
	if in.HumanRequired {
		return Decision{Next: NeedsHuman, Reason: "human checkpoint required"}
	}
	if err := in.Evidence.RequiredPassed(in.RequiredGateIDs); err == nil {
		return Decision{Next: Verified, Reason: "all required gates passed"}
	}
	if in.Attempt >= in.MaxAttempts {
		return Decision{Next: Exhausted, Reason: "attempt budget exhausted"}
	}
	return Decision{Next: Continue, Reason: "required gates not yet satisfied"}
}
