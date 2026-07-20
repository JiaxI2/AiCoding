// Package workspec owns the immutable contract for one bounded work item.
package workspec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/loopkit/controlmode"
)

// Domain classifies work without changing the transition semantics.
type Domain string

const (
	DomainProjectDevelopment    Domain = "project-development"
	DomainRepositoryMaintenance Domain = "repository-maintenance"
	DomainCIRepair              Domain = "ci-repair"
	DomainPerformanceExperiment Domain = "performance-experiment"
	DomainDocumentation         Domain = "documentation-maintenance"
	DomainArchitecture          Domain = "architecture-evolution"
)

// Spec is the complete, immutable input to the transition decision.
type Spec struct {
	SchemaVersion int                 `json:"schemaVersion"`
	ID            string              `json:"id"`
	Domain        Domain              `json:"domain"`
	Control       controlmode.Control `json:"control"`
	Goal          string              `json:"goal"`
	Acceptance    []string            `json:"acceptance"`
}

// Normalized applies deterministic defaults without mutating the caller.
func (s Spec) Normalized() Spec {
	s.Control = s.Control.Normalized()
	return s
}

// Validate rejects incomplete, unbounded, or unsupported work contracts.
func (s Spec) Validate() error {
	s = s.Normalized()
	if s.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion: %d", s.SchemaVersion)
	}
	if strings.TrimSpace(s.ID) == "" {
		return errors.New("id is required")
	}
	if !validID(s.ID) {
		return errors.New("id must be lowercase kebab-case")
	}
	switch s.Domain {
	case DomainProjectDevelopment, DomainRepositoryMaintenance, DomainCIRepair,
		DomainPerformanceExperiment, DomainDocumentation, DomainArchitecture:
	default:
		return fmt.Errorf("unsupported domain: %s", s.Domain)
	}
	if strings.TrimSpace(s.Goal) == "" {
		return errors.New("goal is required")
	}
	if len(s.Acceptance) == 0 {
		return errors.New("at least one acceptance criterion is required")
	}
	for i, criterion := range s.Acceptance {
		if strings.TrimSpace(criterion) == "" {
			return fmt.Errorf("acceptance[%d] must not be empty", i)
		}
	}
	return s.Control.Validate()
}

// Digest returns the content identity of the normalized specification.
func (s Spec) Digest() (string, error) {
	s = s.Normalized()
	if err := s.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validID(id string) bool {
	if id == "" || id[0] == '-' || id[len(id)-1] == '-' {
		return false
	}
	previousDash := false
	for _, char := range id {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			previousDash = false
		case char == '-' && !previousDash:
			previousDash = true
		default:
			return false
		}
	}
	return true
}
