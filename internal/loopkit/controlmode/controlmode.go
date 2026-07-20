// Package controlmode defines the three orthogonal axes of bounded work.
package controlmode

import (
	"errors"
	"fmt"
	"strings"
)

const DefaultContextPressureThreshold = 80.0

// Trigger describes who may propose that bounded work starts.
type Trigger string

const (
	TriggerExplicit      Trigger = "explicit"
	TriggerScheduled     Trigger = "scheduled"
	TriggerAgentProposed Trigger = "agent-proposed"
)

// Stop defines every deterministic rule that can end or checkpoint a loop.
type Stop struct {
	MaxAttempts              int     `json:"maxAttempts"`
	MaxElapsedSeconds        int64   `json:"maxElapsedSeconds"`
	MaxTotalTokens           int64   `json:"maxTotalTokens"`
	StallThreshold           int     `json:"stallThreshold"`
	ContextPressureThreshold float64 `json:"contextPressureThreshold,omitempty"`
}

// WriteScope is the detection boundary used to judge an Agent's changes.
type WriteScope struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// Authority declares what may change, which gates are mandatory, and where a
// person must explicitly take over. It does not grant merge or release rights.
type Authority struct {
	WriteScope    WriteScope `json:"writeScope"`
	RequiredGates []string   `json:"requiredGates"`
	Checkpoints   []string   `json:"checkpoints"`
}

// Control combines orthogonal trigger, stop, and authority axes.
type Control struct {
	Trigger   Trigger   `json:"trigger"`
	Stop      Stop      `json:"stop"`
	Authority Authority `json:"authority"`
}

// Normalized applies deterministic defaults without mutating the caller.
func (c Control) Normalized() Control {
	if c.Stop.ContextPressureThreshold == 0 {
		c.Stop.ContextPressureThreshold = DefaultContextPressureThreshold
	}
	return c
}

// Validate rejects incomplete or unbounded control contracts.
func (c Control) Validate() error {
	c = c.Normalized()
	switch c.Trigger {
	case TriggerExplicit, TriggerScheduled, TriggerAgentProposed:
	default:
		return fmt.Errorf("unsupported trigger: %s", c.Trigger)
	}
	if c.Stop.MaxAttempts < 1 {
		return errors.New("maxAttempts must be >= 1")
	}
	if c.Stop.MaxElapsedSeconds < 1 {
		return errors.New("maxElapsedSeconds must be >= 1")
	}
	if c.Stop.MaxTotalTokens < 1 {
		return errors.New("maxTotalTokens must be >= 1")
	}
	if c.Stop.StallThreshold < 2 {
		return errors.New("stallThreshold must be >= 2")
	}
	if c.Stop.ContextPressureThreshold <= 0 || c.Stop.ContextPressureThreshold > 100 {
		return errors.New("contextPressureThreshold must be in (0, 100]")
	}
	if len(c.Authority.WriteScope.Allow) == 0 {
		return errors.New("writeScope.allow must not be empty")
	}
	if err := validateStrings("writeScope.allow", c.Authority.WriteScope.Allow); err != nil {
		return err
	}
	if err := validateStrings("writeScope.deny", c.Authority.WriteScope.Deny); err != nil {
		return err
	}
	if len(c.Authority.RequiredGates) == 0 {
		return errors.New("requiredGates must not be empty")
	}
	if err := validateStrings("requiredGates", c.Authority.RequiredGates); err != nil {
		return err
	}
	return validateStrings("checkpoints", c.Authority.Checkpoints)
}

func validateStrings(field string, values []string) error {
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s[%d] must not be empty", field, i)
		}
		for previous := 0; previous < i; previous++ {
			if values[previous] == value {
				return fmt.Errorf("%s contains duplicate %q", field, value)
			}
		}
	}
	return nil
}
