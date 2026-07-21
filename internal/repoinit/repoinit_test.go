package repoinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestInitIsIdempotentAndGitNative(t *testing.T) {
	repo := t.TempDir()
	if _, err := gitx.Run(repo, "--version"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}

	first := Init(repo)
	if !first.OK || !first.GitInitialized || first.GitAlreadyRepo {
		t.Fatalf("first init should create the repo: %#v", first)
	}
	if first.HooksPath != ".githooks" || len(first.DocsSkeleton) != len(docsSkeleton) {
		t.Fatalf("provisioned surfaces are incomplete: %#v", first)
	}
	for _, action := range first.Actions {
		if !strings.HasPrefix(action, "created ") {
			t.Fatalf("first-run action is not created: %q", action)
		}
	}
	for key, want := range map[string]string{
		"aicoding.initialized": "true", "aicoding.home": ".aicoding",
		"aicoding.schemaVersion": SchemaVersion, "aicoding.docsSkeleton": "1",
	} {
		if first.ConfigMarkers[key] != want {
			t.Fatalf("marker %s = %q, want %q", key, first.ConfigMarkers[key], want)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".aicoding")); err != nil {
		t.Fatalf(".aicoding home not created: %v", err)
	}
	for _, relative := range docsSkeleton {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("skeleton %s: %v", relative, err)
		}
		if lines := len(strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")); lines > 15 {
			t.Fatalf("skeleton %s has %d lines, want at most 15", relative, lines)
		}
	}
	hub, err := os.ReadFile(filepath.Join(repo, "docs", "README.md"))
	if err != nil || !strings.Contains(string(hub), "AICODING:REPOSITORY_MAP:START") || !strings.Contains(string(hub), "AICODING:REPOSITORY_MAP:END") {
		t.Fatalf("docs hub markers missing: %v %q", err, hub)
	}

	out, err := gitx.Run(repo, "config", "--get", "core.hooksPath")
	if err != nil || trimLine(out) != ".githooks" {
		t.Fatalf("core.hooksPath not persisted in git config: %q %v", out, err)
	}
	markers, initialized := Status(repo)
	if !initialized || markers["aicoding.home"] != ".aicoding" || markers["aicoding.docsSkeleton"] != "1" {
		t.Fatalf("Status did not read markers: %v %v", markers, initialized)
	}

	statusBefore := mustGit(t, repo, "status", "--porcelain", "--untracked-files=all")
	assertOnlySkeletonUntracked(t, statusBefore)
	configBefore, err := os.ReadFile(filepath.Join(repo, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	second := Init(repo)
	if !second.OK || second.GitInitialized || !second.GitAlreadyRepo {
		t.Fatalf("second init should be idempotent: %#v", second)
	}
	for _, action := range second.Actions {
		if !strings.HasPrefix(action, "kept ") {
			t.Fatalf("second-run action is not kept: %q", action)
		}
	}
	statusAfter := mustGit(t, repo, "status", "--porcelain", "--untracked-files=all")
	configAfter, err := os.ReadFile(filepath.Join(repo, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if statusAfter != statusBefore || string(configAfter) != string(configBefore) {
		t.Fatalf("second provision changed state: status before=%q after=%q", statusBefore, statusAfter)
	}
}

func TestInitNeverOverwritesExistingSkeletonFile(t *testing.T) {
	repo := t.TempDir()
	if first := Init(repo); !first.OK {
		t.Fatalf("first Init: %#v", first)
	}
	path := filepath.Join(repo, "docs", "README.md")
	const custom = "# Repository-owned documentation\n"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	third := Init(repo)
	if !third.OK || !containsAction(third.Actions, "kept docs/README.md") {
		t.Fatalf("existing docs README was not kept: %#v", third)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != custom {
		t.Fatalf("existing docs README was overwritten: %v %q", err, content)
	}
}

func TestStatusOnUninitializedRepo(t *testing.T) {
	repo := t.TempDir()
	if _, err := gitx.Run(repo, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	_, initialized := Status(repo)
	if initialized {
		t.Fatal("uninitialized repo should not report aicoding.initialized")
	}
}

func assertOnlySkeletonUntracked(t *testing.T, status string) {
	t.Helper()
	want := make(map[string]bool, len(docsSkeleton))
	for _, relative := range docsSkeleton {
		want["?? "+relative] = true
	}
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(status, "\\", "/")), "\n")
	if len(lines) != len(want) {
		t.Fatalf("git status has %d entries, want %d: %q", len(lines), len(want), status)
	}
	for _, line := range lines {
		if !want[strings.TrimSpace(line)] {
			t.Fatalf("git status contains non-skeleton path: %q", line)
		}
	}
}

func containsAction(actions []string, want string) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}

func mustGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := gitx.Run(repo, args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(out)
}
