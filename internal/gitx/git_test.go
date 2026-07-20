package gitx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentIdentityPrimitives(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "tracked.txt", "one\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-m", "initial")

	head, err := HeadCommit(repo)
	if err != nil {
		t.Fatal(err)
	}
	headTree, err := TreeOID(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	indexTree, err := WriteTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(head) != 40 || headTree == "" || headTree != indexTree {
		t.Fatalf("unexpected identity: head=%q headTree=%q indexTree=%q", head, headTree, indexTree)
	}
	commonDir, err := CommonDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(commonDir) || filepath.Clean(commonDir) != filepath.Join(repo, ".git") {
		t.Fatalf("CommonDir() = %q", commonDir)
	}
	status, err := StatusSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if status != (Status{}) {
		t.Fatalf("clean status = %#v", status)
	}
}

func TestStatusSnapshotClassifiesChanges(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "tracked.txt", "one\n")
	writeGitFile(t, repo, "staged.txt", "one\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-m", "initial")

	writeGitFile(t, repo, "tracked.txt", "two\n")
	writeGitFile(t, repo, "staged.txt", "two\n")
	mustGit(t, repo, "add", "staged.txt")
	writeGitFile(t, repo, "untracked.txt", "new\n")

	status, err := StatusSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !status.TrackedModified || !status.Staged || !status.Untracked || status.SubmoduleDirty || status.Unmerged {
		t.Fatalf("status = %#v", status)
	}
}

func TestStatusSnapshotDetectsDirtySubmodule(t *testing.T) {
	child := newGitRepo(t)
	writeGitFile(t, child, "child.txt", "one\n")
	mustGit(t, child, "add", "child.txt")
	mustGit(t, child, "commit", "-m", "child")

	parent := newGitRepo(t)
	mustGit(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "deps/child")
	mustGit(t, parent, "commit", "-am", "add child")
	writeGitFile(t, filepath.Join(parent, "deps", "child"), "child.txt", "dirty\n")

	status, err := StatusSnapshot(parent)
	if err != nil {
		t.Fatal(err)
	}
	if !status.TrackedModified || !status.SubmoduleDirty {
		t.Fatalf("dirty submodule status = %#v", status)
	}
}

func TestTreeOIDRejectsEmptyRevision(t *testing.T) {
	if _, err := TreeOID(t.TempDir(), "  "); err == nil {
		t.Fatal("TreeOID accepted an empty revision")
	}
}

func TestCommonDirFastPathOwnsLinkedWorktreeLayout(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "tracked.txt", "one\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-m", "initial")
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, repo, "worktree", "add", "--detach", linked, "HEAD")

	t.Setenv("PATH", "")
	for _, worktree := range []string{repo, linked} {
		commonDir, err := CommonDir(worktree)
		if err != nil {
			t.Fatalf("CommonDir(%s): %v", worktree, err)
		}
		if filepath.Clean(commonDir) != filepath.Join(repo, ".git") {
			t.Fatalf("CommonDir(%s) = %q", worktree, commonDir)
		}
	}
}

func TestCommonDirFallsBackToGitForRepositorySubdirectory(t *testing.T) {
	repo := newGitRepo(t)
	nested := filepath.Join(repo, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	commonDir, err := CommonDir(nested)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(commonDir) != filepath.Join(repo, ".git") {
		t.Fatalf("CommonDir(nested) = %q", commonDir)
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "test@example.com")
	mustGit(t, repo, "config", "user.name", "Test User")
	return repo
}

func writeGitFile(t *testing.T, repo, name, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := Run(repo, args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(out)
}
