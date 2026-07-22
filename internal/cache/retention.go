package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func Clean(repo string, options CleanOptions) (CleanResult, error) {
	result := CleanResult{Scope: "all", DryRun: options.DryRun, Keep: options.Keep, Adopt: options.Adopt, AllRepos: options.AllRepos}
	if options.Keep < 0 {
		return result, fmt.Errorf("keep must be at least 1")
	}
	if (options.Adopt || options.AllRepos) && options.Scope != ScopeTemp {
		return result, fmt.Errorf("adopt and all-repos require temp scope")
	}
	if options.Scope != "" {
		result.Scope = string(options.Scope)
		if !ValidScope(options.Scope) {
			return result, fmt.Errorf("unsupported cache scope %q", options.Scope)
		}
	}

	selected := make([]artifactSpec, 0)
	for _, spec := range registry(repo) {
		if options.Scope != "" && spec.scope != options.Scope {
			continue
		}
		if options.Scope == "" && !spec.cleanByDefault {
			continue
		}
		if !spec.cleanable {
			return result, fmt.Errorf("cache scope %q is audit-only and cannot be cleaned", spec.scope)
		}
		selected = append(selected, spec)
	}

	plans := make([]scopePlan, 0, len(selected))
	for _, spec := range selected {
		plan, err := planScope(repo, spec, options)
		if err != nil {
			return result, err
		}
		plans = append(plans, plan)
		result.PlannedCount += len(plan.entries)
		for _, entry := range plan.entries {
			result.PlannedBytes += entry.size
		}
	}

	for _, plan := range plans {
		scopeResult := ScopeCleanResult{
			Scope: plan.spec.scope, Path: plan.spec.displayPath, Policy: plan.spec.policy,
			RetainedCount: plan.retained, Planned: cleanEntries(plan.spec.scope, plan.entries),
		}
		for _, entry := range plan.entries {
			scopeResult.PlannedBytes += entry.size
		}
		if !options.DryRun {
			for _, entry := range plan.entries {
				if plan.spec.scope == ScopeTemp {
					if entry.adopt {
						if err := platform.RecordTempOutcome(repo, entry.path, entry.kind, "adopted"); err != nil {
							result.Scopes = append(result.Scopes, scopeResult)
							return result, err
						}
					}
					if err := platform.RecordTempOutcome(repo, entry.path, entry.kind, "releasing"); err != nil {
						result.Scopes = append(result.Scopes, scopeResult)
						return result, err
					}
				}
				if err := removePlanned(plan.spec, entry); err != nil {
					result.Scopes = append(result.Scopes, scopeResult)
					return result, err
				}
				if plan.spec.scope == ScopeTemp {
					if err := platform.RecordTempOutcome(repo, entry.path, entry.kind, "released"); err != nil {
						result.Scopes = append(result.Scopes, scopeResult)
						return result, err
					}
				}
				removed := CleanEntry{Scope: plan.spec.scope, Path: entry.display, SizeBytes: entry.size, Reason: entry.reason}
				scopeResult.Removed = append(scopeResult.Removed, removed)
				scopeResult.FreedBytes += entry.size
				result.RemovedCount++
				result.FreedBytes += entry.size
			}
		}
		result.Scopes = append(result.Scopes, scopeResult)
	}
	return result, nil
}

type plannedEntry struct {
	path      string
	display   string
	size      int64
	modTime   int64
	directory bool
	reason    string
	kind      string
	adopt     bool
}

type scopePlan struct {
	spec     artifactSpec
	entries  []plannedEntry
	retained int
}

func planScope(repo string, spec artifactSpec, options CleanOptions) (scopePlan, error) {
	if spec.scope == ScopeTemp {
		return planTempScope(repo, spec, options)
	}
	status, entries, err := scan(repo, spec)
	if err != nil {
		return scopePlan{}, err
	}
	plan := scopePlan{spec: spec, retained: status.EntryCount}
	keep := options.Keep
	if keep == 0 {
		keep = DefaultTestResultKeep
	}
	switch spec.scope {
	case ScopeFastPath:
		if !status.Exists {
			return plan, nil
		}
		plan.entries = []plannedEntry{{
			path: spec.root, display: spec.displayPath, size: status.SizeBytes,
			directory: true, reason: "fast-path cache is fully disposable",
		}}
		plan.retained = 0
	case ScopeTestResults:
		for index, entry := range entries {
			protected, err := failedOrUnclearTestResult(entry.path)
			if err != nil {
				return scopePlan{}, err
			}
			if index < keep || protected {
				continue
			}
			plan.entries = append(plan.entries, plannedFromArtifact(entry, "older than keep limit and not failed evidence"))
			plan.retained--
		}
	case ScopeValidationReports:
		referenced, err := referencedValidationReports(repo, spec.root)
		if err != nil {
			return scopePlan{}, err
		}
		for _, entry := range entries {
			identity := "sha256:" + filepath.Base(entry.path)
			if referenced[identity] {
				continue
			}
			plan.entries = append(plan.entries, plannedFromArtifact(entry, "orphaned validation report"))
			plan.retained--
		}
	case ScopePins:
		referenced, err := kit.ReferencedPinIdentities(repo)
		if err != nil {
			return scopePlan{}, fmt.Errorf("resolve registry-referenced pins: %w", err)
		}
		for _, entry := range entries {
			identity := "sha256:" + filepath.Base(entry.path)
			if referenced[identity] {
				continue
			}
			plan.entries = append(plan.entries, plannedFromArtifact(entry, "content-addressed pin is not referenced by the Kit registry"))
			plan.retained--
		}
	}
	sort.Slice(plan.entries, func(i, j int) bool {
		if plan.entries[i].modTime == plan.entries[j].modTime {
			return plan.entries[i].display < plan.entries[j].display
		}
		return plan.entries[i].modTime < plan.entries[j].modTime
	})
	return plan, nil
}

func planTempScope(repo string, spec artifactSpec, options CleanOptions) (scopePlan, error) {
	status, entries, err := scan(repo, spec)
	if err != nil {
		return scopePlan{}, err
	}
	records, err := platform.ReadTempLedger(repo)
	if err != nil {
		return scopePlan{}, err
	}
	latest := latestTempRecords(records)
	keep := options.Keep
	if keep == 0 {
		keep = DefaultTempKeep
	}

	type candidate struct {
		entry     artifactEntry
		record    platform.TempRecord
		createdAt time.Time
		failure   bool
		adopt     bool
	}
	candidates := []candidate{}
	failures := []candidate{}
	for _, entry := range entries {
		record, registered := latest[tempPathKey(entry.path)]
		if !registered {
			if options.Adopt {
				candidates = append(candidates, candidate{entry: entry, createdAt: time.Unix(0, entry.modTime), adopt: true})
			}
			continue
		}
		if !options.AllRepos && !sameTempRepo(record.RepoRoot, repo) {
			continue
		}
		if record.Outcome == "investigating" {
			continue
		}
		createdAt, parseErr := time.Parse(time.RFC3339Nano, record.CreatedAt)
		if parseErr != nil {
			return scopePlan{}, fmt.Errorf("invalid temp ledger createdAt for %s: %w", record.Path, parseErr)
		}
		if cacheNow().UTC().Sub(createdAt) < tempRecentWindow {
			continue
		}
		item := candidate{entry: entry, record: record, createdAt: createdAt}
		switch record.Outcome {
		case "created", "failed", "releasing":
			item.failure = true
			failures = append(failures, item)
		}
		candidates = append(candidates, item)
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].createdAt.After(failures[j].createdAt) })
	protected := map[string]bool{}
	for index := 0; index < len(failures) && index < keep; index++ {
		protected[tempPathKey(failures[index].entry.path)] = true
	}

	plan := scopePlan{spec: spec, retained: status.EntryCount}
	for _, item := range candidates {
		if protected[tempPathKey(item.entry.path)] {
			continue
		}
		kind := item.record.Kind
		reason := "older than 24h and outside retained failure set"
		if item.adopt {
			kind = inferTempKind(filepath.Base(item.entry.path))
			reason = "explicitly adopted unledgered aicoding-* directory"
		}
		plan.entries = append(plan.entries, plannedEntry{
			path: item.entry.path, display: item.entry.display, size: item.entry.size,
			modTime: item.entry.modTime, directory: item.entry.directory,
			reason: reason, kind: kind, adopt: item.adopt,
		})
		plan.retained--
	}
	sort.Slice(plan.entries, func(i, j int) bool {
		if plan.entries[i].modTime == plan.entries[j].modTime {
			return plan.entries[i].display < plan.entries[j].display
		}
		return plan.entries[i].modTime < plan.entries[j].modTime
	})
	return plan, nil
}

func inferTempKind(name string) string {
	trimmed := strings.TrimPrefix(name, platform.TempDirectoryPrefix)
	if strings.HasPrefix(trimmed, "fresh-clone-") {
		return "fresh-clone"
	}
	if index := strings.IndexByte(trimmed, '-'); index > 0 {
		trimmed = trimmed[:index]
	}
	trimmed = strings.Trim(trimmed, "-")
	if trimmed == "" {
		return "adopted"
	}
	for _, char := range trimmed {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return "adopted"
		}
	}
	return trimmed
}

func failedOrUnclearTestResult(path string) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(path, "summary.json"))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	var summary struct {
		Conclusion string `json:"conclusion"`
		Fail       int    `json:"fail"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		return true, nil
	}
	if summary.Fail > 0 || strings.EqualFold(strings.TrimSpace(summary.Conclusion), "FAIL") {
		return true, nil
	}
	if strings.TrimSpace(summary.Conclusion) == "" {
		return true, nil
	}
	return false, nil
}

func referencedValidationReports(repo, reportRoot string) (map[string]bool, error) {
	store, err := validationevidence.Open(repo)
	if err != nil {
		return nil, err
	}
	receipts, err := store.List("")
	if err != nil {
		return nil, err
	}
	referenced := make(map[string]bool, len(receipts))
	for _, receipt := range receipts {
		referenced[receipt.ValidationIdentity] = true
	}

	aliasRoot := filepath.Join(filepath.Dir(reportRoot), "aliases")
	err = filepath.WalkDir(aliasRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		identity := strings.TrimSpace(string(raw))
		if strings.HasPrefix(identity, "sha256:") && validReportDirName(strings.TrimPrefix(identity, "sha256:")) {
			referenced[identity] = true
		}
		return nil
	})
	if os.IsNotExist(err) {
		return referenced, nil
	}
	return referenced, err
}

func plannedFromArtifact(entry artifactEntry, reason string) plannedEntry {
	return plannedEntry{
		path: entry.path, display: entry.display, size: entry.size, modTime: entry.modTime,
		directory: entry.directory, reason: reason,
	}
}

func cleanEntries(scope Scope, planned []plannedEntry) []CleanEntry {
	entries := make([]CleanEntry, 0, len(planned))
	for _, entry := range planned {
		entries = append(entries, CleanEntry{Scope: scope, Path: entry.display, SizeBytes: entry.size, Reason: entry.reason})
	}
	return entries
}

func removePlanned(spec artifactSpec, entry plannedEntry) error {
	root, err := filepath.Abs(spec.root)
	if err != nil {
		return err
	}
	target, err := filepath.Abs(entry.path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refuse to clean path outside %s: %s", root, target)
	}
	if target == root && spec.scope != ScopeFastPath {
		return fmt.Errorf("refuse to remove scope root %s", target)
	}
	if spec.scope == ScopeTemp && !strings.HasPrefix(filepath.Base(target), platform.TempDirectoryPrefix) {
		return fmt.Errorf("refuse to remove non-%s path: %s", platform.TempDirectoryPrefix, target)
	}
	if entry.directory {
		return os.RemoveAll(target)
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
