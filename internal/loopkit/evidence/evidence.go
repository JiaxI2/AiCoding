package evidence

import "fmt"

type Check struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	EvidenceRef string `json:"evidenceRef,omitempty"`
}

type Receipt struct {
	SchemaVersion int     `json:"schemaVersion"`
	WorkDigest    string  `json:"workDigest"`
	Attempt       int     `json:"attempt"`
	Checks        []Check `json:"checks"`
}

func (r Receipt) RequiredPassed(requiredIDs []string) error {
	status := map[string]string{}
	for _, c := range r.Checks {
		status[c.ID] = c.Status
	}
	for _, id := range requiredIDs {
		if status[id] != "PASS" {
			return fmt.Errorf("required gate %s is not PASS", id)
		}
	}
	return nil
}
