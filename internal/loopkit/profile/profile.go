// Package profile provides static defaults for recurring bounded-work domains.
package profile

import (
	"github.com/JiaxI2/AiCoding/internal/loopkit/controlmode"
	"github.com/JiaxI2/AiCoding/internal/loopkit/workspec"
)

type Profile struct {
	ID                 string                `json:"id"`
	Domain             workspec.Domain       `json:"domain"`
	AllowedTriggers    []controlmode.Trigger `json:"allowedTriggers"`
	RequiredGates      []string              `json:"requiredGates"`
	AdvisoryGates      []string              `json:"advisoryGates"`
	DefaultCheckpoints []string              `json:"defaultCheckpoints"`
}

func Builtins() []Profile {
	return []Profile{
		{
			ID:                 "project-development",
			Domain:             workspec.DomainProjectDevelopment,
			AllowedTriggers:    []controlmode.Trigger{controlmode.TriggerExplicit, controlmode.TriggerScheduled},
			RequiredGates:      []string{"full"},
			DefaultCheckpoints: []string{"architecture-change", "merge"},
		},
		{
			ID:                 "repository-maintenance",
			Domain:             workspec.DomainRepositoryMaintenance,
			AllowedTriggers:    []controlmode.Trigger{controlmode.TriggerExplicit, controlmode.TriggerScheduled, controlmode.TriggerAgentProposed},
			RequiredGates:      []string{"full"},
			AdvisoryGates:      []string{"docsync"},
			DefaultCheckpoints: []string{"architecture-unfreeze", "submodule-pin-change", "merge", "release"},
		},
	}
}
