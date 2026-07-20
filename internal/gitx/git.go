package gitx

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
}

// PushUpdate is one record from Git's pre-push stdin protocol.
type PushUpdate struct {
	LocalRef  string `json:"localRef"`
	LocalOID  string `json:"localOID"`
	RemoteRef string `json:"remoteRef"`
	RemoteOID string `json:"remoteOID"`
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
	out, err := Run(repo, "status", "--porcelain=v2", "--untracked-files=normal", "--ignore-submodules=none")
	if err != nil {
		return Status{}, err
	}
	var status Status
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line[0] {
		case '?':
			status.Untracked = true
		case 'u':
			status.Unmerged = true
			status.Staged = true
			status.TrackedModified = true
		case '1', '2':
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return Status{}, fmt.Errorf("parse git status porcelain line %q", line)
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
		default:
			return Status{}, fmt.Errorf("unsupported git status porcelain line %q", line)
		}
	}
	return status, nil
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
