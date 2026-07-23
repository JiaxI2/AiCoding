package kit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var kitTestGitTemplate string

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "aicoding-kit-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "internal/kit TestMain: create temp root: %v\n", err)
		os.Exit(1)
	}
	kitTestGitTemplate = filepath.Join(root, "git-template")
	if err := os.MkdirAll(kitTestGitTemplate, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "internal/kit TestMain: create Git template: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	command := exec.Command("git", "init", "-q")
	command.Dir = kitTestGitTemplate
	if output, err := command.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "internal/kit TestMain: initialize Git template: %v\n%s", err, output)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}

	exitCode := m.Run()
	if err := os.RemoveAll(root); err != nil && exitCode == 0 {
		fmt.Fprintf(os.Stderr, "internal/kit TestMain: remove temp root: %v\n", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func initKitTestGitRepo(t *testing.T, repo string) {
	t.Helper()
	err := filepath.WalkDir(kitTestGitTemplate, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(kitTestGitTemplate, path)
		if err != nil {
			return err
		}
		target := filepath.Join(repo, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy TestMain Git template: %v", err)
	}
}
