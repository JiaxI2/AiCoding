package repocontext

import (
	"os"
	"sort"
	"strings"
)

// Sync incrementally refreshes the generated context after a commit. It is the
// hook-driven internal step of update (not a new lifecycle verb): given the paths
// changed by the commit, it re-scans, reconciles the owned artifacts writing only
// files whose content changed, and reports which top-level domains were affected.
//
// When the domain is not installed (no manifest) it is a quiet no-op, so the
// post-commit hook is harmless for repositories that never ran install.
func Sync(repo string, changedPaths []string, dryRun bool) Report {
	report := Report{Action: "sync", DryRun: dryRun, Status: "failed"}

	existing, err := loadManifest(repo)
	if err != nil {
		if os.IsNotExist(err) {
			report.OK = true
			report.Status = "not-installed"
			report.Installed = false
			return report
		}
		report.Errors = []string{"cannot read manifest: " + err.Error()}
		return report
	}
	report.Installed = true

	facts, snapshot, err := Scan(repo)
	if err != nil {
		report.Errors = []string{"cannot scan repository: " + err.Error()}
		return report
	}
	report.FactsDigest = snapshot.Digest()

	affected := affectedDomains(changedPaths)
	if len(affected) > 0 {
		report.Warnings = append(report.Warnings, "affected domains: "+strings.Join(affected, ", "))
	}

	if dryRun {
		report.OK = true
		report.Status = statusForDryRun(true)
		report.Fresh = existing.FactsDigest == snapshot.Digest()
		return report
	}

	written, err := reconcile(repo, snapshot.Digest(), render(facts), existing, &report)
	if err != nil {
		report.Errors = []string{err.Error()}
		return report
	}
	report.Files = written
	report.OK = true
	report.Status = "ok"
	report.Fresh = true
	return report
}

// affectedDomains derives the sorted, de-duplicated set of top-level domains that
// a commit's changed paths touch. Paths outside any top-level directory (root
// files) and paths inside the owned root are ignored.
func affectedDomains(changedPaths []string) []string {
	seen := map[string]bool{}
	for _, raw := range changedPaths {
		rel := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
		if rel == "" || strings.HasPrefix(rel, ownedRoot+"/") {
			continue
		}
		top := topLevel(rel)
		if top == "" || skipDirs[top] {
			continue
		}
		seen[top] = true
	}
	domains := make([]string, 0, len(seen))
	for domain := range seen {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	return domains
}
