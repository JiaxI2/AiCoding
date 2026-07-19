package repoinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestInitIsIdempotentAndGitNative(t *testing.T) {
	repo := t.TempDir()
	if _, err := gitx.Run(repo, "--version"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}

	// First run on a non-git dir: creates the repo and wires everything.
	first := Init(repo)
	if !first.OK || !first.GitInitialized || first.GitAlreadyRepo {
		t.Fatalf("first init should create the repo: %#v", first)
	}
	if first.HooksPath != ".githooks" {
		t.Fatalf("hooks not wired: %#v", first)
	}
	if first.ConfigMarkers["aicoding.initialized"] != "true" {
		t.Fatalf("markers not written: %#v", first.ConfigMarkers)
	}
	if _, err := os.Stat(filepath.Join(repo, ".aicoding")); err != nil {
		t.Fatalf(".aicoding home not created: %v", err)
	}

	// Markers live in .git/config (local, per-clone), readable via git itself.
	out, err := gitx.Run(repo, "config", "--get", "core.hooksPath")
	if err != nil || trimLine(out) != ".githooks" {
		t.Fatalf("core.hooksPath not persisted in git config: %q %v", out, err)
	}

	// Status reads setup state fast from git config without scanning the tree.
	markers, initialized := Status(repo)
	if !initialized || markers["aicoding.home"] != ".aicoding" {
		t.Fatalf("Status did not read markers: %v %v", markers, initialized)
	}

	// Second run is idempotent: the repo already exists, nothing is re-created.
	second := Init(repo)
	if !second.OK || second.GitInitialized || !second.GitAlreadyRepo {
		t.Fatalf("second init should be idempotent: %#v", second)
	}
}

func TestStatusOnUninitializedRepo(t *testing.T) {
	repo := t.TempDir()
	if _, err := gitx.Run(repo, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	// A git repo that never ran `aicoding init` reports not-initialized.
	_, initialized := Status(repo)
	if initialized {
		t.Fatal("uninitialized repo should not report aicoding.initialized")
	}
}
