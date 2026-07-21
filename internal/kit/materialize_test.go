package kit

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestExtractTarStreamRejectsPathTraversal(t *testing.T) {
	var payload bytes.Buffer
	writer := tar.NewWriter(&payload)
	content := []byte("escape\n")
	if err := writer.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0o644, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	destination := filepath.Join(parent, "source")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	err := extractTarStream(bytes.NewReader(payload.Bytes()), destination)
	if err == nil || !strings.Contains(err.Error(), "unsafe tar path") {
		t.Fatalf("path traversal was not rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(parent, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("path traversal wrote outside destination: %v", err)
	}
}

func TestMaterializeSourceMatchesRecursiveGitTreesAndExcludesWorktreeFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	leaf := filepath.Join(t.TempDir(), "leaf")
	sub := filepath.Join(t.TempDir(), "sub")
	materializeInitRepo(t, leaf)
	materializeWrite(t, filepath.Join(leaf, "leaf.txt"), "leaf\n")
	materializeGit(t, leaf, "add", ".")
	materializeGit(t, leaf, "commit", "-m", "leaf")

	materializeInitRepo(t, sub)
	materializeWrite(t, filepath.Join(sub, "sub.txt"), "sub\n")
	materializeGit(t, sub, "add", ".")
	materializeGit(t, sub, "commit", "-m", "sub")
	materializeGit(t, sub, "-c", "protocol.file.allow=always", "submodule", "add", leaf, "nested/leaf")
	materializeGit(t, sub, "commit", "-am", "add nested leaf")

	materializeInitRepo(t, root)
	materializeWrite(t, filepath.Join(root, "tracked.txt"), "tracked\n")
	materializeWrite(t, filepath.Join(root, "中文目录", "滤波器.txt"), "utf-8 path\n")
	materializeGit(t, root, "add", ".")
	materializeGit(t, root, "commit", "-m", "root")
	materializeGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", sub, "modules/sub")
	materializeGit(t, root, "commit", "-am", "add submodule")
	materializeGit(t, root, "-c", "protocol.file.allow=always", "submodule", "update", "--init", "--recursive")
	materializeWrite(t, filepath.Join(root, "tracked.txt"), "worktree-only modification\n")
	materializeWrite(t, filepath.Join(root, "untracked.txt"), "untracked\n")
	materializeWrite(t, filepath.Join(root, "modules", "sub", "untracked-sub.txt"), "untracked submodule\n")

	tree, err := gitx.TreeOID(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "materialized")
	sourceRoot := filepath.Join(output, "source")
	manifestPath := filepath.Join(output, "source-manifest.json")
	manifest, err := materializeSource(context.Background(), root, tree, sourceRoot, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SourceMode != "materialized" || manifest.SuperprojectTreeOID != tree || manifest.SourceIdentity == "" {
		t.Fatalf("manifest identity = %#v", manifest)
	}
	wantSubmodules := []string{"modules/sub", "modules/sub/nested/leaf"}
	gotSubmodules := make([]string, 0, len(manifest.Submodules))
	for _, submodule := range manifest.Submodules {
		gotSubmodules = append(gotSubmodules, submodule.Path)
		if submodule.CommitOID == "" || submodule.TreeOID == "" {
			t.Fatalf("incomplete submodule manifest: %#v", submodule)
		}
	}
	if !reflect.DeepEqual(gotSubmodules, wantSubmodules) {
		t.Fatalf("submodules = %#v, want %#v", gotSubmodules, wantSubmodules)
	}
	wantFiles := []string{
		".gitmodules",
		"modules/sub/.gitmodules",
		"modules/sub/nested/leaf/leaf.txt",
		"modules/sub/sub.txt",
		"tracked.txt",
		"中文目录/滤波器.txt",
	}
	gotFiles := materializedFiles(t, sourceRoot)
	if !reflect.DeepEqual(gotFiles, wantFiles) || manifest.FileCount != len(wantFiles) {
		t.Fatalf("materialized files = %#v count=%d, want %#v", gotFiles, manifest.FileCount, wantFiles)
	}
	for _, forbidden := range []string{".git", "untracked.txt", "modules/sub/untracked-sub.txt", "source-manifest.json"} {
		if _, err := os.Lstat(filepath.Join(sourceRoot, filepath.FromSlash(forbidden))); !os.IsNotExist(err) {
			t.Fatalf("worktree-only or metadata path leaked into source: %s (%v)", forbidden, err)
		}
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("source manifest is missing outside source root: %v", err)
	}
	tracked, err := os.ReadFile(filepath.Join(sourceRoot, "tracked.txt"))
	if err != nil || strings.ReplaceAll(string(tracked), "\r\n", "\n") != "tracked\n" {
		t.Fatalf("materialized tracked content came from worktree instead of Git object: %q, %v", tracked, err)
	}
	t.Logf("sourceMode=%s identity=%s files=%d submodules=%d", manifest.SourceMode, manifest.SourceIdentity, manifest.FileCount, len(manifest.Submodules))
}

func TestFreshCloneTransportDriftWarnsOnlyForSensitiveChanges(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	materializeInitRepo(t, repo)
	materializeWrite(t, filepath.Join(repo, "README.md"), "base\n")
	materializeGit(t, repo, "add", ".")
	materializeGit(t, repo, "commit", "-m", "base")
	baseline, err := gitx.TreeOID(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := recordFreshCloneBaseline(repo, baseline); err != nil {
		t.Fatal(err)
	}
	if err := CheckFreshCloneTransportDrift(repo, baseline); err != nil {
		t.Fatalf("unchanged baseline warned: %v", err)
	}

	materializeWrite(t, filepath.Join(repo, "docs", "guide.md"), "docs\n")
	materializeGit(t, repo, "add", ".")
	materializeGit(t, repo, "commit", "-m", "docs")
	docsTree, err := gitx.TreeOID(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckFreshCloneTransportDrift(repo, docsTree); err != nil {
		t.Fatalf("unrelated docs change warned: %v", err)
	}

	materializeWrite(t, filepath.Join(repo, ".gitmodules"), "[submodule \"sample\"]\n")
	unstagedWarning := CheckFreshCloneTransportDrift(repo, docsTree)
	if unstagedWarning == nil || !strings.Contains(unstagedWarning.Error(), ".gitmodules") {
		t.Fatalf("unstaged transport change was not reported: %v", unstagedWarning)
	}
	if err := os.Remove(filepath.Join(repo, ".gitmodules")); err != nil {
		t.Fatal(err)
	}
	materializeWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "exit 0\n")
	materializeGit(t, repo, "add", ".githooks/pre-commit")
	stagedTree, err := gitx.WriteTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	stagedWarning := CheckFreshCloneTransportDrift(repo, stagedTree)
	if stagedWarning == nil || !strings.Contains(stagedWarning.Error(), ".githooks/pre-commit") {
		t.Fatalf("staged transport change was not reported: %v", stagedWarning)
	}
	t.Logf("unstaged=%q staged=%q", unstagedWarning, stagedWarning)
}

func TestFreshCloneTransportDriftWarnsWithoutBaseline(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	materializeInitRepo(t, repo)
	materializeWrite(t, filepath.Join(repo, "README.md"), "base\n")
	materializeGit(t, repo, "add", ".")
	materializeGit(t, repo, "commit", "-m", "base")
	tree, err := gitx.TreeOID(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckFreshCloneTransportDrift(repo, tree); err == nil || !strings.Contains(err.Error(), "no successful fresh-clone baseline") {
		t.Fatalf("missing baseline did not produce an advisory error: %v", err)
	}
}

func materializeInitRepo(t *testing.T, repo string) {
	t.Helper()
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	materializeGit(t, repo, "init", "-q")
	materializeGit(t, repo, "config", "user.email", "materialize@example.invalid")
	materializeGit(t, repo, "config", "user.name", "Materialize Test")
}

func materializeGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func materializeWrite(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func materializedFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{}
	err := filepath.Walk(root, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
