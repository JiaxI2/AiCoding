package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var cliTestBinary string
var cliTestGitTemplate string

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "aicoding-cli-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "internal/cli TestMain: create temp root: %v\n", err)
		os.Exit(1)
	}

	cliTestGitTemplate = filepath.Join(root, "git-template")
	if err := os.MkdirAll(cliTestGitTemplate, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "internal/cli TestMain: create Git template: %v\n", err)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}
	initCommand := exec.Command("git", "init", "-q")
	initCommand.Dir = cliTestGitTemplate
	if output, err := initCommand.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "internal/cli TestMain: initialize Git template: %v\n%s", err, output)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}

	cliTestBinary = filepath.Join(root, "aicoding-test.exe")
	buildCommand := exec.Command("go", "build", "-o", cliTestBinary, "../../cmd/aicoding")
	if output, err := buildCommand.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "internal/cli TestMain: build aicoding test binary: %v\n%s", err, output)
		_ = os.RemoveAll(root)
		os.Exit(1)
	}

	exitCode := m.Run()
	if err := os.RemoveAll(root); err != nil && exitCode == 0 {
		fmt.Fprintf(os.Stderr, "internal/cli TestMain: remove temp root: %v\n", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func initCLITestGitRepo(t *testing.T, repo string) {
	t.Helper()
	err := filepath.WalkDir(cliTestGitTemplate, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(cliTestGitTemplate, path)
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
