package releasegate

import "github.com/JiaxI2/AiCoding/internal/repohealth"

type Result struct {
	OK     bool                           `json:"ok"`
	Checks []repohealth.ReleaseNotesCheck `json:"checks"`
	Scope  string                         `json:"scope"`
	Notes  []string                       `json:"notes,omitempty"`
}

func Verify(repo string) (Result, []string) {
	checks, errs := repohealth.VerifyReleaseNotes(repo)
	result := Result{
		OK:     len(errs) == 0,
		Checks: checks,
		Scope:  "structural-fast-check",
		Notes: []string{
			"does not replace scripts/verify-release-governance-overlay.ps1",
			"does not run Full or Release slow path",
		},
	}
	return result, errs
}
