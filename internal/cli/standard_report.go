package cli

import "github.com/JiaxI2/AiCoding/internal/report"

func standardReport(command string, profile string, durationMS int64, summary map[string]interface{}, warnings []string, errors []string, details interface{}) report.StandardReport {
	return report.StandardReport{
		Status:     report.StatusFromOK(len(errors) == 0),
		Summary:    summary,
		Findings:   findingsFromMessages(warnings, errors),
		Command:    command,
		Profile:    profile,
		DurationMS: durationMS,
		Logs:       []report.LogRef{},
		Details:    details,
	}
}

func findingsFromMessages(warnings []string, errors []string) []report.Finding {
	findings := make([]report.Finding, 0, len(warnings)+len(errors))
	for _, msg := range warnings {
		if msg != "" {
			findings = append(findings, report.Finding{Level: "WARN", Message: msg})
		}
	}
	for _, msg := range errors {
		if msg != "" {
			findings = append(findings, report.Finding{Level: "ERROR", Message: msg})
		}
	}
	return findings
}
