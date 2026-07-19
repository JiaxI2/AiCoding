package lifecycle

import (
	"github.com/JiaxI2/AiCoding/internal/repocontext"
)

// runRepoContextAdapter translates a unified lifecycle request into the
// repo-context domain. The domain owns its own generated artifacts and state; the
// kernel modules (snapshot/runner/report) are untouched. See ADR 0003.
//
// Each domain action scans the repository at most once and reports the facts
// digest it worked against; the adapter reuses that digest as InputDigest rather
// than scanning a second time. uninstall reads only the manifest and never scans.
func runRepoContextAdapter(repo string, opts Options) AdapterResult {
	result := AdapterResult{ID: ScopeRepoContext, Action: opts.Action, DryRun: opts.DryRun, OK: false, Status: "failed"}

	var domain repocontext.Report
	switch opts.Action {
	case "install":
		domain = repocontext.Install(repo, opts.DryRun)
	case "update":
		domain = repocontext.Update(repo, opts.DryRun)
	case "uninstall":
		domain = repocontext.Uninstall(repo, opts.DryRun)
	case "status":
		domain = repocontext.Status(repo)
	case "doctor":
		domain = repocontext.Doctor(repo)
	case "verify":
		domain = repocontext.Verify(repo)
	default:
		result.Errors = []string{"unsupported repo-context lifecycle action: " + opts.Action}
		return result
	}

	result.InputDigest = domain.FactsDigest
	result.OK = domain.OK
	result.Status = domain.Status
	result.Data = domain
	result.Warnings = domain.Warnings
	result.Errors = domain.Errors
	return result
}
