package validationevidence

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

// Open binds evidence operations to the Git common directory for repo.
func Open(repo string) (Repository, error) {
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return Repository{}, fmt.Errorf("resolve repository: %w", err)
	}
	commonDir, err := commonDirFromDotGit(absRepo)
	if err != nil {
		commonDir, err = gitx.CommonDir(absRepo)
	}
	if err != nil {
		return Repository{}, &Error{Code: CodeTargetNotFound, Message: err.Error(), RequiredAction: "run inside a Git repository"}
	}
	normalized := filepath.ToSlash(filepath.Clean(commonDir))
	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}
	sum := sha256.Sum256([]byte(normalized))
	return Repository{
		repo:         filepath.Clean(absRepo),
		commonDir:    commonDir,
		root:         filepath.Join(commonDir, "aicoding", "validation"),
		repositoryID: fmt.Sprintf("sha256:%x", sum),
	}, nil
}

// commonDirFromDotGit avoids a third Git process on the validation hot path.
// Git remains the fallback for bare repositories, subdirectories and unusual
// repository layouts that do not expose a conventional worktree .git entry.
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

// Capture reads status exactly once and resolves the requested Git tree.
func (r Repository) Capture(target Target) (Subject, error) {
	status, err := gitx.StatusSnapshot(r.repo)
	if err != nil {
		return Subject{}, &Error{Code: CodeTargetNotFound, Message: err.Error(), RequiredAction: "verify the repository and Git index"}
	}
	subject := Subject{Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
	switch target {
	case TargetAuto:
		switch {
		case !dirtyForHead(status):
			subject.Mode = SubjectHead
			subject.TreeOID, err = gitx.TreeOID(r.repo, "HEAD")
		case status.Staged && !dirtyForIndex(status):
			subject.Mode = SubjectIndex
			subject.TreeOID, err = gitx.WriteTree(r.repo)
		default:
			subject.Mode = SubjectDirty
			subject.Reusable = false
			subject.ReusableReason = statusReason(status, true)
			if status.Staged && !status.Unmerged {
				subject.TreeOID, err = gitx.WriteTree(r.repo)
			} else {
				subject.TreeOID, err = gitx.TreeOID(r.repo, "HEAD")
			}
		}
	case TargetHead:
		subject.Mode = SubjectHead
		subject.TreeOID, err = gitx.TreeOID(r.repo, "HEAD")
		if dirtyForHead(status) {
			subject.Mode = SubjectDirty
			subject.Reusable = false
			subject.ReusableReason = statusReason(status, false)
		}
	case TargetIndex:
		if status.Unmerged {
			return Subject{}, &Error{Code: CodeTargetNotFound, Message: "the Git index contains unmerged entries", RequiredAction: "resolve conflicts and stage the intended content"}
		}
		subject.Mode = SubjectIndex
		subject.TreeOID, err = gitx.WriteTree(r.repo)
		if dirtyForIndex(status) {
			subject.Mode = SubjectDirty
			subject.Reusable = false
			subject.ReusableReason = statusReason(status, true)
		}
	default:
		return Subject{}, &Error{Code: CodeTargetNotFound, Message: fmt.Sprintf("unsupported validation target %q", target), RequiredAction: "use HEAD or INDEX"}
	}
	if err != nil {
		return Subject{}, &Error{Code: CodeTargetNotFound, Message: err.Error(), RequiredAction: "verify that the requested Git tree exists"}
	}
	return subject, nil
}

func dirtyForHead(status gitx.Status) bool {
	return status.TrackedModified || status.Staged || status.Untracked || status.SubmoduleDirty || status.Unmerged
}

func dirtyForIndex(status gitx.Status) bool {
	return status.TrackedModified || status.Untracked || status.SubmoduleDirty || status.Unmerged
}

func statusReason(status gitx.Status, stagedAllowed bool) string {
	reasons := make([]string, 0, 5)
	if status.Unmerged {
		reasons = append(reasons, "unmerged index entries")
	}
	if status.TrackedModified {
		reasons = append(reasons, "tracked worktree changes")
	}
	if status.Staged && !stagedAllowed {
		reasons = append(reasons, "staged changes outside HEAD")
	}
	if status.Untracked {
		reasons = append(reasons, "untracked non-ignored files")
	}
	if status.SubmoduleDirty {
		reasons = append(reasons, "dirty submodule worktree")
	}
	return strings.Join(reasons, "; ")
}
