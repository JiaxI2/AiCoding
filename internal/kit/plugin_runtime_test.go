package kit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncPlatformPluginRefreshesDriftThroughCodexCLI(t *testing.T) {
	repo := t.TempDir()
	codexHome := t.TempDir()
	manifest, sourceBuildInfo := writePlatformPluginFixture(t, repo)
	installedPackage := filepath.Join(codexHome, "plugins", "cache", "aicoding-platform", "aicoding", "0.1.0")
	mustWritePluginRuntime(t, filepath.Join(installedPackage, "BUILDINFO.json"), strings.Replace(sourceBuildInfo, `"skillsDigest":"current-skills"`, `"skillsDigest":"stale-skills"`, 1))
	mustWritePluginRuntime(t, filepath.Join(codexHome, "config.toml"), "[plugins.\"aicoding@aicoding-platform\"]\nenabled = true\n")

	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("AICODING_CODEX_CLI", "fake-codex")
	previousRunner := codexPluginCommand
	t.Cleanup(func() { codexPluginCommand = previousRunner })

	commands := []string{}
	codexPluginCommand = func(path string, args ...string) ([]byte, error) {
		commands = append(commands, strings.Join(args, " "))
		if len(args) >= 2 && args[1] == "add" {
			mustWritePluginRuntime(t, filepath.Join(installedPackage, "BUILDINFO.json"), sourceBuildInfo)
		}
		return []byte(`{"ok":true}`), nil
	}

	result, err := syncPlatformPlugin(repo, manifest)
	if err != nil {
		t.Fatalf("syncPlatformPlugin: %v", err)
	}
	if result.Drift || !result.Refreshed || result.ManualRequired {
		t.Fatalf("unexpected sync result: %#v", result)
	}
	if len(commands) != 2 || !strings.HasPrefix(commands[0], "plugin remove ") || !strings.HasPrefix(commands[1], "plugin add ") {
		t.Fatalf("expected remove/add refresh, got %#v", commands)
	}
}

func TestSyncPlatformPluginPreservesDisabledState(t *testing.T) {
	repo := t.TempDir()
	codexHome := t.TempDir()
	manifest, sourceBuildInfo := writePlatformPluginFixture(t, repo)
	installedPackage := filepath.Join(codexHome, "plugins", "cache", "aicoding-platform", "aicoding", "0.1.0")
	mustWritePluginRuntime(t, filepath.Join(installedPackage, "BUILDINFO.json"), strings.Replace(sourceBuildInfo, `"sourceCommit":"current"`, `"sourceCommit":"stale"`, 1))
	mustWritePluginRuntime(t, filepath.Join(codexHome, "config.toml"), "[plugins.\"aicoding@aicoding-platform\"]\nenabled = false\n")

	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("AICODING_CODEX_CLI", "fake-codex")
	previousRunner := codexPluginCommand
	t.Cleanup(func() { codexPluginCommand = previousRunner })
	called := false
	codexPluginCommand = func(path string, args ...string) ([]byte, error) {
		called = true
		return nil, nil
	}

	result, err := syncPlatformPlugin(repo, manifest)
	if err == nil || !result.ManualRequired {
		t.Fatalf("expected manual refresh for disabled plugin: result=%#v err=%v", result, err)
	}
	if called {
		t.Fatal("disabled plugin must not be removed or re-enabled automatically")
	}
}

func TestSyncPlatformPluginReportsMissingCLI(t *testing.T) {
	repo := t.TempDir()
	codexHome := t.TempDir()
	manifest, _ := writePlatformPluginFixture(t, repo)
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("AICODING_CODEX_CLI", "")
	t.Setenv("LOCALAPPDATA", t.TempDir())

	previousLookPath := codexLookPath
	t.Cleanup(func() { codexLookPath = previousLookPath })
	codexLookPath = func(string) (string, error) { return "", errors.New("not found") }

	result, err := syncPlatformPlugin(repo, manifest)
	if err == nil || !result.ManualRequired {
		t.Fatalf("expected actionable missing CLI error: result=%#v err=%v", result, err)
	}
	if !strings.Contains(err.Error(), result.ManualCommand) || !strings.Contains(err.Error(), result.DeepLink) {
		t.Fatalf("missing manual recovery details: %v", err)
	}
}

func TestBuiltinLifecycleDoesNotWriteStateWhenPluginRefreshFails(t *testing.T) {
	repo := t.TempDir()
	codexHome := t.TempDir()
	manifest, _ := writePlatformPluginFixture(t, repo)
	manifest.State = map[string]string{"installState": ".aicoding/state/kits/aicoding-platform/install-state.json"}
	mustWritePluginRuntime(t, filepath.Join(codexHome, "config.toml"), "[plugins.\"aicoding@aicoding-platform\"]\nenabled = true\n")

	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("AICODING_CODEX_CLI", "fake-codex")
	previousRunner := codexPluginCommand
	t.Cleanup(func() { codexPluginCommand = previousRunner })
	codexPluginCommand = func(path string, args ...string) ([]byte, error) {
		return []byte("refresh failed"), errors.New("exit 1")
	}

	result := runBuiltinLifecycle(
		repo,
		RegistryKit{ID: "aicoding-platform", Manifest: "config/kits/aicoding-platform.json"},
		manifest,
		CommandDef{},
		"update",
		false,
	)
	if result.OK || result.Status != "manual-required" {
		t.Fatalf("expected failed refresh to block lifecycle state: %#v", result)
	}
	state := filepath.Join(repo, ".aicoding", "state", "kits", "aicoding-platform", "install-state.json")
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("install state must not be written after refresh failure: %v", err)
	}
}

func TestWriteInstallStateRecordsVerifiedPluginIdentity(t *testing.T) {
	repo := t.TempDir()
	manifest := Manifest{
		ID:      "aicoding-platform",
		Version: "0.1.0",
		State:   map[string]string{"installState": ".aicoding/state/kits/aicoding-platform/install-state.json"},
	}
	sync := PlatformPluginSync{
		InstalledPackage: filepath.Join(t.TempDir(), "plugins", "cache", "aicoding-platform", "aicoding", "0.1.0"),
		SourceBuildInfo: pluginBuildInfo{
			SourceCommit: "verified-commit",
			SkillsDigest: "verified-digest",
		},
	}
	entry := RegistryKit{ID: "aicoding-platform", Manifest: "config/kits/aicoding-platform.json"}
	if err := writeInstallState(repo, entry, manifest, "update", &sync); err != nil {
		t.Fatalf("writeInstallState: %v", err)
	}
	state, err := readInstallState(filepath.Join(repo, ".aicoding", "state", "kits", "aicoding-platform", "install-state.json"))
	if err != nil {
		t.Fatalf("readInstallState: %v", err)
	}
	if state.PluginSourceCommit != "verified-commit" || state.PluginSkillsDigest != "verified-digest" || state.PluginCachePath != sync.InstalledPackage {
		t.Fatalf("plugin identity was not persisted: %#v", state)
	}
}

func writePlatformPluginFixture(t *testing.T, repo string) (Manifest, string) {
	t.Helper()
	pluginRoot := filepath.Join(repo, "CodingKit", "agents", "skills", "plugins", "AiCoding")
	marketplacePath := filepath.Join(repo, ".agents", "plugins", "marketplace.json")
	mustWritePluginRuntime(t, filepath.Join(pluginRoot, ".codex-plugin", "plugin.json"), `{"name":"aicoding","version":"0.1.0"}`)
	buildInfo := `{"pluginName":"aicoding","pluginVersion":"0.1.0","sourceCommit":"current","sourceTag":null,"packManifestHash":"pack","pluginManifestHash":"plugin","skillsDigest":"current-skills","hooksDigest":"current-hooks","dirtySource":false}`
	mustWritePluginRuntime(t, filepath.Join(pluginRoot, "BUILDINFO.json"), buildInfo)
	mustWritePluginRuntime(t, marketplacePath, `{"name":"aicoding-platform","plugins":[{"name":"aicoding"}]}`)
	return Manifest{
		ID: "aicoding-platform",
		Paths: map[string]string{
			"pluginRoot":  filepath.ToSlash(filepath.Join("CodingKit", "agents", "skills", "plugins", "AiCoding")),
			"marketplace": filepath.ToSlash(filepath.Join(".agents", "plugins", "marketplace.json")),
		},
	}, buildInfo
}

func mustWritePluginRuntime(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
