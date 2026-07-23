package testengine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var engineTestGitTemplate string

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "aicoding-test-engine-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "internal/testengine TestMain: create temp root: %v\n", err)
		os.Exit(1)
	}
	engineTestGitTemplate = filepath.Join(root, "git-template")
	if err := os.MkdirAll(engineTestGitTemplate, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "internal/testengine TestMain: create Git template: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	command := exec.Command("git", "init", "-q")
	command.Dir = engineTestGitTemplate
	if output, err := command.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "internal/testengine TestMain: initialize Git template: %v\n%s", err, output)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}

	exitCode := m.Run()
	if err := os.RemoveAll(root); err != nil && exitCode == 0 {
		fmt.Fprintf(os.Stderr, "internal/testengine TestMain: remove temp root: %v\n", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func initEngineTestGitRepo(t *testing.T, repo string) {
	t.Helper()
	err := filepath.WalkDir(engineTestGitTemplate, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(engineTestGitTemplate, path)
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
