package gitx

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Status is one parsed git status snapshot. A snapshot is produced by one
// porcelain-v2 invocation and includes submodule worktree dirtiness.
type Status struct {
	TrackedModified bool
	Staged          bool
	Untracked       bool
	SubmoduleDirty  bool
	Unmerged        bool
	Paths           []string
	StagedPaths     []string
}

// PushUpdate is one record from Git's pre-push stdin protocol.
type PushUpdate struct {
	LocalRef  string `json:"localRef"`
	LocalOID  string `json:"localOID"`
	RemoteRef string `json:"remoteRef"`
	RemoteOID string `json:"remoteOID"`
}

// TreeEntry is one tracked object from a recursive Git tree listing.
type TreeEntry struct {
	Mode string `json:"mode"`
	Type string `json:"type"`
	OID  string `json:"oid"`
	Path string `json:"path"`
}

func Run(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if repo != "" {
		cmd.Dir = repo
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// Archive streams one Git tree or commit as a tar archive without reading
// worktree files. The caller owns extraction and destination lifecycle.
func Archive(ctx context.Context, repo, rev string, destination io.Writer) error {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		return fmt.Errorf("git archive revision is empty")
	}
	if destination == nil {
		return fmt.Errorf("git archive destination is nil")
	}
	cmd := exec.CommandContext(ctx, "git", "archive", "--format=tar", rev)
	if repo != "" {
		cmd.Dir = repo
	}
	var stderr bytes.Buffer
	cmd.Stdout = destination
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git archive %s: %w: %s", rev, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// ParsePushUpdates parses Git's four-field pre-push stdin protocol without
// assigning repository policy to the refs.
func ParsePushUpdates(reader io.Reader) ([]PushUpdate, error) {
	if reader == nil {
		return nil, fmt.Errorf("pre-push input is required")
	}
	updates := make([]PushUpdate, 0)
	scanner := bufio.NewScanner(reader)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 4 {
			return nil, fmt.Errorf("parse pre-push line %d: expected 4 fields", lineNumber)
		}
		if !validObjectID(fields[1]) || !validObjectID(fields[3]) {
			return nil, fmt.Errorf("parse pre-push line %d: invalid object id", lineNumber)
		}
		updates = append(updates, PushUpdate{
			LocalRef: fields[0], LocalOID: strings.ToLower(fields[1]), RemoteRef: fields[2], RemoteOID: strings.ToLower(fields[3]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read pre-push input: %w", err)
	}
	return updates, nil
}

// IsAncestor reports whether ancestor is reachable from descendant. Exit code
// 1 is a normal negative answer; other Git failures remain errors.
func IsAncestor(repo, ancestor, descendant string) (bool, error) {
	ancestor = strings.TrimSpace(ancestor)
	descendant = strings.TrimSpace(descendant)
	if ancestor == "" || descendant == "" {
		return false, fmt.Errorf("git ancestry requires two revisions")
	}
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	if repo != "" {
		cmd.Dir = repo
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git merge-base --is-ancestor: %w: %s", err, strings.TrimSpace(stderr.String()))
}

func StagedFiles(repo string) ([]string, error) {
	out, err := Run(repo, "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	if err != nil {
		return nil, err
	}
	return splitFileList(out), nil
}

// DiffTreeFiles returns the repository-relative paths changed between two Git
// trees. The result includes additions, modifications, renames, and deletions
// in deterministic order.
func DiffTreeFiles(repo, fromTree, toTree string) ([]string, error) {
	fromTree = strings.TrimSpace(fromTree)
	toTree = strings.TrimSpace(toTree)
	if fromTree == "" || toTree == "" {
		return nil, fmt.Errorf("git tree diff requires two tree object ids")
	}
	out, err := Run(repo, "diff", "--name-only", fromTree, toTree, "--")
	if err != nil {
		return nil, err
	}
	files := splitFileList(out)
	sort.Strings(files)
	return files, nil
}

// HeadCommit returns the commit object currently named by HEAD.
func HeadCommit(repo string) (string, error) {
	return runOID(repo, "rev-parse", "HEAD")
}

// TreeOID returns the tree object for rev.
func TreeOID(repo, rev string) (string, error) {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		return "", fmt.Errorf("git tree revision is empty")
	}
	return runOID(repo, "rev-parse", rev+"^{tree}")
}

// TreeEntries returns every tracked blob and gitlink in one deterministic
// ls-tree invocation. File content is never read from the worktree.
func TreeEntries(repo, tree string) ([]TreeEntry, error) {
	tree = strings.TrimSpace(tree)
	if tree == "" {
		return nil, fmt.Errorf("git tree object id is empty")
	}
	out, err := Run(repo, "ls-tree", "-r", "-z", "--full-tree", tree)
	if err != nil {
		return nil, err
	}
	entries := make([]TreeEntry, 0)
	for _, record := range strings.Split(out, "\x00") {
		if record == "" {
			continue
		}
		tab := strings.IndexByte(record, '\t')
		if tab < 1 || tab == len(record)-1 {
			return nil, fmt.Errorf("parse git ls-tree record %q", record)
		}
		metadata := strings.Fields(record[:tab])
		if len(metadata) != 3 || !validObjectID(metadata[2]) {
			return nil, fmt.Errorf("parse git ls-tree metadata %q", record[:tab])
		}
		entries = append(entries, TreeEntry{
			Mode: metadata[0], Type: metadata[1], OID: strings.ToLower(metadata[2]), Path: record[tab+1:],
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// WriteTree writes the current index as a Git tree object and returns its OID.
// It does not modify the worktree, index, or HEAD.
func WriteTree(repo string) (string, error) {
	return runOID(repo, "write-tree")
}

// CommonDir returns the absolute Git common directory shared by linked
// worktrees. Conventional worktree metadata is resolved without spawning Git;
// bare repositories, subdirectories and unusual layouts fall back to Git.
func CommonDir(repo string) (string, error) {
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("resolve repository: %w", err)
	}
	if dir, fastErr := commonDirFromDotGit(absRepo); fastErr == nil {
		if canonical, canonicalErr := canonicalExistingDir(dir); canonicalErr == nil {
			return canonical, nil
		}
	}
	out, err := Run(absRepo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(out)
	if dir == "" {
		return "", fmt.Errorf("git common directory is empty")
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(absRepo, dir)
	}
	return canonicalExistingDir(dir)
}

func canonicalExistingDir(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve git common directory: %w", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("canonicalize git common directory: %w", err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("invalid Git common directory")
	}
	return filepath.Clean(dir), nil
}

func commonDirFromDotGit(repo string) (string, error) {
	dotGit := filepath.Join(repo, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Abs(dotGit)
	}
	raw, err := os.ReadFile(dotGit)
	if err != nil {
		return "", err
	}
	const prefix = "gitdir:"
	line := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(strings.ToLower(line), prefix) {
		return "", fmt.Errorf("invalid .git file")
	}
	gitDir := strings.TrimSpace(line[len(prefix):])
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repo, gitDir)
	}
	gitDir, err = filepath.Abs(gitDir)
	if err != nil {
		return "", err
	}
	commonDir := gitDir
	if rawCommonDir, readErr := os.ReadFile(filepath.Join(gitDir, "commondir")); readErr == nil {
		commonDir = strings.TrimSpace(string(rawCommonDir))
		if !filepath.IsAbs(commonDir) {
			commonDir = filepath.Join(gitDir, commonDir)
		}
	} else if !os.IsNotExist(readErr) {
		return "", readErr
	}
	commonDir, err = filepath.Abs(commonDir)
	if err != nil {
		return "", err
	}
	if info, err = os.Stat(commonDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("invalid Git common directory")
	}
	return filepath.Clean(commonDir), nil
}

// StatusSnapshot parses tracked, staged, untracked, unmerged, and submodule
// dirtiness from one Git status process.
func StatusSnapshot(repo string) (Status, error) {
	out, err := Run(repo, "status", "--porcelain=v2", "-z", "--untracked-files=normal", "--ignore-submodules=none")
	if err != nil {
		return Status{}, err
	}
	return parseStatusSnapshot(out)
}

func parseStatusSnapshot(out string) (Status, error) {
	var status Status
	paths := map[string]struct{}{}
	stagedPaths := map[string]struct{}{}
	records := strings.Split(out, "\x00")
	for index := 0; index < len(records); index++ {
		record := records[index]
		if record == "" || strings.HasPrefix(record, "#") {
			continue
		}
		switch record[0] {
		case '?':
			if len(record) < 3 || record[1] != ' ' {
				return Status{}, fmt.Errorf("parse git status untracked record %q", record)
			}
			status.Untracked = true
			addStatusPath(paths, record[2:])
		case 'u':
			fields := strings.SplitN(record, " ", 11)
			if len(fields) != 11 {
				return Status{}, fmt.Errorf("parse git status unmerged record %q", record)
			}
			status.Unmerged = true
			status.Staged = true
			status.TrackedModified = true
			addStatusPath(paths, fields[10])
			addStatusPath(stagedPaths, fields[10])
		case '1':
			fields := strings.SplitN(record, " ", 9)
			if len(fields) != 9 {
				return Status{}, fmt.Errorf("parse git status ordinary record %q", record)
			}
			xy, sub := fields[1], fields[2]
			if len(xy) != 2 {
				return Status{}, fmt.Errorf("parse git status XY %q", xy)
			}
			status.Staged = status.Staged || xy[0] != '.'
			status.TrackedModified = status.TrackedModified || xy[1] != '.'
			if len(sub) == 4 && sub[0] == 'S' && sub[1:] != "..." {
				status.SubmoduleDirty = true
			}
			addStatusPath(paths, fields[8])
			if xy[0] != '.' {
				addStatusPath(stagedPaths, fields[8])
			}
		case '2':
			fields := strings.SplitN(record, " ", 10)
			if len(fields) != 10 || index+1 >= len(records) || records[index+1] == "" {
				return Status{}, fmt.Errorf("parse git status rename record %q", record)
			}
			xy, sub := fields[1], fields[2]
			if len(xy) != 2 {
				return Status{}, fmt.Errorf("parse git status XY %q", xy)
			}
			status.Staged = status.Staged || xy[0] != '.'
			status.TrackedModified = status.TrackedModified || xy[1] != '.'
			if len(sub) == 4 && sub[0] == 'S' && sub[1:] != "..." {
				status.SubmoduleDirty = true
			}
			addStatusPath(paths, fields[9])
			addStatusPath(paths, records[index+1])
			if xy[0] != '.' {
				addStatusPath(stagedPaths, fields[9])
				addStatusPath(stagedPaths, records[index+1])
			}
			index++
		default:
			return Status{}, fmt.Errorf("unsupported git status porcelain record %q", record)
		}
	}
	status.Paths = statusPathList(paths)
	status.StagedPaths = statusPathList(stagedPaths)
	return status, nil
}

func addStatusPath(paths map[string]struct{}, value string) {
	value = filepath.ToSlash(value)
	if value != "" {
		paths[value] = struct{}{}
	}
}

func statusPathList(paths map[string]struct{}) []string {
	if len(paths) == 0 {
		return nil
	}
	values := make([]string, 0, len(paths))
	for value := range paths {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

// CommitFiles returns the paths changed by a single commit reference (default
// HEAD when ref is empty). Paths use forward slashes.
func CommitFiles(repo, ref string) ([]string, error) {
	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}
	out, err := Run(repo, "diff-tree", "--no-commit-id", "--name-only", "-r", ref)
	if err != nil {
		return nil, err
	}
	return splitFileList(out), nil
}

func splitFileList(out string) []string {
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\\", "/"))
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func runOID(repo string, args ...string) (string, error) {
	out, err := Run(repo, args...)
	if err != nil {
		return "", err
	}
	oid := strings.TrimSpace(out)
	if oid == "" {
		return "", fmt.Errorf("git %s returned an empty object id", strings.Join(args, " "))
	}
	return oid, nil
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
