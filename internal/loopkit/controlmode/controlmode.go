package controlmode

import "fmt"

type Mode string

const (
	Turn      Mode = "turn"
	Goal      Mode = "goal"
	Time      Mode = "time"
	Proactive Mode = "proactive"
)

type Constraints struct {
	RequiresExternalEvaluator bool `json:"requiresExternalEvaluator"`
	RequiresExplicitTrigger   bool `json:"requiresExplicitTrigger"`
	RequiresAuthorityBoundary bool `json:"requiresAuthorityBoundary"`
}

func Rules(mode Mode) (Constraints, error) {
	switch mode {
	case Turn:
		return Constraints{RequiresExplicitTrigger: true}, nil
	case Goal:
		return Constraints{RequiresExternalEvaluator: true, RequiresExplicitTrigger: true}, nil
	case Time:
		return Constraints{RequiresExternalEvaluator: true, RequiresExplicitTrigger: true}, nil
	case Proactive:
		return Constraints{RequiresExternalEvaluator: true, RequiresExplicitTrigger: true, RequiresAuthorityBoundary: true}, nil
	default:
		return Constraints{}, fmt.Errorf("unsupported control mode: %s", mode)
	}
}
