package repocontext

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

const manifestSchemaVersion = 1

// FactsDigest scans the repository and returns the stable facts digest. It is the
// InputDigest the lifecycle adapter reports for every repo-context action.
func FactsDigest(repo string) (string, error) {
	_, snapshot, err := Scan(repo)
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

// Install generates scoped context files from the current repository facts and
// records them in the manifest. With dryRun it only reports the planned files and
// writes nothing.
func Install(repo string, dryRun bool) Report {
	return apply(repo, "install", dryRun)
}

// Update re-scans and converges the generated artifacts to the current facts. It
// shares the same domain path as Install (plan = action + dryRun).
func Update(repo string, dryRun bool) Report {
	return apply(repo, "update", dryRun)
}

func apply(repo, action string, dryRun bool) Report {
	report := Report{Action: action, DryRun: dryRun, Status: "failed"}
	facts, snapshot, err := Scan(repo)
	if err != nil {
		report.Errors = []string{"cannot scan repository: " + err.Error()}
		return report
	}
	report.FactsDigest = snapshot.Digest()
	desired := render(facts)

	existing, _ := loadManifest(repo) // absent manifest is treated as empty
	existingByPath := map[string]string{}
	for _, file := range existing.Files {
		existingByPath[file.Path] = file.Digest
	}

	planned := make([]PlannedFile, 0, len(desired))
	for _, file := range desired {
		digest := contentDigest(file.Content)
		change := "create"
		if prev, ok := existingByPath[file.Path]; ok {
			if prev == digest {
				change = "unchanged"
			} else {
				change = "update"
			}
		}
		planned = append(planned, PlannedFile{Path: file.Path, Digest: digest, Change: change})
	}
	report.Planned = planned

	if dryRun {
		report.OK = true
		report.Status = "planned"
		report.Installed = len(existing.Files) > 0
		report.Fresh = report.Installed && manifestMatches(existing, desired)
		return report
	}

	written, err := reconcile(repo, snapshot.Digest(), desired, existing, &report)
	if err != nil {
		report.Errors = []string{err.Error()}
		return report
	}
	report.Files = written
	report.OK = true
	report.Status = "ok"
	report.Installed = true
	report.Fresh = true
	return report
}

// reconcile converges the owned artifacts on disk to the desired set, writing only
// files whose content actually changed and removing owned files no longer desired.
// It rewrites the manifest with the new facts digest. Returns the paths whose
// content was created or updated. Unchanged files keep their bytes and mtime, so a
// change to one source domain never rewrites unrelated domain context.
func reconcile(repo, factsDigest string, desired []generated, existing Manifest, report *Report) ([]string, error) {
	existingByPath := map[string]string{}
	for _, file := range existing.Files {
		existingByPath[file.Path] = file.Digest
	}
	desiredPaths := map[string]bool{}
	for _, file := range desired {
		desiredPaths[file.Path] = true
	}
	for _, file := range existing.Files {
		if desiredPaths[file.Path] {
			continue
		}
		if err := removeOwned(repo, file); err != nil {
			report.Warnings = append(report.Warnings, "cannot remove stale artifact "+file.Path+": "+err.Error())
		}
	}

	written := []string{}
	manifest := Manifest{SchemaVersion: manifestSchemaVersion, FactsDigest: factsDigest}
	for _, file := range desired {
		digest := contentDigest(file.Content)
		if existingByPath[file.Path] != digest {
			if err := writeOwned(repo, file.Path, file.Content); err != nil {
				return nil, errors.New("cannot write artifact " + file.Path + ": " + err.Error())
			}
			written = append(written, file.Path)
		}
		manifest.Files = append(manifest.Files, ManifestFile{Path: file.Path, Digest: digest})
	}
	if err := writeManifest(repo, manifest); err != nil {
		return nil, errors.New("cannot write manifest: " + err.Error())
	}
	return written, nil
}

// Uninstall deletes only manifest-listed artifacts whose on-disk content still
// matches the recorded digest, then removes the manifest. It never touches files
// it did not generate. With dryRun it reports what would be removed.
func Uninstall(repo string, dryRun bool) Report {
	report := Report{Action: "uninstall", DryRun: dryRun, Status: "failed"}
	manifest, err := loadManifest(repo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.OK = true
			report.Status = statusForDryRun(dryRun)
			report.Installed = false
			return report
		}
		report.Errors = []string{"cannot read manifest: " + err.Error()}
		return report
	}
	report.FactsDigest = manifest.FactsDigest
	report.Installed = true
	for _, file := range manifest.Files {
		report.Planned = append(report.Planned, PlannedFile{Path: file.Path, Digest: file.Digest, Change: "remove"})
	}
	if dryRun {
		report.OK = true
		report.Status = "planned"
		return report
	}
	for _, file := range manifest.Files {
		if err := removeOwned(repo, file); err != nil {
			report.Warnings = append(report.Warnings, "cannot remove "+file.Path+": "+err.Error())
			continue
		}
		report.Files = append(report.Files, file.Path)
	}
	if err := os.Remove(manifestPath(repo)); err != nil && !errors.Is(err, os.ErrNotExist) {
		report.Errors = []string{"cannot remove manifest: " + err.Error()}
		return report
	}
	pruneOwnedDirs(repo)
	report.OK = true
	report.Status = "ok"
	report.Installed = false
	return report
}

// Status compares the current repository facts against the manifest and reports
// whether the generated context is present and fresh.
func Status(repo string) Report {
	report := Report{Action: "status", Status: "failed"}
	_, snapshot, err := Scan(repo)
	if err != nil {
		report.Errors = []string{"cannot scan repository: " + err.Error()}
		return report
	}
	report.FactsDigest = snapshot.Digest()
	manifest, err := loadManifest(repo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.OK = true
			report.Status = "not-installed"
			report.Installed = false
			return report
		}
		report.Errors = []string{"cannot read manifest: " + err.Error()}
		return report
	}
	report.Installed = true
	report.Fresh = manifest.FactsDigest == snapshot.Digest()
	report.OK = true
	if report.Fresh {
		report.Status = "fresh"
	} else {
		report.Status = "drift"
		report.Warnings = append(report.Warnings, "generated context is stale: repository facts changed since last generation")
	}
	return report
}

// Doctor verifies that every manifest-listed artifact exists and its on-disk
// content still matches the recorded digest, and reports facts drift. Read only.
func Doctor(repo string) Report {
	report := Report{Action: "doctor", Status: "failed"}
	_, snapshot, err := Scan(repo)
	if err != nil {
		report.Errors = []string{"cannot scan repository: " + err.Error()}
		return report
	}
	report.FactsDigest = snapshot.Digest()
	manifest, err := loadManifest(repo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.OK = true
			report.Status = "not-installed"
			return report
		}
		report.Errors = []string{"cannot read manifest: " + err.Error()}
		return report
	}
	report.Installed = true
	issues := []string{}
	for _, file := range manifest.Files {
		content, err := os.ReadFile(ownedPath(repo, file.Path))
		if err != nil {
			issues = append(issues, "missing generated artifact: "+file.Path)
			continue
		}
		if contentDigest(string(content)) != file.Digest {
			issues = append(issues, "generated artifact was modified outside the domain: "+file.Path)
		}
	}
	report.Fresh = manifest.FactsDigest == snapshot.Digest()
	if !report.Fresh {
		report.Warnings = append(report.Warnings, "generated context is stale; run lifecycle update --scope repo-context")
	}
	report.Errors = issues
	report.OK = len(issues) == 0
	report.Status = statusFromOK(report.OK)
	return report
}

// Verify checks the domain's structural contract: the facts snapshot builds and,
// when a manifest exists, every listed artifact is present. Read only.
func Verify(repo string) Report {
	report := Report{Action: "verify", Status: "failed"}
	_, snapshot, err := Scan(repo)
	if err != nil {
		report.Errors = []string{"cannot scan repository: " + err.Error()}
		return report
	}
	report.FactsDigest = snapshot.Digest()
	manifest, err := loadManifest(repo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.OK = true
			report.Status = "ok"
			return report
		}
		report.Errors = []string{"manifest is unreadable: " + err.Error()}
		return report
	}
	report.Installed = true
	issues := []string{}
	if manifest.SchemaVersion != manifestSchemaVersion {
		issues = append(issues, "unsupported manifest schemaVersion")
	}
	for _, file := range manifest.Files {
		if _, err := os.Stat(ownedPath(repo, file.Path)); err != nil {
			issues = append(issues, "manifest references missing artifact: "+file.Path)
		}
	}
	// Freshness is surfaced as a warning, not a structural failure: drift is
	// transient and auto-healed by the post-commit hook, so it must not fail the
	// aggregate verify gate. Integrity breaks above remain hard errors.
	report.Fresh = manifest.FactsDigest == snapshot.Digest()
	if !report.Fresh {
		report.Warnings = append(report.Warnings, "generated context is stale; run lifecycle update --scope repo-context")
	}
	report.Errors = issues
	report.OK = len(issues) == 0
	report.Status = statusFromOK(report.OK)
	return report
}

// Snapshot exposes the facts snapshot for callers that need the digest view.
func Snapshot(repo string) (registryobject.Snapshot, error) {
	_, snapshot, err := Scan(repo)
	return snapshot, err
}

func manifestMatches(manifest Manifest, desired []generated) bool {
	if len(manifest.Files) != len(desired) {
		return false
	}
	byPath := map[string]string{}
	for _, file := range manifest.Files {
		byPath[file.Path] = file.Digest
	}
	for _, file := range desired {
		if byPath[file.Path] != contentDigest(file.Content) {
			return false
		}
	}
	return true
}

func loadManifest(repo string) (Manifest, error) {
	content, err := os.ReadFile(manifestPath(repo))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, err
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	return manifest, nil
}

func writeManifest(repo string, manifest Manifest) error {
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return writeOwned(repo, manifestRel, string(content)+"\n")
}

func writeOwned(repo, rel, content string) error {
	full := ownedPath(repo, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

func removeOwned(repo string, file ManifestFile) error {
	full := ownedPath(repo, file.Path)
	content, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if contentDigest(string(content)) != file.Digest {
		return errors.New("content digest mismatch; refusing to delete unowned content")
	}
	return os.Remove(full)
}

func pruneOwnedDirs(repo string) {
	// Remove now-empty owned directories, deepest first. os.Remove only succeeds
	// on an empty directory, so this never deletes remaining content.
	os.Remove(ownedPath(repo, ownedRoot+"/domains"))
	os.Remove(ownedPath(repo, ownedRoot))
}

func manifestPath(repo string) string {
	return ownedPath(repo, manifestRel)
}

// ownedPath resolves a repo-relative artifact path (already prefixed with
// ownedRoot) to an absolute path.
func ownedPath(repo, rel string) string {
	return filepath.Join(repo, filepath.FromSlash(rel))
}

func statusForDryRun(dryRun bool) string {
	if dryRun {
		return "planned"
	}
	return "ok"
}

func statusFromOK(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}
