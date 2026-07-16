package report

import "time"

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
	SchemaVersion int                    `json:"schemaVersion"`
	Status        string                 `json:"status"`
	Summary       map[string]interface{} `json:"summary"`
	Findings      []Finding              `json:"findings"`
	Command       string                 `json:"command"`
	Profile       string                 `json:"profile"`
	DurationMS    int64                  `json:"duration_ms"`
	Logs          []LogRef               `json:"logs"`
	Details       interface{}            `json:"details,omitempty"`
}

type Check struct {
	ID         string      `json:"id"`
	Category   string      `json:"category"`
	OK         bool        `json:"ok"`
	Status     string      `json:"status"`
	DurationMS int64       `json:"duration_ms"`
	Warnings   []string    `json:"warnings,omitempty"`
	Errors     []string    `json:"errors,omitempty"`
	Details    interface{} `json:"details,omitempty"`
}

func StatusFromOK(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func StatusFromMessages(warnings, errors []string) string {
	if len(errors) > 0 {
		return "FAIL"
	}
	if len(warnings) > 0 {
		return "PASS_WITH_WARNINGS"
	}
	return "PASS"
}

func NewCheck(id, category string, started time.Time, details interface{}, warnings, errors []string) Check {
	return Check{
		ID:         id,
		Category:   category,
		OK:         len(errors) == 0,
		Status:     StatusFromMessages(warnings, errors),
		DurationMS: time.Since(started).Milliseconds(),
		Warnings:   append([]string{}, warnings...),
		Errors:     append([]string{}, errors...),
		Details:    details,
	}
}

func AggregateChecks(checks []Check) (map[string]interface{}, []string, []string) {
	pass := 0
	warn := 0
	fail := 0
	warnings := []string{}
	errors := []string{}
	for _, check := range checks {
		switch {
		case !check.OK || len(check.Errors) > 0 || check.Status == "FAIL":
			fail++
		case len(check.Warnings) > 0 || check.Status == "PASS_WITH_WARNINGS":
			warn++
		default:
			pass++
		}
		for _, warning := range check.Warnings {
			warnings = append(warnings, check.ID+": "+warning)
		}
		for _, issue := range check.Errors {
			errors = append(errors, check.ID+": "+issue)
		}
	}
	return map[string]interface{}{
		"total": len(checks),
		"pass":  pass,
		"warn":  warn,
		"fail":  fail,
	}, warnings, errors
}
