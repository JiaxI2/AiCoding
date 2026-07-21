package gitx

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveStreamsTrackedTreeOnly(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "tracked.txt", "tracked\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-m", "initial")
	writeGitFile(t, repo, "untracked.txt", "untracked\n")

	var archive bytes.Buffer
	if err := Archive(context.Background(), repo, "HEAD", &archive); err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(archive.Bytes()))
	files := map[string]string{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.ToSlash(header.Name)] = strings.ReplaceAll(string(content), "\r\n", "\n")
	}
	if files["tracked.txt"] != "tracked\n" || files["untracked.txt"] != "" {
		t.Fatalf("archive files = %#v", files)
	}
}

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
	if !filepath.IsAbs(commonDir) || filepath.Clean(commonDir) != canonicalGitTestPath(t, filepath.Join(repo, ".git")) {
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

func TestTreeEntriesReturnsTrackedObjectsWithoutWorktreeFiles(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "z.txt", "z\n")
	writeGitFile(t, repo, "nested/a.go", "package nested\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-m", "tree")
	tree, err := TreeOID(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := TreeEntries(repo, tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Path != "nested/a.go" || entries[1].Path != "z.txt" {
		t.Fatalf("tree entries = %#v", entries)
	}
	for _, entry := range entries {
		if entry.Mode != "100644" || entry.Type != "blob" || len(entry.OID) != 40 {
			t.Fatalf("invalid tree entry = %#v", entry)
		}
	}
	if _, err := TreeEntries(repo, ""); err == nil {
		t.Fatal("TreeEntries accepted an empty tree")
	}
}

func TestParsePushUpdates(t *testing.T) {
	zero := strings.Repeat("0", 40)
	local := strings.Repeat("a", 40)
	remote := strings.Repeat("b", 40)
	updates, err := ParsePushUpdates(strings.NewReader(
		"refs/heads/feature " + local + " refs/heads/feature " + remote + "\n" +
			"(delete) " + zero + " refs/tags/v1.0.0 " + remote + "\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 || updates[0].LocalOID != local || updates[1].LocalRef != "(delete)" || updates[1].LocalOID != zero {
		t.Fatalf("updates = %#v", updates)
	}
	if _, err := ParsePushUpdates(strings.NewReader("refs/heads/main bad refs/heads/main " + zero)); err == nil {
		t.Fatal("ParsePushUpdates accepted an invalid object id")
	}
	if _, err := ParsePushUpdates(strings.NewReader("only three fields")); err == nil {
		t.Fatal("ParsePushUpdates accepted a malformed record")
	}
}

func TestIsAncestorDistinguishesNegativeAnswerFromFailure(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "tracked.txt", "one\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-m", "first")
	first := mustGit(t, repo, "rev-parse", "HEAD")
	writeGitFile(t, repo, "tracked.txt", "two\n")
	mustGit(t, repo, "commit", "-am", "second")
	second := mustGit(t, repo, "rev-parse", "HEAD")

	ancestor, err := IsAncestor(repo, first, second)
	if err != nil || !ancestor {
		t.Fatalf("first ancestor of second = %v, %v", ancestor, err)
	}
	ancestor, err = IsAncestor(repo, second, first)
	if err != nil || ancestor {
		t.Fatalf("second ancestor of first = %v, %v", ancestor, err)
	}
	if _, err := IsAncestor(repo, "missing", second); err == nil {
		t.Fatal("IsAncestor treated an invalid revision as a negative answer")
	}
}

func TestCommonDirFastPathOwnsLinkedWorktreeLayout(t *testing.T) {
	repo := newGitRepo(t)
	writeGitFile(t, repo, "tracked.txt", "one\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-m", "initial")
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, repo, "worktree", "add", "--detach", linked, "HEAD")
	want := canonicalGitTestPath(t, filepath.Join(repo, ".git"))

	t.Setenv("PATH", "")
	for _, worktree := range []string{repo, linked} {
		commonDir, err := CommonDir(worktree)
		if err != nil {
			t.Fatalf("CommonDir(%s): %v", worktree, err)
		}
		if filepath.Clean(commonDir) != want {
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
	if filepath.Clean(commonDir) != canonicalGitTestPath(t, filepath.Join(repo, ".git")) {
		t.Fatalf("CommonDir(nested) = %q", commonDir)
	}
}

func canonicalGitTestPath(t *testing.T, path string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(canonical)
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
