package kit

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

const freshCloneBaselineFile = "fresh-clone-baseline"

type SourceManifest struct {
	SchemaVersion       int               `json:"schemaVersion"`
	SourceMode          string            `json:"sourceMode"`
	SourceIdentity      string            `json:"sourceIdentity"`
	SuperprojectTreeOID string            `json:"superprojectTreeOID"`
	Submodules          []SourceSubmodule `json:"submodules"`
	FileCount           int               `json:"fileCount"`
}

type SourceSubmodule struct {
	Path      string `json:"path"`
	CommitOID string `json:"commitOID"`
	TreeOID   string `json:"treeOID"`
}

type materializedEntry struct {
	Mode string
	OID  string
}

type materializationState struct {
	entries    map[string]materializedEntry
	submodules []SourceSubmodule
}

// MaterializeRelease verifies the Release leaf against a Git-object-only
// source tree. It never copies worktree files and never performs network I/O.
func MaterializeRelease(ctx context.Context, repo, treeOID string) (report FreshCloneReport) {
	report = FreshCloneReport{
		SchemaVersion: 1,
		Profile:       "Release",
		OK:            true,
		SourceMode:    "materialized",
		SourceRoot:    repo,
		SourceTreeOID: strings.ToLower(strings.TrimSpace(treeOID)),
	}
	add := func(name string, started time.Time, ok bool, message, output string) {
		report.Steps = append(report.Steps, FreshCloneStep{
			Name: name, OK: ok, Message: message, Output: trimOutput(output), ElapsedMS: time.Since(started).Milliseconds(),
		})
		if !ok {
			report.OK = false
			report.Errors = append(report.Errors, name+": "+message)
		}
	}

	started := time.Now()
	tempRoot, err := platform.CreateTempDir(repo, "materialize")
	report.TempRoot = tempRoot
	report.MaterializedRoot = filepath.Join(tempRoot, "source")
	report.ManifestPath = filepath.Join(tempRoot, "source-manifest.json")
	if err != nil {
		add("temp", started, false, err.Error(), "")
		return report
	}
	add("temp", started, true, "created and registered temporary directory", "")
	defer func() {
		cleanupStarted := time.Now()
		if report.OK {
			if err := platform.ReleaseTempDir(repo, tempRoot, "materialize"); err != nil {
				report.KeptTemp = true
				add("temp.release", cleanupStarted, false, err.Error(), "")
				return
			}
			add("temp.release", cleanupStarted, true, "released and recorded temporary directory", "")
			return
		}
		report.KeptTemp = true
		if err := platform.RecordTempOutcome(repo, tempRoot, "materialize", "failed"); err != nil {
			add("temp.ledger", cleanupStarted, false, err.Error(), "")
			return
		}
		add("temp.ledger", cleanupStarted, true, "failed evidence retained and registered", "")
	}()

	started = time.Now()
	manifest, err := materializeSource(ctx, repo, treeOID, report.MaterializedRoot, report.ManifestPath)
	if err != nil {
		add("source.materialize", started, false, err.Error(), "")
		return report
	}
	report.SourceTreeOID = manifest.SuperprojectTreeOID
	report.SourceManifest = &manifest
	add("source.materialize", started, true, fmt.Sprintf("materialized %d tracked files without clone", manifest.FileCount), "")

	bin := filepath.Join(report.MaterializedRoot, "bin", "aicoding.exe")
	started = time.Now()
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		add("go.build.mkdir", started, false, err.Error(), "")
		return report
	}
	started = time.Now()
	if out, err := runFreshContext(ctx, report.MaterializedRoot, "go", "build", "-o", bin, "./cmd/aicoding"); err != nil {
		add("go.build", started, false, err.Error(), out)
		return report
	} else {
		add("go.build", started, true, "built Go CLI", out)
	}
	checks, err := freshCloneChecks(bin, report.Profile)
	if err != nil {
		add("profile", time.Now(), false, err.Error(), "")
		return report
	}
	for _, check := range checks {
		if len(check) >= 3 && check[0] == bin && check[1] == "release" && check[2] == "verify" {
			check = append(append([]string{}, check...), "--repo-root", report.MaterializedRoot)
		}
		started = time.Now()
		out, err := runFreshContext(ctx, report.MaterializedRoot, check[0], check[1:]...)
		name := "check." + filepath.Base(check[0]) + " " + strings.Join(check[1:], " ")
		if err != nil {
			add(name, started, false, err.Error(), out)
			return report
		}
		add(name, started, true, "passed", out)
	}
	return report
}

func materializeSource(ctx context.Context, repo, treeOID, sourceRoot, manifestPath string) (SourceManifest, error) {
	resolvedTree, err := gitx.TreeOID(repo, treeOID)
	if err != nil {
		return SourceManifest{}, fmt.Errorf("resolve materialization tree: %w", err)
	}
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		return SourceManifest{}, fmt.Errorf("create materialized source root: %w", err)
	}
	state := materializationState{entries: map[string]materializedEntry{}}
	if err := materializeRepository(ctx, repo, resolvedTree, "", sourceRoot, &state); err != nil {
		return SourceManifest{}, err
	}
	sort.Slice(state.submodules, func(i, j int) bool { return state.submodules[i].Path < state.submodules[j].Path })
	if err := validateMaterializedFiles(sourceRoot, state.entries); err != nil {
		return SourceManifest{}, err
	}
	manifest := SourceManifest{
		SchemaVersion: 1, SourceMode: "materialized", SuperprojectTreeOID: resolvedTree,
		Submodules: state.submodules, FileCount: len(state.entries),
	}
	manifest.SourceIdentity, err = sourceManifestIdentity(manifest)
	if err != nil {
		return SourceManifest{}, err
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return SourceManifest{}, fmt.Errorf("encode source manifest: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(manifestPath, payload, 0o600); err != nil {
		return SourceManifest{}, fmt.Errorf("write source manifest: %w", err)
	}
	return manifest, nil
}

func materializeRepository(ctx context.Context, objectRepo, rev, prefix, destination string, state *materializationState) error {
	treeOID, err := gitx.TreeOID(objectRepo, rev)
	if err != nil {
		return fmt.Errorf("resolve tree for %s: %w", displaySourcePath(prefix), err)
	}
	entries, err := gitx.TreeEntries(objectRepo, treeOID)
	if err != nil {
		return fmt.Errorf("list tree for %s: %w", displaySourcePath(prefix), err)
	}
	if err := extractGitArchive(ctx, objectRepo, rev, destination); err != nil {
		return fmt.Errorf("archive %s: %w", displaySourcePath(prefix), err)
	}
	for _, entry := range entries {
		materializedPath := path.Join(prefix, filepath.ToSlash(entry.Path))
		switch entry.Type {
		case "blob":
			state.entries[materializedPath] = materializedEntry{Mode: entry.Mode, OID: entry.OID}
		case "commit":
			submoduleRepo := filepath.Join(objectRepo, filepath.FromSlash(entry.Path))
			submoduleDestination := filepath.Join(destination, filepath.FromSlash(entry.Path))
			submoduleTree, err := gitx.TreeOID(submoduleRepo, entry.OID)
			if err != nil {
				return fmt.Errorf("resolve initialized submodule %s at %s: %w", materializedPath, entry.OID, err)
			}
			state.submodules = append(state.submodules, SourceSubmodule{
				Path: materializedPath, CommitOID: entry.OID, TreeOID: submoduleTree,
			})
			if err := os.MkdirAll(submoduleDestination, 0o755); err != nil {
				return fmt.Errorf("create submodule destination %s: %w", materializedPath, err)
			}
			if err := materializeRepository(ctx, submoduleRepo, entry.OID, materializedPath, submoduleDestination, state); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported Git tree entry type %q at %s", entry.Type, materializedPath)
		}
	}
	return nil
}

func extractGitArchive(ctx context.Context, repo, rev, destination string) error {
	reader, writer := io.Pipe()
	archiveDone := make(chan error, 1)
	go func() {
		err := gitx.Archive(ctx, repo, rev, writer)
		_ = writer.CloseWithError(err)
		archiveDone <- err
	}()
	extractErr := extractTarStream(reader, destination)
	if extractErr != nil {
		_ = reader.CloseWithError(extractErr)
		<-archiveDone
		return fmt.Errorf("tar extract: %w", extractErr)
	}
	_, drainErr := io.Copy(io.Discard, reader)
	_ = reader.Close()
	archiveErr := <-archiveDone
	if drainErr != nil {
		return fmt.Errorf("drain tar stream: %w", drainErr)
	}
	if archiveErr != nil {
		return archiveErr
	}
	return nil
}

func extractTarStream(reader io.Reader, destination string) error {
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read header: %w", err)
		}
		if header.Typeflag == tar.TypeXHeader || header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		target, err := tarTarget(destination, header.Name)
		if err != nil {
			return err
		}
		mode := os.FileMode(header.Mode) & os.ModePerm
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, mode); err != nil {
				return fmt.Errorf("create directory %s: %w", header.Name, err)
			}
		case tar.TypeReg, 0:
			if err := writeTarFile(target, mode, archive); err != nil {
				return fmt.Errorf("write file %s: %w", header.Name, err)
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create symlink parent %s: %w", header.Name, err)
			}
			if runtime.GOOS == "windows" {
				if err := os.WriteFile(target, []byte(header.Linkname), mode); err != nil {
					return fmt.Errorf("write symlink payload %s: %w", header.Name, err)
				}
			} else if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("create symlink %s: %w", header.Name, err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type %d at %s", header.Typeflag, header.Name)
		}
	}
}

func tarTarget(destination, name string) (string, error) {
	clean := path.Clean(name)
	if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe tar path %q", name)
	}
	target := filepath.Join(destination, filepath.FromSlash(clean))
	relative, err := filepath.Rel(destination, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("tar path escapes destination %q", name)
	}
	return target, nil
}

func writeTarFile(target string, mode os.FileMode, reader io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(target, mode)
	}
	return nil
}

func validateMaterializedFiles(sourceRoot string, expected map[string]materializedEntry) error {
	actual := map[string]os.FileInfo{}
	err := filepath.Walk(sourceRoot, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceRoot, current)
		if err != nil {
			return err
		}
		actual[filepath.ToSlash(rel)] = info
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk materialized source: %w", err)
	}
	if len(actual) != len(expected) {
		return fmt.Errorf("materialized file count mismatch: got %d want %d", len(actual), len(expected))
	}
	for rel, entry := range expected {
		info, exists := actual[rel]
		if !exists {
			return fmt.Errorf("materialized source is missing tracked file %s", rel)
		}
		if runtime.GOOS != "windows" {
			executable := info.Mode().Perm()&0o111 != 0
			wantExecutable := entry.Mode == "100755"
			if executable != wantExecutable {
				return fmt.Errorf("materialized mode mismatch for %s: got %s want %s", rel, info.Mode().Perm(), entry.Mode)
			}
		}
	}
	return nil
}

func sourceManifestIdentity(manifest SourceManifest) (string, error) {
	payload, err := json.Marshal(struct {
		SuperprojectTreeOID string            `json:"superprojectTreeOID"`
		Submodules          []SourceSubmodule `json:"submodules"`
	}{manifest.SuperprojectTreeOID, manifest.Submodules})
	if err != nil {
		return "", fmt.Errorf("encode source identity: %w", err)
	}
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("sha256:%x", digest), nil
}

func runFreshContext(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

func displaySourcePath(prefix string) string {
	if prefix == "" {
		return "superproject"
	}
	return prefix
}

func recordFreshCloneBaseline(repo, treeOID string) error {
	treeOID, err := gitx.TreeOID(repo, treeOID)
	if err != nil {
		return err
	}
	commonDir, err := gitx.CommonDir(repo)
	if err != nil {
		return err
	}
	directory := filepath.Join(commonDir, "aicoding")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, freshCloneBaselineFile), []byte(treeOID+"\n"), 0o600)
}

func readFreshCloneBaseline(repo string) (string, error) {
	commonDir, err := gitx.CommonDir(repo)
	if err != nil {
		return "", err
	}
	payload, err := os.ReadFile(filepath.Join(commonDir, "aicoding", freshCloneBaselineFile))
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no successful fresh-clone baseline; run bin/aicoding.exe fresh-clone --profile Release --json")
	}
	if err != nil {
		return "", err
	}
	baseline := strings.TrimSpace(string(payload))
	if baseline == "" || strings.ContainsAny(baseline, " \t\r\n") {
		return "", fmt.Errorf("fresh-clone baseline is invalid; rerun explicit fresh-clone")
	}
	return baseline, nil
}

// CheckFreshCloneTransportDrift returns an advisory error when transport or
// bootstrap paths changed after the last successful explicit fresh-clone.
func CheckFreshCloneTransportDrift(repo, targetTree string) error {
	baseline, err := readFreshCloneBaseline(repo)
	if err != nil {
		return fmt.Errorf("FRESH-004: %w", err)
	}
	targetTree, err = gitx.TreeOID(repo, targetTree)
	if err != nil {
		return fmt.Errorf("FRESH-004: resolve target tree: %w", err)
	}
	changed, err := gitx.DiffTreeFiles(repo, baseline, targetTree)
	if err != nil {
		return fmt.Errorf("FRESH-004: compare fresh-clone baseline: %w", err)
	}
	working, err := workingTreePaths(repo)
	if err != nil {
		return fmt.Errorf("FRESH-004: inspect working tree: %w", err)
	}
	changed = append(changed, working...)
	seen := map[string]bool{}
	sensitive := []string{}
	for _, candidate := range changed {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		if candidate == "" || seen[candidate] || !isTransportSensitivePath(candidate) {
			continue
		}
		seen[candidate] = true
		sensitive = append(sensitive, candidate)
	}
	sort.Strings(sensitive)
	if len(sensitive) > 0 {
		return fmt.Errorf("FRESH-004: transport-sensitive paths changed since last successful fresh-clone: %s; run bin/aicoding.exe fresh-clone --profile Release --json", strings.Join(sensitive, ", "))
	}
	return nil
}

func workingTreePaths(repo string) ([]string, error) {
	paths := []string{}
	for _, args := range [][]string{
		{"diff", "--name-only", "-z", "--"},
		{"ls-files", "--others", "--exclude-standard", "-z", "--"},
	} {
		output, err := gitx.Run(repo, args...)
		if err != nil {
			return nil, err
		}
		for _, item := range strings.Split(output, "\x00") {
			if item != "" {
				paths = append(paths, filepath.ToSlash(item))
			}
		}
	}
	return paths, nil
}

func isTransportSensitivePath(candidate string) bool {
	candidate = strings.ToLower(filepath.ToSlash(candidate))
	return candidate == ".gitmodules" ||
		candidate == ".gitattributes" ||
		candidate == "taskfile.yml" ||
		candidate == "cmd/aicoding/main.go" ||
		strings.HasPrefix(candidate, ".githooks/") ||
		strings.HasPrefix(candidate, "internal/bootstrap/") ||
		strings.HasPrefix(candidate, "internal/repoinit/")
}
