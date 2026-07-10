package report

type Finding struct {
	Level    string `json:"level"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Category string `json:"category,omitempty"`
	ID       string `json:"id,omitempty"`
}

type LogRef struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type StandardReport struct {
	Status     string                 `json:"status"`
	Summary    map[string]interface{} `json:"summary"`
	Findings   []Finding              `json:"findings"`
	Command    string                 `json:"command"`
	Profile    string                 `json:"profile"`
	DurationMS int64                  `json:"duration_ms"`
	Logs       []LogRef               `json:"logs"`
	Details    interface{}            `json:"details,omitempty"`
}

func StatusFromOK(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
