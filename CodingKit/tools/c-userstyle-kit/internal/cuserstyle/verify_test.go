package cuserstyle

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigWithOverlays(t *testing.T) {
	overlayPath := filepath.Join(t.TempDir(), "pdo.overlay.json")
	overlay := `{
  "id": "pdo-dynamic",
  "naming": {"modulePrefix": "PDO"},
  "documentation": {"requireFileMetadata": false},
  "scope": {"exclude": ["vendor"]}
}`
	if err := os.WriteFile(overlayPath, []byte(overlay), 0o600); err != nil {
		t.Fatal(err)
	}
	finalOverlayPath := filepath.Join(t.TempDir(), "final.overlay.json")
	if err := os.WriteFile(finalOverlayPath, []byte(`{"id":"pdo-dynamic-final"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, digest, err := LoadConfigWithOverlays(
		testBaseConfigPath(),
		[]string{overlayPath, finalOverlayPath},
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != "pdo-dynamic-final" || cfg.Naming.ModulePrefix != "PDO" {
		t.Fatalf("overlay values were not applied: %+v", cfg)
	}
	if cfg.Docs.RequireFileMetadata || !cfg.Docs.AllFunctions {
		t.Fatalf("nested overlay did not preserve omitted fields: %+v", cfg.Docs)
	}
	if len(cfg.Scope.Exclude) != 1 || cfg.Scope.Exclude[0] != "vendor" {
		t.Fatalf("overlay array did not replace the base value: %v", cfg.Scope.Exclude)
	}
	if len(digest) != 64 {
		t.Fatalf("unexpected effective config digest %q", digest)
	}
}

func TestConfigOverlayRejectsUnsafeShapes(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "locked", content: `{"standard":"c11"}`, want: "locked"},
		{name: "null", content: `{"documentation":{"allFunctions":null}}`, want: "null"},
		{name: "unknown", content: `{"documentation":{"unknownRule":true}}`, want: "unknown field"},
		{name: "duplicate", content: `{"id":"one","id":"two"}`, want: "duplicate JSON key"},
		{name: "invalid-policy", content: `{"documentation":{"employeeIdPolicy":"never"}}`, want: "employeeIdPolicy"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "overlay.json")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, _, err := LoadConfigWithOverlays(testBaseConfigPath(), []string{path})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestConfigOverlayRejectsUnsafeGateChanges(t *testing.T) {
	base, err := LoadConfig(testBaseConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		override map[string]any
		want     string
	}{
		{
			name:     "empty-flags",
			override: map[string]any{"flags": []string{}},
			want:     "missing required flag",
		},
		{
			name:     "warnings-not-errors",
			override: map[string]any{"warningsAsErrors": false},
			want:     "warningsAsErrors",
		},
	}
	for _, unsafeFlag := range []string{
		"-fplugin=evil.dll",
		"-specs=evil.specs",
		"-wrapper",
		"@evil.rsp",
		"-O2",
	} {
		flag := unsafeFlag
		tests = append(tests, struct {
			name     string
			override map[string]any
			want     string
		}{
			name: "reject-" + strings.TrimLeft(flag, "-@"),
			override: map[string]any{
				"flags": append(append([]string{}, base.Gates.GCC.Flags...), flag),
			},
			want: "not allowed",
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, marshalErr := json.Marshal(map[string]any{
				"gates": map[string]any{"gcc": test.override},
			})
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			path := filepath.Join(t.TempDir(), "unsafe-gate.overlay.json")
			if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
				t.Fatal(writeErr)
			}
			_, _, loadErr := LoadConfigWithOverlays(testBaseConfigPath(), []string{path})
			if loadErr == nil || !strings.Contains(loadErr.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, loadErr)
			}
		})
	}
}

func TestExecuteVerificationFastUsesOnlyGCC(t *testing.T) {
	targetPath, overlayPath := makeVerifyFixture(t, false, true)
	runner := &fakeVerifyRunner{
		candidateOutput: []byte("same-output\n"),
		baselineOutput:  []byte("same-output\n"),
	}
	result, err := executeVerification(context.Background(), verifyOptions{
		ConfigPath:   testBaseConfigPath(),
		OverlayPaths: []string{overlayPath},
		TargetPath:   targetPath,
		Profile:      "fast",
		Timings:      true,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("fast verification failed: %+v", result)
	}
	wantSteps := "scope-hash,lint,readability,gcc-c99,gcc-header-c99,candidate-host-test"
	if got := verifyStepIDs(result); got != wantSteps {
		t.Fatalf("unexpected fast steps %q", got)
	}
	for _, step := range result.Steps {
		if step.DurationMS == nil {
			t.Fatalf("--timings omitted duration for step %+v", step)
		}
	}
	for _, command := range runner.commands {
		base := filepath.Base(command.executable)
		if strings.Contains(base, "clang") || base == "g++" || base == "g++.exe" {
			t.Fatalf("fast profile invoked a full-only tool: %+v", command)
		}
		if !strings.HasPrefix(filepath.Base(command.cwd), "cstylekit-verify-") {
			t.Fatalf("command escaped the verification temp cwd: %+v", command)
		}
		for _, argument := range command.args {
			if strings.Contains(argument, filepath.Dir(targetPath)) {
				t.Fatalf("command read an original manifest path instead of a snapshot: %+v", command)
			}
		}
	}
}

func TestExecuteVerificationFullDetectsBehaviorDifference(t *testing.T) {
	targetPath, overlayPath := makeVerifyFixture(t, false, true)
	runner := &fakeVerifyRunner{
		candidateOutput: []byte("candidate\n"),
		baselineOutput:  []byte("baseline\n"),
	}
	result, err := executeVerification(context.Background(), verifyOptions{
		ConfigPath:   testBaseConfigPath(),
		OverlayPaths: []string{overlayPath},
		TargetPath:   targetPath,
		Profile:      "full",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("full verification accepted different baseline and candidate output")
	}
	last := result.Steps[len(result.Steps)-1]
	if last.ID != "behavior-equivalence" || last.Status != "fail" {
		t.Fatalf("unexpected final step: %+v", last)
	}
}

func TestExecuteVerificationRequiresHostHarness(t *testing.T) {
	targetPath, overlayPath := makeVerifyFixture(t, false, false)
	runner := &fakeVerifyRunner{}
	result, err := executeVerification(context.Background(), verifyOptions{
		ConfigPath:   testBaseConfigPath(),
		OverlayPaths: []string{overlayPath},
		TargetPath:   targetPath,
		Profile:      "fast",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("verification accepted a target without host.testSource")
	}
	last := result.Steps[len(result.Steps)-1]
	if last.ID != "candidate-host-test" || !strings.Contains(last.Message, "testSource") {
		t.Fatalf("unexpected missing harness result: %+v", last)
	}
}

func TestExecuteVerificationRejectsExplicitExcludedFile(t *testing.T) {
	targetPath, overlayPath := makeVerifyFixture(t, true, true)
	result, err := executeVerification(context.Background(), verifyOptions{
		ConfigPath:   testBaseConfigPath(),
		OverlayPaths: []string{overlayPath},
		TargetPath:   targetPath,
		Profile:      "fast",
	}, &fakeVerifyRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || len(result.Steps) != 1 || result.Steps[0].ID != "scope-hash" {
		t.Fatalf("excluded target did not fail at scope check: %+v", result)
	}
	if !strings.Contains(result.Steps[0].Message, "excluded") {
		t.Fatalf("unexpected excluded target message: %+v", result.Steps[0])
	}
}

func TestCollectVerifyFilesUsesImmutableSnapshots(t *testing.T) {
	targetPath, overlayPath := makeVerifyFixture(t, false, true)
	cfg, _, err := LoadConfigWithOverlays(testBaseConfigPath(), []string{overlayPath})
	if err != nil {
		t.Fatal(err)
	}
	target, err := loadVerifyTarget(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	reports, dataByPath, snapshot, err := collectVerifyFiles(target, cfg, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	originalBytes := append([]byte{}, dataByPath[target.Candidate.Source]...)
	if err := os.WriteFile(target.Candidate.Source, []byte("mutated after scope-hash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshotBytes, err := os.ReadFile(snapshot.Candidate.Source)
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshotBytes) != string(originalBytes) {
		t.Fatal("source snapshot changed after the original file was modified")
	}
	if !strings.HasPrefix(snapshot.Candidate.Source, tempDir) {
		t.Fatalf("snapshot escaped temp directory: %s", snapshot.Candidate.Source)
	}
	if len(reports) == 0 || reports[0].Path != filepath.ToSlash(target.Candidate.Source) {
		t.Fatalf("report did not retain the original path: %+v", reports)
	}
}

func TestSanitizedVerifyEnvironmentRemovesCompilerInjection(t *testing.T) {
	actual := sanitizedVerifyEnvironment([]string{
		"PATH=C:\\tools",
		"CPATH=C:\\untrusted-headers",
		"GCC_EXEC_PREFIX=C:\\untrusted-tools",
		"LD_PRELOAD=/tmp/untrusted.so",
		"TEMP=C:\\temp",
	})
	joined := strings.Join(actual, "\n")
	if !strings.Contains(joined, "PATH=C:\\tools") || !strings.Contains(joined, "TEMP=C:\\temp") {
		t.Fatalf("required environment was removed: %v", actual)
	}
	for _, blocked := range []string{"CPATH=", "GCC_EXEC_PREFIX=", "LD_PRELOAD="} {
		if strings.Contains(joined, blocked) {
			t.Fatalf("compiler injection environment survived: %v", actual)
		}
	}
}

type fakeVerifyCommand struct {
	executable string
	cwd        string
	args       []string
}

type fakeVerifyRunner struct {
	commands        []fakeVerifyCommand
	candidateOutput []byte
	baselineOutput  []byte
}

func (runner *fakeVerifyRunner) LookPath(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty tool")
	}
	return name, nil
}

func (runner *fakeVerifyRunner) Run(
	_ context.Context,
	cwd string,
	executable string,
	args ...string,
) ([]byte, error) {
	runner.commands = append(runner.commands, fakeVerifyCommand{
		executable: executable,
		cwd:        cwd,
		args:       append([]string{}, args...),
	})
	if len(args) == 1 && args[0] == "--version" {
		return []byte("clang version test\nTarget: x86_64-w64-windows-gnu\n"), nil
	}
	base := strings.ToLower(filepath.Base(executable))
	if strings.Contains(base, "candidate-host") && len(args) == 0 {
		return runner.candidateOutput, nil
	}
	if strings.Contains(base, "baseline-host") && len(args) == 0 {
		return runner.baselineOutput, nil
	}
	return nil, nil
}

func makeVerifyFixture(t *testing.T, excluded, includeHarness bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	moduleDir := root
	if excluded {
		moduleDir = filepath.Join(root, "build")
		if err := os.MkdirAll(moduleDir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	copyTestFile(t, filepath.Join("..", "..", "generated-demo", "demo.c"), filepath.Join(moduleDir, "demo.c"))
	copyTestFile(t, filepath.Join("..", "..", "generated-demo", "demo.h"), filepath.Join(moduleDir, "demo.h"))
	harnessPath := filepath.Join(root, "demo_test.c")
	if includeHarness {
		copyTestFile(
			t,
			filepath.Join("..", "..", "internal", "cuserstyle", "templates", "tests", "demo_test.c"),
			harnessPath,
		)
	}

	target := map[string]any{
		"schema": verifyTargetSchema,
		"id":     "verify-fixture",
		"candidate": map[string]any{
			"source": relativeTestPath(t, root, filepath.Join(moduleDir, "demo.c")),
			"header": relativeTestPath(t, root, filepath.Join(moduleDir, "demo.h")),
		},
		"baseline": map[string]any{
			"source": relativeTestPath(t, root, filepath.Join(moduleDir, "demo.c")),
			"header": relativeTestPath(t, root, filepath.Join(moduleDir, "demo.h")),
		},
	}
	if includeHarness {
		target["host"] = map[string]any{
			"testSource": relativeTestPath(t, root, harnessPath),
		}
	}
	data, err := json.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(root, "target.json")
	if err := os.WriteFile(targetPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(root, "verify-test.overlay.json")
	overlay := `{
  "documentation": {
    "fileHeader": false,
    "allFunctions": false,
    "requireCaseIntentComment": false,
    "requireGlobalVariableDetail": false,
    "requireExternC": false
  },
  "macros": {"requireDocumentation": false},
  "readability": {
    "complexFunction": {"requireNumberedFlow": false},
    "requireNumberedIntentCommentPlacement": false
  }
}`
	if err := os.WriteFile(overlayPath, []byte(overlay), 0o600); err != nil {
		t.Fatal(err)
	}
	return targetPath, overlayPath
}

func copyTestFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func relativeTestPath(t *testing.T, base, target string) string {
	t.Helper()
	relative, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(relative)
}

func testBaseConfigPath() string {
	return filepath.Join("..", "..", "examples", "c-kit.json")
}

func verifyStepIDs(result VerifyResult) string {
	ids := make([]string, 0, len(result.Steps))
	for _, step := range result.Steps {
		ids = append(ids, step.ID)
	}
	return strings.Join(ids, ",")
}
