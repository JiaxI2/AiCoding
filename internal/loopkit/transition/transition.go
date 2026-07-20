// Package transition implements the pure bounded-work adjudicator.
package transition

import (
	"errors"
	"fmt"
	"time"

	"github.com/JiaxI2/AiCoding/internal/loopkit/gateref"
	"github.com/JiaxI2/AiCoding/internal/loopkit/workspec"
	"github.com/JiaxI2/AiCoding/internal/report/tokenusage"
)

// State names the single non-terminal result and the five terminal results.
type State string

const (
	Continue      State = "continue"
	StopSatisfied State = "stop-satisfied"
	StopBudget    State = "stop-budget"
	StopStalled   State = "stop-stalled"
	StopViolation State = "stop-violation"
	Checkpoint    State = "checkpoint"
)

// GateState is the caller-provided interpretation of validation evidence.
type GateState string

const (
	GatePending   GateState = "pending"
	GateSatisfied GateState = "satisfied"
	GateViolation GateState = "violation"
)

// GateStatus describes one validationevidence check for a subject tree.
type GateStatus struct {
	Ref            gateref.GateRef `json:"ref"`
	SubjectTreeOID string          `json:"subjectTreeOID"`
	State          GateState       `json:"state"`
}

// Attempt is an immutable record of one Agent-owned action and its evidence refs.
type Attempt struct {
	Number         int               `json:"number"`
	SubjectTreeOID string            `json:"subjectTreeOID"`
	TokenUsage     tokenusage.Usage  `json:"tokenUsage"`
	GateRefs       []gateref.GateRef `json:"gateRefs"`
	StartedAt      time.Time         `json:"startedAt"`
	EndedAt        time.Time         `json:"endedAt"`
}

type BudgetStatus struct {
	AttemptsUsed   int   `json:"attemptsUsed"`
	MaxAttempts    int   `json:"maxAttempts"`
	ElapsedSeconds int64 `json:"elapsedSeconds"`
	MaxElapsed     int64 `json:"maxElapsedSeconds"`
	TotalTokens    int64 `json:"totalTokens"`
	MaxTotalTokens int64 `json:"maxTotalTokens"`
}

type StallStatus struct {
	Count     int `json:"count"`
	Threshold int `json:"threshold"`
}

type CheckpointStatus struct {
	Reason             string  `json:"reason"`
	ContextUsedPercent float64 `json:"contextUsedPercent"`
	ContextThreshold   float64 `json:"contextPressureThreshold"`
}

// Decision is deterministic for the same four inputs.
type Decision struct {
	State         State             `json:"state"`
	Attempt       int               `json:"attempt"`
	Reason        string            `json:"reason"`
	Budget        BudgetStatus      `json:"budget"`
	Stall         StallStatus       `json:"stall"`
	RequiredGates []GateStatus      `json:"requiredGates"`
	Checkpoint    *CheckpointStatus `json:"checkpoint,omitempty"`
}

// Decide evaluates every stop rule in contract order and returns on the first
// match. It performs no I/O; all facts, including time and gate status, are injected.
func Decide(spec workspec.Spec, history []Attempt, gates []GateStatus, now time.Time) (Decision, error) {
	spec = spec.Normalized()
	if err := spec.Validate(); err != nil {
		return Decision{}, fmt.Errorf("validate spec: %w", err)
	}
	if now.IsZero() {
		return Decision{}, errors.New("now is required")
	}
	if err := validateAttempts(history); err != nil {
		return Decision{}, err
	}
	if err := validateGates(gates); err != nil {
		return Decision{}, err
	}

	decision := Decision{
		State:   Continue,
		Attempt: len(history) + 1,
		Budget: BudgetStatus{
			AttemptsUsed:   len(history),
			MaxAttempts:    spec.Control.Stop.MaxAttempts,
			ElapsedSeconds: elapsedSeconds(history, now),
			MaxElapsed:     spec.Control.Stop.MaxElapsedSeconds,
			TotalTokens:    totalTokens(history),
			MaxTotalTokens: spec.Control.Stop.MaxTotalTokens,
		},
		Stall: StallStatus{
			Count:     trailingSameTree(history),
			Threshold: spec.Control.Stop.StallThreshold,
		},
		RequiredGates: requiredGateStatuses(spec, history, gates),
	}

	if decision.Budget.AttemptsUsed >= decision.Budget.MaxAttempts {
		return finish(decision, StopBudget, "attempt budget exhausted"), nil
	}
	if decision.Budget.ElapsedSeconds >= decision.Budget.MaxElapsed {
		return finish(decision, StopBudget, "elapsed-time budget exhausted"), nil
	}
	if decision.Budget.TotalTokens >= decision.Budget.MaxTotalTokens {
		return finish(decision, StopBudget, "token budget exhausted"), nil
	}
	if decision.Stall.Count >= decision.Stall.Threshold {
		return finish(decision, StopStalled, "subject tree did not change across consecutive attempts"), nil
	}
	if len(history) > 0 {
		used := history[len(history)-1].TokenUsage.ContextUsedPercent
		threshold := spec.Control.Stop.ContextPressureThreshold
		if used > threshold {
			decision.Checkpoint = &CheckpointStatus{
				Reason:             "context pressure",
				ContextUsedPercent: used,
				ContextThreshold:   threshold,
			}
			return finish(decision, Checkpoint, "context pressure"), nil
		}
	}
	for _, gate := range gates {
		if gate.State == GateViolation {
			return finish(decision, StopViolation, "authority boundary violation detected"), nil
		}
	}
	for _, gate := range decision.RequiredGates {
		if gate.State != GateSatisfied {
			return finish(decision, Continue, "required gates are not satisfied for the current tree"), nil
		}
	}
	return finish(decision, StopSatisfied, "all required gates are satisfied for the current tree"), nil
}

func finish(decision Decision, state State, reason string) Decision {
	decision.State = state
	decision.Reason = reason
	return decision
}

func validateAttempts(history []Attempt) error {
	for i, attempt := range history {
		if attempt.Number < 1 {
			return fmt.Errorf("history[%d].number must be >= 1", i)
		}
		if attempt.SubjectTreeOID == "" {
			return fmt.Errorf("history[%d].subjectTreeOID is required", i)
		}
		if attempt.StartedAt.IsZero() || attempt.EndedAt.IsZero() {
			return fmt.Errorf("history[%d] timestamps are required", i)
		}
		if attempt.EndedAt.Before(attempt.StartedAt) {
			return fmt.Errorf("history[%d].endedAt precedes startedAt", i)
		}
		if i > 0 && attempt.Number <= history[i-1].Number {
			return fmt.Errorf("history[%d].number must increase", i)
		}
	}
	return nil
}

func validateGates(gates []GateStatus) error {
	for i, gate := range gates {
		if gate.Ref.Profile == "" {
			return fmt.Errorf("gates[%d].ref.profile is required", i)
		}
		switch gate.State {
		case GatePending, GateSatisfied, GateViolation:
		default:
			return fmt.Errorf("gates[%d] has unsupported state %q", i, gate.State)
		}
		if gate.State == GateSatisfied && (gate.Ref.ValidationIdentity == "" || gate.Ref.ReceiptID == "") {
			return fmt.Errorf("gates[%d] satisfied status requires validation identity and receipt ID", i)
		}
	}
	return nil
}

func elapsedSeconds(history []Attempt, now time.Time) int64 {
	if len(history) == 0 || now.Before(history[0].StartedAt) {
		return 0
	}
	return int64(now.Sub(history[0].StartedAt) / time.Second)
}

func totalTokens(history []Attempt) int64 {
	var total int64
	for _, attempt := range history {
		total += attempt.TokenUsage.TotalTokens
	}
	return total
}

func trailingSameTree(history []Attempt) int {
	if len(history) == 0 || history[len(history)-1].SubjectTreeOID == "" {
		return 0
	}
	count := 1
	last := history[len(history)-1].SubjectTreeOID
	for i := len(history) - 2; i >= 0; i-- {
		if history[i].SubjectTreeOID != last {
			break
		}
		count++
	}
	return count
}

func requiredGateStatuses(spec workspec.Spec, history []Attempt, gates []GateStatus) []GateStatus {
	required := make([]GateStatus, 0, len(spec.Control.Authority.RequiredGates))
	currentTree := ""
	if len(history) > 0 {
		currentTree = history[len(history)-1].SubjectTreeOID
	}
	for _, profile := range spec.Control.Authority.RequiredGates {
		status := GateStatus{Ref: gateref.GateRef{Profile: profile}, SubjectTreeOID: currentTree, State: GatePending}
		for _, candidate := range gates {
			if candidate.Ref.Profile != profile {
				continue
			}
			if currentTree != "" && candidate.SubjectTreeOID != "" && candidate.SubjectTreeOID != currentTree {
				continue
			}
			status = candidate
			break
		}
		required = append(required, status)
	}
	return required
}
