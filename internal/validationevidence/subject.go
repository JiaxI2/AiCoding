package validationevidence

import (
	"crypto/sha256"
	"fmt"
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
	commonDir, err := gitx.CommonDir(absRepo)
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

// Capture reads status exactly once and resolves the requested Git tree.
func (r Repository) Capture(target Target) (Subject, error) {
	status, err := gitx.StatusSnapshot(r.repo)
	if err != nil {
		return Subject{}, &Error{Code: CodeTargetNotFound, Message: err.Error(), RequiredAction: "verify the repository and Git index"}
	}
	subject := Subject{Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
	switch target {
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
