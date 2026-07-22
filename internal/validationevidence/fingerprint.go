package validationevidence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

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

type toolchainSemantic struct {
	Domain       string `json:"domain"`
	Version      int    `json:"version"`
	GoVersion    string `json:"goVersion"`
	GitVersion   string `json:"gitVersion"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
}

type toolchainCache struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Go            executableFingerprint `json:"go"`
	Git           executableFingerprint `json:"git"`
	Semantic      toolchainSemantic     `json:"semantic"`
	Digest        string                `json:"digest"`
	Integrity     string                `json:"integrity"`
}

type toolchainProbe struct {
	Platform     string
	Architecture string
	Locate       func(string) (executableFingerprint, error)
	Version      func(string, executableFingerprint) (string, error)
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
	return r.toolchainDigestWith(systemToolchainProbe())
}

func systemToolchainProbe() toolchainProbe {
	return toolchainProbe{
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Locate:       fingerprintExecutable,
		Version: func(name string, executable executableFingerprint) (string, error) {
			switch name {
			case "go":
				output, err := exec.Command(executable.Path, "version").CombinedOutput()
				if err != nil {
					return "", fmt.Errorf("go version: %w: %s", err, strings.TrimSpace(string(output)))
				}
				return string(output), nil
			case "git":
				output, err := gitx.Run("", "--version")
				if err != nil {
					return "", err
				}
				return output, nil
			default:
				return "", fmt.Errorf("unsupported toolchain probe %q", name)
			}
		},
	}
}

func (r Repository) toolchainDigestWith(probe toolchainProbe) (string, error) {
	cachePath := filepath.Join(r.root, "toolchain.json")
	platform, architecture, err := normalizeToolchainPlatform(probe.Platform, probe.Architecture)
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	if probe.Locate == nil || probe.Version == nil {
		return "", fingerprintError("toolchain probe is incomplete")
	}
	goFingerprint, err := probe.Locate("go")
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	gitFingerprint, err := probe.Locate("git")
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	if cached, readErr := readToolchainCache(cachePath); readErr == nil {
		if cached.Go == goFingerprint && cached.Git == gitFingerprint &&
			cached.Semantic.Platform == platform && cached.Semantic.Architecture == architecture {
			return cached.Digest, nil
		}
	}
	goOutput, err := probe.Version("go", goFingerprint)
	if err != nil {
		return "", fingerprintError(fmt.Sprintf("probe go version: %v", err))
	}
	goVersion, err := normalizeToolVersion("go", goOutput)
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	gitOutput, err := probe.Version("git", gitFingerprint)
	if err != nil {
		return "", fingerprintError(fmt.Sprintf("probe git version: %v", err))
	}
	gitVersion, err := normalizeToolVersion("git", gitOutput)
	if err != nil {
		return "", fingerprintError(err.Error())
	}
	cache := toolchainCache{
		SchemaVersion: toolchainSchemaVersion,
		Go:            goFingerprint,
		Git:           gitFingerprint,
		Semantic: toolchainSemantic{
			Domain: toolchainDigestDomain, Version: toolchainDigestVersion,
			GoVersion: goVersion, GitVersion: gitVersion,
			Platform: platform, Architecture: architecture,
		},
	}
	cache.Digest = semanticToolchainDigest(cache.Semantic)
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
	if cache.SchemaVersion != toolchainSchemaVersion || !validExecutableFingerprint(cache.Go) || !validExecutableFingerprint(cache.Git) ||
		!validToolchainSemantic(cache.Semantic) || !validDigest(cache.Digest) || cache.Integrity != toolchainIntegrity(cache) ||
		cache.Digest != semanticToolchainDigest(cache.Semantic) {
		return toolchainCache{}, fmt.Errorf("invalid toolchain cache")
	}
	return cache, nil
}

func semanticToolchainDigest(semantic toolchainSemantic) string {
	payload, _ := json.Marshal(semantic)
	domainSeparated := append([]byte("toolchainDigest.v2\x00"), payload...)
	return digestBytes(domainSeparated)
}

func toolchainIntegrity(cache toolchainCache) string {
	payload, _ := json.Marshal(struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Go            executableFingerprint `json:"go"`
		Git           executableFingerprint `json:"git"`
		Semantic      toolchainSemantic     `json:"semantic"`
		Digest        string                `json:"digest"`
	}{cache.SchemaVersion, cache.Go, cache.Git, cache.Semantic, cache.Digest})
	return digestBytes(payload)
}

func validExecutableFingerprint(fingerprint executableFingerprint) bool {
	return filepath.IsAbs(fingerprint.Path) && filepath.Clean(fingerprint.Path) == fingerprint.Path && fingerprint.Size >= 0
}

func validToolchainSemantic(semantic toolchainSemantic) bool {
	if semantic.Domain != toolchainDigestDomain || semantic.Version != toolchainDigestVersion {
		return false
	}
	platform, architecture, err := normalizeToolchainPlatform(semantic.Platform, semantic.Architecture)
	if err != nil || platform != semantic.Platform || architecture != semantic.Architecture {
		return false
	}
	goVersion, goErr := normalizeToolVersion("go", semantic.GoVersion)
	gitVersion, gitErr := normalizeToolVersion("git", semantic.GitVersion)
	return goErr == nil && gitErr == nil && goVersion == semantic.GoVersion && gitVersion == semantic.GitVersion
}

func normalizeToolchainPlatform(platform, architecture string) (string, string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	architecture = strings.ToLower(strings.TrimSpace(architecture))
	valid := func(value string) bool {
		if value == "" {
			return false
		}
		for _, char := range value {
			if char < 'a' || char > 'z' {
				if char < '0' || char > '9' {
					return false
				}
			}
		}
		return true
	}
	if !valid(platform) || !valid(architecture) {
		return "", "", fmt.Errorf("toolchain platform/architecture is invalid")
	}
	return platform, architecture, nil
}

func normalizeToolVersion(name, output string) (string, error) {
	if !utf8.ValidString(output) || strings.ContainsRune(output, utf8.RuneError) || strings.IndexFunc(output, func(char rune) bool {
		return unicode.IsControl(char) && !unicode.IsSpace(char)
	}) >= 0 {
		return "", fmt.Errorf("%s version output is not valid text", name)
	}
	fields := strings.Fields(output)
	normalized := strings.Join(fields, " ")
	switch name {
	case "go":
		if len(fields) < 4 || fields[0] != "go" || fields[1] != "version" || !strings.Contains(fields[len(fields)-1], "/") {
			return "", fmt.Errorf("go version output is not recognized: %q", normalized)
		}
		if fields[2] != "devel" && !strings.HasPrefix(fields[2], "go") {
			return "", fmt.Errorf("go version output is not recognized: %q", normalized)
		}
	case "git":
		if len(fields) < 3 || fields[0] != "git" || fields[1] != "version" || fields[2] == "" || fields[2][0] < '0' || fields[2][0] > '9' {
			return "", fmt.Errorf("git version output is not recognized: %q", normalized)
		}
	default:
		return "", fmt.Errorf("unsupported toolchain version output %q", name)
	}
	return normalized, nil
}

func fingerprintError(message string) *Error {
	return &Error{Code: CodeFingerprintInvalid, Message: message, RequiredAction: "recompute validation evidence from current repository inputs"}
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", sum)
}
