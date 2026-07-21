package validationevidence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

type configEntry struct {
	Path    string `json:"path"`
	Digest  string `json:"digest,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}

type executableFingerprint struct {
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	ModTimeUnixNano int64  `json:"modTimeUnixNano"`
}

type toolchainCache struct {
	SchemaVersion    int                   `json:"schemaVersion"`
	SearchPathDigest string                `json:"searchPathDigest"`
	Go               executableFingerprint `json:"go"`
	Git              executableFingerprint `json:"git"`
	GoVersion        string                `json:"goVersion"`
	GitVersion       string                `json:"gitVersion"`
	Digest           string                `json:"digest"`
	Integrity        string                `json:"integrity"`
}

// Fingerprint computes one validation identity without hashing repository files
// beyond the explicitly supplied config paths.
func (r Repository) Fingerprint(subject Subject, spec FingerprintSpec) (Fingerprint, error) {
	if !validTreeOID(subject.TreeOID) {
		return Fingerprint{}, fingerprintError("subject tree OID is invalid")
	}
	profile := strings.ToLower(strings.TrimSpace(spec.Profile))
	if !validProfile(profile) {
		return Fingerprint{}, fingerprintError("profile is invalid")
	}
	for name, value := range map[string]string{
		"validation plan": spec.ValidationPlanDigest,
		"engine semantic": spec.EngineSemanticDigest,
		"options":         spec.OptionsDigest,
	} {
		if !validDigest(value) {
			return Fingerprint{}, fingerprintError(name + " digest is invalid")
		}
	}
	configDigest, err := r.configDigest(spec.ConfigPaths)
	if err != nil {
		return Fingerprint{}, err
	}
	toolchainDigest, err := r.toolchainDigest()
	if err != nil {
		return Fingerprint{}, err
	}
	fingerprint := Fingerprint{
		RepositoryID:         r.repositoryID,
		SubjectTreeOID:       subject.TreeOID,
		Profile:              profile,
		ValidationPlanDigest: spec.ValidationPlanDigest,
		EngineSemanticDigest: spec.EngineSemanticDigest,
		ConfigDigest:         configDigest,
		ToolchainDigest:      toolchainDigest,
		OptionsDigest:        spec.OptionsDigest,
	}
	payload, err := json.Marshal(fingerprint)
	if err != nil {
		return Fingerprint{}, fingerprintError(err.Error())
	}
	fingerprint.Identity = digestBytes(payload)
	return fingerprint, nil
}

// DeriveNodeFingerprint replaces the whole-tree identity input with one
// testengine-owned node digest while preserving all validation semantics.
func (r Repository) DeriveNodeFingerprint(base Fingerprint, node, inputDigest string) (Fingerprint, error) {
	node = strings.ToLower(strings.TrimSpace(node))
	if base.RepositoryID != r.repositoryID || base.Node != "" || !validFingerprint(base) {
		return Fingerprint{}, fingerprintError("base fingerprint is not a whole-tree identity for this repository")
	}
	if !validNodeName(node) {
		return Fingerprint{}, fingerprintError("node name is invalid")
	}
	if !validDigest(inputDigest) {
		return Fingerprint{}, fingerprintError("node input digest is invalid")
	}
	fingerprint := base
	fingerprint.Identity = ""
	fingerprint.SubjectTreeOID = ""
	fingerprint.Node = node
	fingerprint.NodeInputDigest = inputDigest
	payload, err := json.Marshal(fingerprint)
	if err != nil {
		return Fingerprint{}, fingerprintError(err.Error())
	}
	fingerprint.Identity = digestBytes(payload)
	return fingerprint, nil
}

func (r Repository) configDigest(paths []string) (string, error) {
	entries := make([]configEntry, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, configured := range paths {
		abs, rel, err := r.resolveConfigPath(configured)
		if err != nil {
			return "", err
		}
		if _, exists := seen[rel]; exists {
			continue
		}
		seen[rel] = struct{}{}
		content, err := os.ReadFile(abs)
		if os.IsNotExist(err) {
			entries = append(entries, configEntry{Path: rel, Missing: true})
			continue
		}
		if err != nil {
			return "", fingerprintError(fmt.Sprintf("read config %s: %v", rel, err))
		}
		entries = append(entries, configEntry{Path: rel, Digest: digestBytes(content)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	payload, err := json.Marshal(entries)
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	return digestBytes(payload), nil
}

func (r Repository) resolveConfigPath(configured string) (string, string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return "", "", fingerprintError("config path is empty")
	}
	abs := configured
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(r.repo, filepath.FromSlash(configured))
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", "", fingerprintError(err.Error())
	}
	rel, err := filepath.Rel(r.repo, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fingerprintError(fmt.Sprintf("config path escapes repository: %s", configured))
	}
	return abs, filepath.ToSlash(rel), nil
}

func (r Repository) toolchainDigest() (string, error) {
	cachePath := filepath.Join(r.root, "toolchain.json")
	searchPathDigest := digestBytes([]byte(os.Getenv("PATH")))
	if cached, err := readToolchainCache(cachePath); err == nil && cached.SearchPathDigest == searchPathDigest {
		goFingerprint, goErr := fingerprintExecutablePath(cached.Go.Path)
		gitFingerprint, gitErr := fingerprintExecutablePath(cached.Git.Path)
		if goErr == nil && gitErr == nil && cached.Go == goFingerprint && cached.Git == gitFingerprint {
			return cached.Digest, nil
		}
	}
	goFingerprint, err := fingerprintExecutable("go")
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	gitFingerprint, err := fingerprintExecutable("git")
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	goOutput, err := exec.Command(goFingerprint.Path, "version").CombinedOutput()
	if err != nil {
		return "", fingerprintError(fmt.Sprintf("go version: %v", err))
	}
	gitOutput, err := gitx.Run("", "--version")
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	cache := toolchainCache{
		SchemaVersion:    toolchainSchemaVersion,
		SearchPathDigest: searchPathDigest,
		Go:               goFingerprint,
		Git:              gitFingerprint,
		GoVersion:        strings.TrimSpace(string(goOutput)),
		GitVersion:       strings.TrimSpace(gitOutput),
	}
	cache.Digest = toolchainDigest(cache)
	cache.Integrity = toolchainIntegrity(cache)
	encoded, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	encoded = append(encoded, '\n')
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return "", &Error{Code: CodeStoreError, Message: fmt.Sprintf("replace toolchain cache: %v", err), RequiredAction: "check Git common-dir permissions"}
	}
	if err := atomicWriteFile(cachePath, encoded); err != nil {
		return "", &Error{Code: CodeStoreError, Message: fmt.Sprintf("write toolchain cache: %v", err), RequiredAction: "check Git common-dir permissions"}
	}
	return cache.Digest, nil
}

func fingerprintExecutable(name string) (executableFingerprint, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return executableFingerprint{}, fmt.Errorf("locate %s: %w", name, err)
	}
	return fingerprintExecutablePath(path)
}

func fingerprintExecutablePath(path string) (executableFingerprint, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return executableFingerprint{}, fmt.Errorf("resolve executable: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return executableFingerprint{}, fmt.Errorf("stat executable: %w", err)
	}
	return executableFingerprint{Path: filepath.Clean(path), Size: info.Size(), ModTimeUnixNano: info.ModTime().UnixNano()}, nil
}

func readToolchainCache(path string) (toolchainCache, error) {
	var cache toolchainCache
	raw, err := os.ReadFile(path)
	if err != nil {
		return cache, err
	}
	if err := json.Unmarshal(raw, &cache); err != nil {
		return cache, err
	}
	if cache.SchemaVersion != toolchainSchemaVersion || !validDigest(cache.SearchPathDigest) || !validDigest(cache.Digest) || cache.Integrity != toolchainIntegrity(cache) || cache.Digest != toolchainDigest(cache) {
		return toolchainCache{}, fmt.Errorf("invalid toolchain cache")
	}
	return cache, nil
}

func toolchainDigest(cache toolchainCache) string {
	payload, _ := json.Marshal(struct {
		Go         executableFingerprint `json:"go"`
		Git        executableFingerprint `json:"git"`
		GoVersion  string                `json:"goVersion"`
		GitVersion string                `json:"gitVersion"`
	}{cache.Go, cache.Git, cache.GoVersion, cache.GitVersion})
	return digestBytes(payload)
}

func toolchainIntegrity(cache toolchainCache) string {
	payload, _ := json.Marshal(struct {
		SchemaVersion    int                   `json:"schemaVersion"`
		SearchPathDigest string                `json:"searchPathDigest"`
		Go               executableFingerprint `json:"go"`
		Git              executableFingerprint `json:"git"`
		GoVersion        string                `json:"goVersion"`
		GitVersion       string                `json:"gitVersion"`
		Digest           string                `json:"digest"`
	}{cache.SchemaVersion, cache.SearchPathDigest, cache.Go, cache.Git, cache.GoVersion, cache.GitVersion, cache.Digest})
	return digestBytes(payload)
}

func fingerprintError(message string) *Error {
	return &Error{Code: CodeFingerprintInvalid, Message: message, RequiredAction: "recompute validation evidence from current repository inputs"}
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", sum)
}
