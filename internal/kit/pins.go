package kit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

const (
	pinMetadataName = "metadata.json"
	pinObjectDir    = "object.git"
	pinContentDir   = "content"
)

type PinStatus struct {
	SchemaVersion  int           `json:"schemaVersion"`
	KitID          string        `json:"kitId"`
	Source         *PinnedSource `json:"source"`
	Identity       string        `json:"identity"`
	CachePath      string        `json:"cachePath"`
	Resolved       bool          `json:"resolved"`
	ResolvedCommit string        `json:"resolvedCommit,omitempty"`
	ContentDigest  string        `json:"contentDigest,omitempty"`
	TreeOID        string        `json:"treeOID,omitempty"`
	NetworkCalls   int           `json:"networkCalls"`
	RequiredAction string        `json:"requiredAction,omitempty"`
}

type PinMaterialization struct {
	SchemaVersion    int    `json:"schemaVersion"`
	KitID            string `json:"kitId"`
	SourceIdentity   string `json:"sourceIdentity"`
	ContentIdentity  string `json:"contentIdentity"`
	CachePath        string `json:"cachePath"`
	MaterializedPath string `json:"materializedPath"`
	NetworkCalls     int    `json:"networkCalls"`
}

type pinMetadata struct {
	SchemaVersion  int           `json:"schemaVersion"`
	Source         *PinnedSource `json:"source"`
	Identity       string        `json:"identity"`
	ResolvedCommit string        `json:"resolvedCommit,omitempty"`
	ContentDigest  string        `json:"contentDigest,omitempty"`
	TreeOID        string        `json:"treeOID,omitempty"`
}

type PinCacheMissError struct {
	Identity       string
	RequiredAction string
}

func (err *PinCacheMissError) Error() string {
	return "pinned source is not prefetched: " + err.Identity
}

var runPinFetch = func(objectRepository, locator, commit string) error {
	_, err := gitx.Run(objectRepository, "fetch", "--no-tags", "--depth=1", locator, commit)
	return err
}

func PinCacheRoot(repo string) (string, error) {
	commonDir, err := gitx.CommonDir(repo)
	if err != nil {
		return "", fmt.Errorf("resolve Git common-dir for pin cache: %w", err)
	}
	return filepath.Join(commonDir, "aicoding", "pins"), nil
}

func InspectPin(repo, kitID string, source *PinnedSource) (PinStatus, error) {
	identity, err := PinnedSourceIdentity(source)
	if err != nil {
		return PinStatus{}, err
	}
	root, err := PinCacheRoot(repo)
	if err != nil {
		return PinStatus{}, err
	}
	cachePath := filepath.Join(root, strings.TrimPrefix(identity, "sha256:"))
	status := PinStatus{
		SchemaVersion: 1, KitID: kitID, Source: clonePinnedSource(source), Identity: identity,
		CachePath: cachePath, RequiredAction: prefetchRequiredAction(kitID),
	}
	info, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return status, fmt.Errorf("inspect pin cache %s: %w", identity, err)
	}
	if !info.IsDir() {
		return status, fmt.Errorf("pin cache entry is not a directory: %s", identity)
	}
	metadataContent, err := os.ReadFile(filepath.Join(cachePath, pinMetadataName))
	if err != nil {
		return status, fmt.Errorf("pin cache metadata is missing for %s: %w", identity, err)
	}
	var metadata pinMetadata
	if err := decodeStrictJSON(metadataContent, &metadata); err != nil {
		return status, fmt.Errorf("pin cache metadata is invalid for %s: %w", identity, err)
	}
	if metadata.SchemaVersion != 1 || metadata.Identity != identity || !samePinnedSource(metadata.Source, source) {
		return status, fmt.Errorf("pin cache metadata does not match source identity %s", identity)
	}
	status.ResolvedCommit = metadata.ResolvedCommit
	status.ContentDigest = metadata.ContentDigest
	status.TreeOID = metadata.TreeOID
	switch source.Kind {
	case "git":
		if metadata.ResolvedCommit != source.Commit || !pinnedCommitPattern.MatchString(metadata.ResolvedCommit) || metadata.TreeOID == "" {
			return status, fmt.Errorf("git pin cache metadata is incomplete for %s", identity)
		}
		objectRepository := filepath.Join(cachePath, pinObjectDir)
		resolved, err := gitx.Run(objectRepository, "rev-parse", "refs/aicoding/pin^{commit}")
		if err != nil {
			return status, fmt.Errorf("git pin cache commit verification failed for %s: %w", identity, err)
		}
		if got := strings.ToLower(strings.TrimSpace(resolved)); got != source.Commit {
			return status, fmt.Errorf("git pin cache commit verification failed for %s: got %s want %s", identity, got, source.Commit)
		}
		treeOID, err := gitx.TreeOID(objectRepository, source.Commit)
		if err != nil {
			return status, fmt.Errorf("git pin cache tree verification failed for %s: %w", identity, err)
		}
		if treeOID != metadata.TreeOID {
			return status, fmt.Errorf("git pin cache tree verification failed for %s: got %s want %s", identity, treeOID, metadata.TreeOID)
		}
	case "content":
		contentDigest, err := digestDirectory(filepath.Join(cachePath, pinContentDir))
		if err != nil {
			return status, fmt.Errorf("content pin cache verification failed for %s: %w", identity, err)
		}
		if contentDigest != source.Digest || metadata.ContentDigest != source.Digest {
			return status, fmt.Errorf("content pin cache digest mismatch for %s", identity)
		}
	default:
		return status, fmt.Errorf("unsupported pinned source kind %q", source.Kind)
	}
	status.Resolved = true
	status.RequiredAction = ""
	return status, nil
}

func PrefetchPin(ctx context.Context, repo, kitID string, source *PinnedSource) (PinStatus, error) {
	if source != nil && source.Kind == "content" {
		if err := prepareContentPinMetadata(repo, source); err != nil {
			return PinStatus{}, err
		}
	}
	status, err := InspectPin(repo, kitID, source)
	if err == nil && status.Resolved {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	if source.Kind == "content" {
		return status, &PinCacheMissError{Identity: status.Identity, RequiredAction: status.RequiredAction}
	}
	root := filepath.Dir(status.CachePath)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return status, fmt.Errorf("create pin cache root: %w", err)
	}
	// Keep the staging leaf short: Git's bare-repository templates add nested
	// hook filenames and Windows still encounters legacy MAX_PATH consumers.
	temporary, err := os.MkdirTemp(root, ".pin-")
	if err != nil {
		return status, fmt.Errorf("create pin cache staging directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	objectRepository := filepath.Join(temporary, pinObjectDir)
	if _, err := gitx.Run("", "init", "--bare", objectRepository); err != nil {
		return status, fmt.Errorf("initialize pin object cache: %w", err)
	}
	// Fetch happens under the short staging leaf, but the published cache path
	// adds a 64-hex identity. Keep loose-object access valid after that rename
	// on Windows, where the final path can otherwise cross MAX_PATH.
	if _, err := gitx.Run(objectRepository, "config", "core.longpaths", "true"); err != nil {
		return status, fmt.Errorf("enable long paths for pin object cache: %w", err)
	}
	status.NetworkCalls = 1
	if err := runPinFetch(objectRepository, source.URL, source.Commit); err != nil {
		return status, fmt.Errorf("prefetch pinned commit %s: %w", source.Commit, err)
	}
	resolved, err := gitx.Run(objectRepository, "rev-parse", "FETCH_HEAD^{commit}")
	if err != nil {
		return status, fmt.Errorf("resolve fetched pin: %w", err)
	}
	resolved = strings.ToLower(strings.TrimSpace(resolved))
	if resolved != source.Commit {
		return status, fmt.Errorf("fetched commit mismatch: got %s want %s", resolved, source.Commit)
	}
	if _, err := gitx.Run(objectRepository, "update-ref", "refs/aicoding/pin", resolved); err != nil {
		return status, fmt.Errorf("publish fetched pin ref: %w", err)
	}
	treeOID, err := gitx.TreeOID(objectRepository, resolved)
	if err != nil {
		return status, fmt.Errorf("resolve fetched pin tree: %w", err)
	}
	metadata := pinMetadata{
		SchemaVersion: 1, Source: clonePinnedSource(source), Identity: status.Identity,
		ResolvedCommit: resolved, TreeOID: treeOID,
	}
	if err := writePinMetadata(filepath.Join(temporary, pinMetadataName), metadata); err != nil {
		return status, err
	}
	if err := os.Rename(temporary, status.CachePath); err != nil {
		if concurrent, inspectErr := InspectPin(repo, kitID, source); inspectErr == nil && concurrent.Resolved {
			concurrent.NetworkCalls = status.NetworkCalls
			return concurrent, nil
		}
		return status, fmt.Errorf("publish pin cache entry: %w", err)
	}
	resolvedStatus, err := InspectPin(repo, kitID, source)
	resolvedStatus.NetworkCalls = status.NetworkCalls
	return resolvedStatus, err
}

func prepareContentPinMetadata(repo string, source *PinnedSource) error {
	identity, err := PinnedSourceIdentity(source)
	if err != nil {
		return err
	}
	root, err := PinCacheRoot(repo)
	if err != nil {
		return err
	}
	cachePath := filepath.Join(root, strings.TrimPrefix(identity, "sha256:"))
	metadataPath := filepath.Join(cachePath, pinMetadataName)
	if _, err := os.Stat(metadataPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect content pin metadata: %w", err)
	}
	contentRoot := filepath.Join(cachePath, pinContentDir)
	if info, err := os.Stat(contentRoot); err != nil || !info.IsDir() {
		return nil
	}
	digest, err := digestDirectory(contentRoot)
	if err != nil {
		return err
	}
	if digest != source.Digest {
		return fmt.Errorf("content pin cache digest mismatch: got %s want %s", digest, source.Digest)
	}
	return writePinMetadata(metadataPath, pinMetadata{
		SchemaVersion: 1, Source: clonePinnedSource(source), Identity: identity, ContentDigest: digest,
	})
}

func PinnedPathExists(repo string, source *PinnedSource, relative string) (bool, error) {
	status, err := InspectPin(repo, "", source)
	if err != nil {
		return false, err
	}
	if !status.Resolved {
		return false, &PinCacheMissError{Identity: status.Identity, RequiredAction: status.RequiredAction}
	}
	relative, err = cleanPinnedPath(relative)
	if err != nil {
		return false, err
	}
	if relative == "." {
		return true, nil
	}
	if source.Kind == "content" {
		_, err := os.Stat(filepath.Join(status.CachePath, pinContentDir, filepath.FromSlash(relative)))
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	}
	_, err = gitx.Run(filepath.Join(status.CachePath, pinObjectDir), "cat-file", "-e", source.Commit+":"+relative)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func ReadPinnedFile(repo string, source *PinnedSource, relative string) ([]byte, error) {
	status, err := InspectPin(repo, "", source)
	if err != nil {
		return nil, err
	}
	if !status.Resolved {
		return nil, &PinCacheMissError{Identity: status.Identity, RequiredAction: status.RequiredAction}
	}
	relative, err = cleanPinnedPath(relative)
	if err != nil || relative == "." {
		return nil, fmt.Errorf("invalid pinned file path %q", relative)
	}
	if source.Kind == "content" {
		return os.ReadFile(filepath.Join(status.CachePath, pinContentDir, filepath.FromSlash(relative)))
	}
	content, err := gitx.Run(filepath.Join(status.CachePath, pinObjectDir), "show", source.Commit+":"+relative)
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

func MaterializePinnedSource(ctx context.Context, repo, kitID string, source *PinnedSource) (PinMaterialization, error) {
	status, err := InspectPin(repo, kitID, source)
	result := PinMaterialization{
		SchemaVersion: 1, KitID: kitID, SourceIdentity: status.Identity,
		CachePath: status.CachePath, NetworkCalls: 0,
	}
	if err != nil {
		return result, err
	}
	if !status.Resolved {
		return result, &PinCacheMissError{Identity: status.Identity, RequiredAction: prefetchRequiredAction(kitID)}
	}
	if !kitInitIDPattern.MatchString(kitID) {
		return result, fmt.Errorf("invalid kit id for pinned materialization: %s", kitID)
	}
	stateRoot := platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "state", "kits", kitID)))
	if err := os.MkdirAll(stateRoot, 0o755); err != nil {
		return result, fmt.Errorf("create Kit state root: %w", err)
	}
	temporary, err := os.MkdirTemp(stateRoot, ".pin-source-")
	if err != nil {
		return result, fmt.Errorf("create materialization staging directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	switch source.Kind {
	case "git":
		if err := extractGitArchive(ctx, filepath.Join(status.CachePath, pinObjectDir), source.Commit, temporary); err != nil {
			return result, fmt.Errorf("materialize pinned Git source: %w", err)
		}
		result.ContentIdentity = "git-tree:" + status.TreeOID
	case "content":
		if err := copyPinnedTree(filepath.Join(status.CachePath, pinContentDir), temporary); err != nil {
			return result, fmt.Errorf("materialize pinned content source: %w", err)
		}
		result.ContentIdentity = status.ContentDigest
	default:
		return result, fmt.Errorf("unsupported pinned source kind %q", source.Kind)
	}
	destination := filepath.Join(stateRoot, "source")
	if err := replacePinnedMaterialization(temporary, destination); err != nil {
		return result, err
	}
	result.MaterializedPath = destination
	return result, nil
}

func RemovePinnedMaterialization(repo, kitID string) error {
	if !kitInitIDPattern.MatchString(kitID) {
		return fmt.Errorf("invalid kit id for pinned materialization: %s", kitID)
	}
	return os.RemoveAll(platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "state", "kits", kitID, "source"))))
}

func ReferencedPinIdentities(repo string) (map[string]bool, error) {
	entries, err := LoadRegistry(repo)
	if err != nil {
		return nil, err
	}
	referenced := map[string]bool{}
	for _, entry := range entries {
		manifest, err := LoadManifest(repo, entry.Manifest)
		if err != nil {
			return nil, fmt.Errorf("load registered Kit %s: %w", entry.ID, err)
		}
		if manifest.Source == nil {
			continue
		}
		identity, err := PinnedSourceIdentity(manifest.Source)
		if err != nil {
			return nil, fmt.Errorf("resolve registered Kit %s source: %w", entry.ID, err)
		}
		referenced[identity] = true
	}
	return referenced, nil
}

func prefetchRequiredAction(kitID string) string {
	if strings.TrimSpace(kitID) == "" {
		return "aicoding kit prefetch --id <kit-id> --json"
	}
	return "aicoding kit prefetch --id " + kitID + " --json"
}

func samePinnedSource(left, right *PinnedSource) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftCopy, rightCopy := *left, *right
	if leftCopy.normalizeAndValidate() != nil || rightCopy.normalizeAndValidate() != nil {
		return false
	}
	return leftCopy == rightCopy
}

func writePinMetadata(target string, metadata pinMetadata) error {
	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode pin cache metadata: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(target, content, 0o600); err != nil {
		return fmt.Errorf("write pin cache metadata: %w", err)
	}
	return nil
}

func cleanPinnedPath(relative string) (string, error) {
	relative = filepath.ToSlash(strings.TrimSpace(relative))
	if relative == "" || strings.Contains(relative, ":") || strings.ContainsAny(relative, "\r\n\x00") {
		return "", fmt.Errorf("pinned source path is empty or unsafe")
	}
	clean := path.Clean(relative)
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("pinned source path escapes source root: %s", relative)
	}
	return clean, nil
}

func replacePinnedMaterialization(staged, destination string) error {
	backup := destination + ".previous"
	_ = os.RemoveAll(backup)
	hadDestination := false
	if _, err := os.Stat(destination); err == nil {
		if err := os.Rename(destination, backup); err != nil {
			return fmt.Errorf("stage previous pinned materialization: %w", err)
		}
		hadDestination = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect previous pinned materialization: %w", err)
	}
	if err := os.Rename(staged, destination); err != nil {
		if hadDestination {
			_ = os.Rename(backup, destination)
		}
		return fmt.Errorf("publish pinned materialization: %w", err)
	}
	if hadDestination {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove previous pinned materialization: %w", err)
		}
	}
	return nil
}

func copyPinnedTree(source, destination string) error {
	return filepath.Walk(source, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, current)
		if err != nil || relative == "." {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("content pin contains unsupported symlink: %s", filepath.ToSlash(relative))
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		input, err := os.Open(current)
		if err != nil {
			return err
		}
		defer input.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func digestDirectory(root string) (string, error) {
	files := []string{}
	err := filepath.Walk(root, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("content pin contains unsupported symlink")
		}
		if !info.IsDir() {
			relative, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	hash := sha256.New()
	for _, relative := range files {
		_, _ = io.WriteString(hash, relative)
		_, _ = hash.Write([]byte{0})
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
