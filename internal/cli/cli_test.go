package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunNewFastPathCommands(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/repo\n\ngo 1.22\n")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "tasks:\n  smoke:\n    cmds:\n      - bin/aicoding.exe kit verify --all --profile Smoke --json\n")
	mustWrite(t, filepath.Join(repo, "config", "tagging-policy.json"), `{"schemaVersion":1}`)
	writeReleaseFixture(t, repo)

	start := time.Now()
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"bootstrap", func() error {
			res, err := runBootstrap([]string{"--repo-root", repo, "--no-build", "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"cache status", func() error {
			res, err := runCache([]string{"status", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"doctor pwsh-budget", func() error {
			res, err := runDoctor([]string{"pwsh-budget", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"tag audit", func() error {
			res, err := runTag([]string{"audit", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
		{"release verify", func() error {
			res, err := runRelease([]string{"verify", "--repo-root", repo, "--json"}, start)
			return resultErr(res.OK, err)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunWorkflowSmartVerifyCommand(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	mustWrite(t, filepath.Join(repo, "README.md"), "# test\n")

	res, err := runWorkflow([]string{"smart-verify", "--repo-root", repo, "--json"}, time.Now())
	if err == nil || res.OK {
		t.Fatalf("expected workflow to fail on incomplete fixture repo, got res=%#v err=%v", res, err)
	}
}

func resultErr(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return os.ErrInvalid
	}
	return nil
}

func writeReleaseFixture(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "CHANGELOG.md"), "# CHANGELOG\n\n## [Unreleased]\n")
	mustWrite(t, filepath.Join(repo, ".github", "RELEASE_TEMPLATE.md"), "## 摘要 / Summary\n\n## 变更内容 / What's Changed\n\n## 可追溯性 / Traceability\n")
	mustWrite(t, filepath.Join(repo, "docs", "TAGGING_POLICY.md"), "vMAJOR.MINOR.PATCH\nkit/<kit-id>/vMAJOR.MINOR.PATCH\nmilestone/YYYY.MM.DD-<name>\n")
	mustWrite(t, filepath.Join(repo, "docs", "RELEASE_POLICY.md"), "Platform Release\nKit / Component Release\nMilestone / Historical Snapshot\n")
	for _, rel := range []string{
		"docs/RELEASE_GOVERNANCE_OVERLAY.md",
		"scripts/aicoding-tag-governance.ps1",
		"scripts/verify-release-governance-overlay.ps1",
		"config/kits/release-governance-overlay-kit.json",
		".aicoding/templates/perf-cache-plan.json",
	} {
		mustWrite(t, filepath.Join(repo, filepath.FromSlash(rel)), "ok\n")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMainSwitchRoutesNewCommands(t *testing.T) {
	repo := t.TempDir()
	cmd := exec.Command("go", "run", "../../cmd/aicoding", "cache", "status", "--repo-root", repo, "--json")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run cache status: %v: %s", err, out)
	}
	if !strings.Contains(string(out), `"command": "cache status"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}
