package workspec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Control struct {
	Mode         string                 `json:"mode"`
	Trigger      map[string]interface{} `json:"trigger"`
	StoppingRule map[string]interface{} `json:"stoppingRule"`
}

type WriteScope struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type Budget struct {
	MaxAttempts       int `json:"maxAttempts"`
	MaxElapsedSeconds int `json:"maxElapsedSeconds"`
}

type Policy struct {
	Workspace        string                 `json:"workspace"`
	WriteScope       WriteScope             `json:"writeScope"`
	Verification     map[string]interface{} `json:"verification"`
	Budget           Budget                 `json:"budget"`
	HumanCheckpoints []string               `json:"humanCheckpoints"`
}

type Spec struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Domain        string   `json:"domain"`
	Control       Control  `json:"control"`
	Goal          string   `json:"goal"`
	Acceptance    []string `json:"acceptance"`
	Policy        Policy   `json:"policy"`
}

var validModes = map[string]bool{"turn": true, "goal": true, "time": true, "proactive": true}

func (s Spec) Validate() error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion: %d", s.SchemaVersion)
	}
	if strings.TrimSpace(s.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(s.Domain) == "" {
		return errors.New("domain is required")
	}
	if !validModes[s.Control.Mode] {
		return fmt.Errorf("unsupported control mode: %s", s.Control.Mode)
	}
	if strings.TrimSpace(s.Goal) == "" {
		return errors.New("goal is required")
	}
	if len(s.Acceptance) == 0 {
		return errors.New("at least one acceptance criterion is required")
	}
	if s.Policy.Budget.MaxAttempts < 1 {
		return errors.New("maxAttempts must be >= 1")
	}
	if s.Policy.Budget.MaxElapsedSeconds < 1 {
		return errors.New("maxElapsedSeconds must be >= 1")
	}
	return nil
}

func (s Spec) Digest() (string, error) {
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
