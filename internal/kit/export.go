package kit

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/runner"
)

type ExportOptions struct {
	Zip    bool `json:"zip"`
	DryRun bool `json:"dryRun"`
}

type PackageResult struct {
	OK                 bool                `json:"ok"`
	Status             string              `json:"status"`
	PackageFile        string              `json:"packageFile,omitempty"`
	ManifestFile       string              `json:"manifestFile,omitempty"`
	Sha256File         string              `json:"sha256File,omitempty"`
	Sha256             string              `json:"sha256,omitempty"`
	IncludedFilesCount int                 `json:"includedFilesCount"`
	ExcludedFilesCount int                 `json:"excludedFilesCount"`
	MissingIncludes    []string            `json:"missingIncludes,omitempty"`
	Files              []FileManifestEntry `json:"files,omitempty"`
}

type FileManifestEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type BundleManifest struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	GeneratedAt   time.Time           `json:"generatedAt"`
	GitCommit     string              `json:"gitCommit"`
	GitBranch     string              `json:"gitBranch"`
	PackageFile   string              `json:"packageFile"`
	Sha256        string              `json:"sha256"`
	Files         []FileManifestEntry `json:"files"`
}

func ExportKit(repo string, entry RegistryKit, manifest Manifest, command CommandDef, opts ExportOptions) (PackageResult, error) {
	include := command.Include
	if len(include) == 0 {
		return PackageResult{OK: false, Status: "failed"}, errorsf("%s export include is empty", entry.ID)
	}
	files, missing, excluded, err := collectPackageFiles(repo, include, command.Exclude)
	if err != nil {
		return PackageResult{OK: false, Status: "failed"}, err
	}
	result := PackageResult{OK: len(missing) == 0, Status: "ok", IncludedFilesCount: len(files), ExcludedFilesCount: len(excluded), MissingIncludes: missing}
	if len(missing) > 0 {
		result.Status = "missing"
		return result, errorsf("export include paths did not match: %s", strings.Join(missing, ", "))
	}
	if opts.DryRun {
		result.Status = "dry-run"
		return result, nil
	}
	outName := resolveTokens(command.OutputName, entry.ID, manifest.Version)
	if outName == "" {
		outName = entry.ID + "-" + manifest.Version + ".zip"
	}
	out := platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "packages", outName)))
	if err := writeZip(repo, out, files); err != nil {
		return result, err
	}
	sha, err := fileSHA256(out)
	if err != nil {
		return result, err
	}
	shaFile := out + ".sha256"
	if err := os.WriteFile(shaFile, []byte(fmt.Sprintf("%s  %s\n", sha, filepath.Base(out))), 0o644); err != nil {
		return result, err
	}
	result.PackageFile = out
	result.Sha256File = shaFile
	result.Sha256 = sha
	return result, nil
}

// ValidateExportCommand reuses the real dry-run collector without creating a ZIP.
func ValidateExportCommand(repo string, entry RegistryKit, manifest Manifest, command CommandDef) error {
	if _, err := ExportKit(repo, entry, manifest, command, ExportOptions{DryRun: true}); err != nil {
		return err
	}
	output := resolveTokens(command.OutputName, entry.ID, manifest.Version)
	if output == "" {
		output = entry.ID + "-" + manifest.Version + ".zip"
	}
	if strings.Contains(output, "${") {
		return errorsf("%s export outputName has an unresolved token: %s", entry.ID, output)
	}
	return nil
}

func ExportBundle(repo string, version string) (PackageResult, error) {
	if version == "" {
		version = readConfigVersion(repo)
	}
	if version == "" {
		version = "0.1.0"
	}
	files, excluded, err := collectRepoFiles(repo)
	if err != nil {
		return PackageResult{}, err
	}
	name := fmt.Sprintf("aicoding-kit-%s.zip", version)
	out := platform.RepoPath(repo, filepath.ToSlash(filepath.Join("dist", name)))
	manifestFile := out + ".manifest.json"
	shaFile := out + ".sha256"
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return PackageResult{}, err
	}
	if err := writeZip(repo, out, files); err != nil {
		return PackageResult{}, err
	}
	sha, err := fileSHA256(out)
	if err != nil {
		return PackageResult{}, err
	}
	entries, err := fileManifestEntries(repo, files)
	if err != nil {
		return PackageResult{}, err
	}
	commit, _ := gitx.Run(repo, "rev-parse", "HEAD")
	branch, _ := gitx.Run(repo, "branch", "--show-current")
	manifest := BundleManifest{SchemaVersion: 1, Name: "AiCoding", Version: version, GeneratedAt: time.Now().UTC(), GitCommit: strings.TrimSpace(commit), GitBranch: strings.TrimSpace(branch), PackageFile: filepath.Base(out), Sha256: sha, Files: entries}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return PackageResult{}, err
	}
	if err := os.WriteFile(manifestFile, append(b, '\n'), 0o644); err != nil {
		return PackageResult{}, err
	}
	if err := os.WriteFile(shaFile, []byte(fmt.Sprintf("%s  %s\n", sha, filepath.Base(out))), 0o644); err != nil {
		return PackageResult{}, err
	}
	return PackageResult{OK: true, Status: "ok", PackageFile: out, ManifestFile: manifestFile, Sha256File: shaFile, Sha256: sha, IncludedFilesCount: len(files), ExcludedFilesCount: len(excluded), Files: entries}, nil
}

func collectRepoFiles(repo string) ([]string, []string, error) {
	out, err := gitx.Run(repo, "-c", "core.quotePath=false", "ls-files")
	if err != nil {
		return nil, nil, err
	}
	files := []string{}
	excluded := []string{}
	for _, line := range strings.Split(out, "\n") {
		rel := filepath.ToSlash(strings.TrimSpace(line))
		if rel == "" {
			continue
		}
		if info, statErr := os.Stat(platform.RepoPath(repo, rel)); statErr != nil || info.IsDir() {
			excluded = append(excluded, rel)
			continue
		}
		if shouldExcludeBundlePath(rel, false) {
			excluded = append(excluded, rel)
			continue
		}
		files = append(files, rel)
	}
	sort.Strings(files)
	sort.Strings(excluded)
	return files, excluded, nil
}

func shouldExcludeBundlePath(rel string, isDir bool) bool {
	lower := strings.ToLower(filepath.ToSlash(rel))
	prefixes := []string{".git", "bin", ".aicoding/packages", ".aicoding/state", ".aicoding/tmp", ".aicoding/cache", ".aicoding/reports", ".agentpatch", ".ai-debug-repair", ".pytest_cache", "node_modules", ".venv", "venv"}
	for _, p := range prefixes {
		if lower == p || strings.HasPrefix(lower, p+"/") {
			return true
		}
	}
	if strings.HasPrefix(lower, "dist/aicoding-kit-") && (strings.HasSuffix(lower, ".zip") || strings.Contains(lower, ".zip.")) {
		return true
	}
	if strings.Contains(lower, "/__pycache__/") || strings.HasSuffix(lower, "/__pycache__") {
		return true
	}
	return false
}

func collectPackageFiles(repo string, include, exclude []string) ([]string, []string, []string, error) {
	set := map[string]bool{}
	missing := []string{}
	excluded := []string{}
	all, _, err := collectRepoFiles(repo)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, pattern := range include {
		matched := false
		for _, rel := range all {
			if globMatch(pattern, rel) {
				set[rel] = true
				matched = true
			}
		}
		if !matched {
			missing = append(missing, pattern)
		}
	}
	files := []string{}
	for rel := range set {
		isExcluded := false
		for _, pattern := range exclude {
			if globMatch(pattern, rel) {
				isExcluded = true
				break
			}
		}
		if isExcluded {
			excluded = append(excluded, rel)
		} else {
			files = append(files, rel)
		}
	}
	sort.Strings(files)
	sort.Strings(missing)
	sort.Strings(excluded)
	return files, missing, excluded, nil
}

func globMatch(pattern, rel string) bool {
	pattern = strings.TrimPrefix(filepath.ToSlash(pattern), "./")
	rel = filepath.ToSlash(rel)
	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		return rel == base || strings.HasPrefix(rel, base+"/")
	}
	if !strings.ContainsAny(pattern, "*?") {
		return rel == pattern || strings.HasPrefix(rel, strings.TrimSuffix(pattern, "/")+"/")
	}
	regex := globRegex(pattern)
	matched, _ := filepath.Match(regex, rel)
	if matched {
		return true
	}
	return simpleGlob(pattern, rel)
}

func simpleGlob(pattern, rel string) bool {
	parts := strings.Split(pattern, "**")
	pos := 0
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part == "" {
			continue
		}
		idx := strings.Index(rel[pos:], part)
		if idx < 0 {
			return false
		}
		pos += idx + len(part)
	}
	if strings.HasPrefix(pattern, "**/") {
		return true
	}
	return strings.HasPrefix(rel, strings.TrimSuffix(parts[0], "/"))
}

func globRegex(pattern string) string { return strings.ReplaceAll(pattern, "**", "*") }

func writeZip(repo, out string, rels []string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", out, time.Now().UnixNano())
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	for _, rel := range rels {
		full := platform.RepoPath(repo, rel)
		info, err := os.Stat(full)
		if err != nil {
			zw.Close()
			f.Close()
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			zw.Close()
			f.Close()
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		w, err := zw.CreateHeader(header)
		if err != nil {
			zw.Close()
			f.Close()
			return err
		}
		in, err := os.Open(full)
		if err != nil {
			zw.Close()
			f.Close()
			return err
		}
		_, copyErr := io.Copy(w, in)
		closeErr := in.Close()
		if copyErr != nil {
			zw.Close()
			f.Close()
			return copyErr
		}
		if closeErr != nil {
			zw.Close()
			f.Close()
			return closeErr
		}
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_ = os.Remove(out)
	return os.Rename(tmp, out)
}

func fileManifestEntries(repo string, rels []string) ([]FileManifestEntry, error) {
	tasks := make([]runner.Task, 0, len(rels))
	for _, rel := range rels {
		rel := rel
		tasks = append(tasks, runner.Task{
			ID:    rel,
			Group: "export-manifest",
			Run: func(context.Context) runner.TaskResult {
				full := platform.RepoPath(repo, rel)
				info, err := os.Stat(full)
				if err != nil {
					return runner.TaskResult{OK: false, Errors: []string{err.Error()}}
				}
				sha, err := fileSHA256(full)
				if err != nil {
					return runner.TaskResult{OK: false, Errors: []string{err.Error()}}
				}
				return runner.TaskResult{OK: true, Data: FileManifestEntry{Path: rel, Size: info.Size(), SHA256: sha}}
			},
		})
	}

	entries := make([]FileManifestEntry, 0, len(rels))
	errs := []string{}
	for _, result := range runner.Run(context.Background(), tasks, runner.Options{}) {
		if !result.OK {
			for _, errText := range result.Errors {
				errs = append(errs, result.ID+": "+errText)
			}
			continue
		}
		entry, ok := result.Data.(FileManifestEntry)
		if !ok {
			errs = append(errs, result.ID+": invalid export manifest result")
			continue
		}
		entries = append(entries, entry)
	}
	if len(errs) > 0 {
		return nil, errorsf(strings.Join(errs, "; "))
	}
	return entries, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readConfigVersion(repo string) string {
	b, err := os.ReadFile(platform.RepoPath(repo, "config/codex-kit.json"))
	if err != nil {
		return ""
	}
	var cfg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return ""
	}
	return cfg.Version
}

func resolveTokens(value, id, version string) string {
	value = strings.ReplaceAll(value, "${kitId}", id)
	value = strings.ReplaceAll(value, "${version}", version)
	return value
}
func errorsf(format string, args ...interface{}) error { return fmt.Errorf(format, args...) }
