package mcpcontrol

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type lifecycleFixture struct {
	repo       string
	configPath string
	original   string
	entry      RegistryEntry
	component  Component
	snapshots  []ComponentSnapshot
	fakePython string
	venvPython string
}

func TestLifecycleStateWriteFailureRestoresManagedConfig(t *testing.T) {
	fixture := newLifecycleFixture(t)
	blockedStateDir := filepath.Dir(statePath(fixture.repo, fixture.component.ID))
	writeTestFile(t, blockedStateDir, "state directory is intentionally blocked\n")

	results := RunCatalogLifecycle(
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
		"update",
		false,
	)
	result := requireSingleLifecycleResult(t, results)
	if result.OK || result.Status != "failed" || len(result.Errors) == 0 {
		t.Fatalf("state-write failure unexpectedly passed: %#v", result)
	}
	if result.BackupPath == "" {
		t.Fatalf("managed config was not written before state failure: %#v", result)
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil || string(backup) != fixture.original {
		t.Fatalf("config backup is not the pre-operation state: err=%v content=%q", err, backup)
	}
	config, err := os.ReadFile(fixture.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != fixture.original {
		t.Fatalf(
			"config was not restored after writeInstallState failure:\nwant=%q\ngot=%q\nresult=%#v",
			fixture.original,
			config,
			result,
		)
	}
	if platformState := statePath(fixture.repo, fixture.component.ID); platformState == "" {
		t.Fatal("state path was not resolved")
	}
}

func TestLifecycleEntryInstallDryRunAndUninstall(t *testing.T) {
	fixture := newLifecycleFixture(t)

	planned := requireSingleLifecycleResult(t, RunLifecycle(
		fixture.repo,
		fixture.configPath,
		[]RegistryEntry{fixture.entry},
		"install",
		true,
	))
	if !planned.OK || planned.Status != "planned" || !planned.DryRun {
		t.Fatalf("install dry-run did not produce a plan: %#v", planned)
	}
	if data, err := os.ReadFile(fixture.configPath); err != nil || string(data) != fixture.original {
		t.Fatalf("install dry-run changed config: err=%v content=%q", err, data)
	}

	installed := requireSingleLifecycleResult(t, RunLifecycle(
		fixture.repo,
		fixture.configPath,
		[]RegistryEntry{fixture.entry},
		"install",
		false,
	))
	if !installed.OK || installed.Status != "installed" || installed.BackupPath == "" {
		t.Fatalf("install failed: %#v", installed)
	}
	assertManagedConfig(t, fixture.configPath, fixture.component.Codex.ServerName, true)
	if _, err := os.Stat(installed.StatePath); err != nil {
		t.Fatalf("install state was not written: %v", err)
	}

	uninstalled := requireSingleLifecycleResult(t, RunLifecycle(
		fixture.repo,
		fixture.configPath,
		[]RegistryEntry{fixture.entry},
		"uninstall",
		false,
	))
	if !uninstalled.OK || uninstalled.Status != "uninstalled" || uninstalled.BackupPath == "" {
		t.Fatalf("uninstall failed: %#v", uninstalled)
	}
	if data, err := os.ReadFile(fixture.configPath); err != nil || string(data) != fixture.original {
		t.Fatalf("uninstall did not remove the managed block: err=%v content=%q", err, data)
	}
	if _, err := os.Stat(fixture.venvPython); !os.IsNotExist(err) {
		t.Fatalf("uninstall retained the owned venv: %v", err)
	}
	if _, err := os.Stat(installed.StatePath); !os.IsNotExist(err) {
		t.Fatalf("uninstall retained install state: %v", err)
	}
}

func TestLifecycleCatalogDryRunAndUnsupportedAction(t *testing.T) {
	fixture := newLifecycleFixture(t)
	planned := requireSingleLifecycleResult(t, RunCatalogLifecycle(
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
		"update",
		true,
	))
	if !planned.OK || planned.Status != "planned" || planned.Action != "update" {
		t.Fatalf("catalog update dry-run failed: %#v", planned)
	}

	unsupported := requireSingleLifecycleResult(t, RunCatalogLifecycle(
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
		"replace",
		false,
	))
	if unsupported.OK || !strings.Contains(strings.Join(unsupported.Errors, " "), "unsupported") {
		t.Fatalf("unsupported action was not rejected: %#v", unsupported)
	}
}

func TestLifecycleEntryAndCatalogStatus(t *testing.T) {
	fixture := newLifecycleFixture(t)
	backup, err := writeManagedBlock(
		fixture.configPath,
		fixture.component,
		fixture.venvPython,
		componentRoot(fixture.repo, fixture.component),
	)
	if err != nil || backup == "" {
		t.Fatalf("status fixture config write failed: backup=%q err=%v", backup, err)
	}
	if err := writeInstallState(
		statePath(fixture.repo, fixture.component.ID),
		fixture.component,
		fixture.configPath,
	); err != nil {
		t.Fatal(err)
	}

	entryStatus := requireSingleStatusResult(t, Status(
		fixture.repo,
		fixture.configPath,
		[]RegistryEntry{fixture.entry},
	))
	assertHealthyStatus(t, entryStatus)

	catalogStatus := requireSingleStatusResult(t, StatusCatalog(
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
	))
	assertHealthyStatus(t, catalogStatus)

	if err := os.Remove(statePath(fixture.repo, fixture.component.ID)); err != nil {
		t.Fatal(err)
	}
	drift := requireSingleStatusResult(t, StatusCatalog(
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
	))
	if !drift.OK || len(drift.Warnings) != 1 || !strings.Contains(drift.Warnings[0], "inconsistent") {
		t.Fatalf("venv/state drift was not reported: %#v", drift)
	}
}

func TestLifecycleStatusReportsLoadAndConfigFailures(t *testing.T) {
	repo := t.TempDir()
	missing := Status(repo, filepath.Join(repo, "missing", "config.toml"), []RegistryEntry{{
		ID:       "missing",
		Manifest: "config/mcp/components/missing.json",
	}})
	result := requireSingleStatusResult(t, missing)
	if result.OK || result.ID != "missing" || len(result.Errors) == 0 {
		t.Fatalf("missing manifest was not reported: %#v", result)
	}

	fixture := newLifecycleFixture(t)
	writeTestFile(t, fixture.configPath, "[mcp_servers.visio-mcp]\ncommand = \"user.exe\"\n")
	collision := requireSingleStatusResult(t, StatusCatalog(
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
	))
	if collision.OK || !collision.UnmanagedCollision || len(collision.Errors) == 0 {
		t.Fatalf("unmanaged collision was not reported: %#v", collision)
	}
}

func TestLifecycleEntryAndCatalogVerify(t *testing.T) {
	fixture := newLifecycleFixture(t)

	entryReport := Verify(
		context.Background(),
		fixture.repo,
		fixture.configPath,
		[]RegistryEntry{fixture.entry},
		"smoke",
		false,
	)
	assertSuccessfulVerify(t, entryReport, "Smoke")

	catalogReport := VerifyCatalog(
		context.Background(),
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
		"Release",
		true,
	)
	assertSuccessfulVerify(t, catalogReport, "Release")
	if len(catalogReport.Configured) != 0 || len(catalogReport.Warnings) != 1 {
		t.Fatalf("empty configured inventory was not reported: %#v", catalogReport)
	}
	if err := VerifyErrors(catalogReport); err != nil {
		t.Fatalf("successful verify returned an error: %v", err)
	}
}

func TestLifecycleVerifyReportsLoadAndProfileFailures(t *testing.T) {
	repo := t.TempDir()
	loadFailure := Verify(
		context.Background(),
		repo,
		filepath.Join(repo, "config.toml"),
		[]RegistryEntry{{ID: "missing", Manifest: "missing.json"}},
		"Full",
		false,
	)
	if loadFailure.OK || len(loadFailure.Managed) != 1 || len(loadFailure.Errors) != 1 {
		t.Fatalf("missing component was not reported: %#v", loadFailure)
	}
	if err := VerifyErrors(loadFailure); err == nil || !strings.Contains(err.Error(), "1 errors") {
		t.Fatalf("verify error summary is missing: %v", err)
	}

	fixture := newLifecycleFixture(t)
	undefined := VerifyCatalog(
		context.Background(),
		fixture.repo,
		fixture.configPath,
		fixture.snapshots,
		"Canary",
		false,
	)
	if undefined.OK || len(undefined.Managed) != 1 {
		t.Fatalf("undefined profile unexpectedly passed: %#v", undefined)
	}
	if !strings.Contains(strings.Join(undefined.Errors, " "), "profile is not defined") {
		t.Fatalf("undefined profile error is missing: %#v", undefined)
	}
}

func TestWriteInstallStatePreservesInstalledTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "install-state.json")
	component := testComponent()
	installedAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	previous := installState{
		SchemaVersion: 1,
		ComponentID:   component.ID,
		Version:       "0.0.1",
		InstalledAt:   installedAt,
		UpdatedAt:     installedAt,
		CodexConfig:   "old.toml",
	}
	data, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(data))
	if err := writeInstallState(path, component, "new.toml"); err != nil {
		t.Fatal(err)
	}

	updatedData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var updated installState
	if err := json.Unmarshal(updatedData, &updated); err != nil {
		t.Fatal(err)
	}
	if !updated.InstalledAt.Equal(installedAt) || !updated.UpdatedAt.After(installedAt) {
		t.Fatalf("install timestamps were not preserved/updated: %#v", updated)
	}
	if updated.Version != component.Version || updated.CodexConfig != "new.toml" {
		t.Fatalf("install state fields were not refreshed: %#v", updated)
	}
}

func TestRestoreConfigBackupSuccessNoopAndMissingBackup(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	backupPath := filepath.Join(directory, "config.toml.bak")
	writeTestFile(t, configPath, "changed\n")
	writeTestFile(t, backupPath, "original\n")

	if err := restoreConfigBackup(configPath, backupPath); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(configPath); err != nil || string(data) != "original\n" {
		t.Fatalf("backup was not restored: err=%v content=%q", err, data)
	}
	if err := restoreConfigBackup(configPath, ""); err != nil {
		t.Fatalf("empty backup should be a no-op: %v", err)
	}
	if err := restoreConfigBackup(configPath, filepath.Join(directory, "missing.bak")); !os.IsNotExist(err) {
		t.Fatalf("missing backup did not return its read error: %v", err)
	}
}

func newLifecycleFixture(t *testing.T) lifecycleFixture {
	t.Helper()
	repo := t.TempDir()
	component := testComponent()
	component.SchemaVersion = 1
	component.Transport = "stdio"
	component.Runtime.Kind = "python-venv"
	component.Runtime.Requirements = "requirements.txt"
	component.Runtime.MinimumPython = "3.10"
	component.Runtime.Module = "visio_mcp"
	component.Doctor.Args = []string{"--doctor"}
	component.Verify = map[string][][]string{
		"Smoke":   {{"--verify", "smoke"}},
		"Full":    {{"--verify", "full"}},
		"Release": {{"--verify", "release"}},
	}
	entry := RegistryEntry{
		ID:       component.ID,
		Enabled:  true,
		Order:    10,
		Manifest: "config/mcp/components/visio-mcp.json",
	}
	writeJSONTestFile(t, filepath.Join(repo, "config", "mcp-registry.json"), Registry{
		SchemaVersion: 1,
		Name:          "lifecycle-test",
		Components:    []RegistryEntry{entry},
	})
	writeJSONTestFile(t, filepath.Join(repo, filepath.FromSlash(entry.Manifest)), component)
	writeTestFile(t, filepath.Join(componentRoot(repo, component), component.Runtime.Requirements), "")

	fakePython := buildFakePython(t)
	venv := venvPython(componentRoot(repo, component))
	copyExecutable(t, fakePython, venv)
	t.Setenv(component.Runtime.PythonEnvVar, fakePython)

	configPath := filepath.Join(repo, "codex", "config.toml")
	original := "personality = \"pragmatic\"\n"
	writeTestFile(t, configPath, original)
	catalog, err := LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	return lifecycleFixture{
		repo:       repo,
		configPath: configPath,
		original:   original,
		entry:      entry,
		component:  component,
		snapshots:  catalog.Components(),
		fakePython: fakePython,
		venvPython: venv,
	}
}

func buildFakePython(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "main.go")
	writeTestFile(t, sourcePath, `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-c" {
		fmt.Println("3.12")
		return
	}
	fmt.Println("{\"ok\":true}")
}
`)
	executable := filepath.Join(directory, "fake-python")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	command := exec.Command("go", "build", "-o", executable, sourcePath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fake Python: %v\n%s", err, output)
	}
	return executable
}

func copyExecutable(t *testing.T, source, target string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeJSONTestFile(t *testing.T, path string, value interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(append(data, '\n')))
}

func requireSingleLifecycleResult(t *testing.T, results []LifecycleResult) LifecycleResult {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("expected one lifecycle result, got %#v", results)
	}
	return results[0]
}

func requireSingleStatusResult(t *testing.T, results []StatusResult) StatusResult {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("expected one status result, got %#v", results)
	}
	return results[0]
}

func assertManagedConfig(t *testing.T, configPath, serverName string, expected bool) {
	t.Helper()
	managed, collision, err := managedBlockStatus(configPath, serverName)
	if err != nil || managed != expected || collision {
		t.Fatalf("unexpected managed config status: managed=%v collision=%v err=%v", managed, collision, err)
	}
}

func assertHealthyStatus(t *testing.T, result StatusResult) {
	t.Helper()
	if !result.OK || !result.RootExists || !result.Installed || !result.Registered || !result.StateExists {
		t.Fatalf("component status is not healthy: %#v", result)
	}
	if len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("healthy status contains findings: %#v", result)
	}
}

func assertSuccessfulVerify(t *testing.T, report VerifyReport, profile string) {
	t.Helper()
	if !report.OK || report.Profile != profile || len(report.Managed) != 1 || len(report.Errors) != 0 {
		t.Fatalf("verify report failed: %#v", report)
	}
	managed := report.Managed[0]
	if !managed.OK || len(managed.Steps) != 1 || !managed.Steps[0].OK {
		t.Fatalf("managed verify step failed: %#v", managed)
	}
	if len(managed.Steps[0].Output) == 0 {
		t.Fatalf("verify JSON output was not captured: %#v", managed.Steps[0])
	}
}
