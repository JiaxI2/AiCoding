package cstyle

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSkillConfig(t *testing.T, root string, style string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "config", "skills", "c99-standard-c", "skill.json"), `{
  "schemaVersion": 1,
  "id": "c99-standard-c",
  "title": "C99 Standard C Skill",
  "language": "c",
  "standard": "c99",
  "formatter": { "id": "clang-format", "config": "style/clang-format.yaml" },
  "commentTemplates": "templates/comment-templates.json",
  "rules": "rules/embedded-c-rules.md",
  "excludedDirectories": ["vendor", "third_party", "generated", "Drivers", "device", "build", "out", "dist"]
}
`)
	writeFile(t, filepath.Join(root, "config", "skills", "c99-standard-c", "style", "clang-format.yaml"), style)
	writeFile(t, filepath.Join(root, "config", "skills", "c99-standard-c", "templates", "comment-templates.json"), `{
  "schemaVersion": 1,
  "templates": [
    {
      "id": "c-file-header-cn",
      "title": "C File Header (CN)",
      "description": "中文 C 文件头注释模板。",
      "language": "c",
      "kind": "file-header",
      "body": ["/**", " * @brief {{brief}}", " */"],
      "variables": { "author": { "description": "作者。", "default": "HU JIAXUAN" } }
    }
  ]
}
`)
	writeFile(t, filepath.Join(root, "config", "skills", "c99-standard-c", "rules", "embedded-c-rules.md"), "# rules\n")
}

func writeSkillKitFixture(t *testing.T, root string) {
	t.Helper()
	kitRoot := filepath.Join(root, filepath.FromSlash(DefaultKitRoot))
	writeFile(t, filepath.Join(kitRoot, "go.mod"), "module c-userstyle-kit\n\ngo 1.22\n")
	writeFile(t, filepath.Join(kitRoot, filepath.FromSlash(DefaultKitConfig)), "{}\n")
	writeFile(t, filepath.Join(kitRoot, filepath.FromSlash(DefaultKitSnippets)), "{}\n")
	writeFile(t, filepath.Join(kitRoot, filepath.FromSlash(DefaultKitQuickTarget)), "{}\n")
}

type fakeVerifyRunner struct {
	stdout             []byte
	stderr             []byte
	err                error
	repoRoot           string
	kitRoot            string
	args               []string
	contextHasDeadline bool
}

func (r *fakeVerifyRunner) Run(
	ctx context.Context,
	repoRoot string,
	kitRoot string,
	args []string,
) ([]byte, []byte, error) {
	r.repoRoot = repoRoot
	r.kitRoot = kitRoot
	r.args = append([]string{}, args...)
	_, r.contextHasDeadline = ctx.Deadline()
	return r.stdout, r.stderr, r.err
}

func TestCollectFilesAllExcludesVendorAndGenerated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "foc.c"), "int main(void){return 0;}\n")
	writeFile(t, filepath.Join(root, "include", "foc.h"), "int foc(void);\n")
	writeFile(t, filepath.Join(root, "vendor", "x.c"), "int vendor(void);\n")
	writeFile(t, filepath.Join(root, "generated", "x.h"), "int generated(void);\n")
	writeFile(t, filepath.Join(root, "Drivers", "x.c"), "int driver(void);\n")
	writeFile(t, filepath.Join(root, "device", "x.h"), "int device(void);\n")

	files, err := CollectFiles(root, ScopeAll, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, f := range files {
		got[f] = true
	}

	if !got["include/foc.h"] || !got["src/foc.c"] {
		t.Fatalf("expected src/include files, got %#v", files)
	}
	if got["vendor/x.c"] || got["generated/x.h"] || got["Drivers/x.c"] || got["device/x.h"] {
		t.Fatalf("excluded files must be ignored, got %#v", files)
	}
}

func TestCollectFilesAllSupportsRepoRelativeExclusion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CodingKit", "tools", "c-userstyle-kit", "golden", "demo.c"), "int demo(void);\n")
	writeFile(t, filepath.Join(root, "CodingKit", "tools", "c-userstyle-kit-old", "keep.c"), "int keep_similar(void);\n")
	writeFile(t, filepath.Join(root, "CodingKit", "tools", "other-kit", "keep.c"), "int keep_tool(void);\n")
	writeFile(t, filepath.Join(root, "CodingKit", "modules", "keep.h"), "int keep_module(void);\n")
	writeFile(t, filepath.Join(root, "tools", "keep.c"), "int keep_root_tool(void);\n")

	files, err := CollectFilesWithExclusions(
		root,
		ScopeAll,
		nil,
		[]string{"CodingKit/tools/c-userstyle-kit"},
	)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, file := range files {
		got[file] = true
	}
	if got["CodingKit/tools/c-userstyle-kit/golden/demo.c"] {
		t.Fatalf("repo-relative exclusion must omit only the configured subtree, got %#v", files)
	}
	for _, want := range []string{
		"CodingKit/modules/keep.h",
		"CodingKit/tools/c-userstyle-kit-old/keep.c",
		"CodingKit/tools/other-kit/keep.c",
		"tools/keep.c",
	} {
		if !got[want] {
			t.Fatalf("repo-relative exclusion unexpectedly omitted %s: %#v", want, files)
		}
	}
}

func TestRepoRelativeExclusionUsesPlatformPathSemantics(t *testing.T) {
	excluded := excludedSet([]string{"CodingKit/tools/c-userstyle-kit", "vendor"})
	if !isExcluded("CodingKit/tools/c-userstyle-kit/generated-demo/demo.c", excluded) {
		t.Fatal("configured repo-relative subtree must be excluded")
	}
	if isExcluded("CodingKit/tools/c-userstyle-kit-old/demo.c", excluded) {
		t.Fatal("repo-relative exclusion must stop at a path boundary")
	}

	if runtime.GOOS != "windows" {
		return
	}
	if !isExcluded(`codingkit\TOOLS\C-USERSTYLE-KIT\generated-demo\demo.c`, excluded) {
		t.Fatal("Windows path matching must ignore case and accept backslashes")
	}
	if !isExcluded(`src\VENDOR\driver.c`, excluded) {
		t.Fatal("Windows directory-name matching must ignore case")
	}

	root := t.TempDir()
	target := filepath.Join(root, "CodingKit", "tools", "c-userstyle-kit", "generated-demo", "demo.c")
	writeFile(t, target, "int demo(void);\n")
	files, err := CollectFilesWithExclusions(
		root,
		ScopePaths,
		[]string{strings.ToLower(target)},
		[]string{"CodingKit/tools/c-userstyle-kit"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("case-variant absolute Windows path bypassed exclusion: %#v", files)
	}
}

func TestRunCheckDetectsDriftThenFormatFixes(t *testing.T) {
	if _, err := exec.LookPath("clang-format"); err != nil {
		t.Skip("clang-format not available")
	}

	root := t.TempDir()
	writeSkillConfig(t, root, "BasedOnStyle: LLVM\nIndentWidth: 4\nBreakBeforeBraces: Allman\n")
	writeFile(t, filepath.Join(root, "src", "foc.c"), "int foc(int x){if(x){return 1;}return 0;}\n")

	_, err := Run(Options{RepoRoot: root, Scope: ScopePaths, Paths: []string{"src/foc.c"}, Check: true})
	if err == nil {
		t.Fatalf("expected check to detect formatting drift")
	}

	if _, err := Run(Options{RepoRoot: root, Scope: ScopePaths, Paths: []string{"src/foc.c"}}); err != nil {
		t.Fatalf("format failed: %v", err)
	}

	if _, err := Run(Options{RepoRoot: root, Scope: ScopePaths, Paths: []string{"src/foc.c"}, Check: true}); err != nil {
		t.Fatalf("check after format failed: %v", err)
	}
}

func TestLoadSkillConfig(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadSkillConfig(repoRoot, DefaultSkillID)
	if err != nil {
		t.Fatalf("load skill config: %v", err)
	}
	if cfg.ID != DefaultSkillID || cfg.Language != "c" || cfg.Standard != "c99" {
		t.Fatalf("unexpected skill config: %#v", cfg)
	}

	formatterPath, err := ResolveFormatterConfig(repoRoot, cfg)
	if err != nil {
		t.Fatalf("resolve formatter config: %v", err)
	}
	if _, err := os.Stat(formatterPath); err != nil {
		t.Fatalf("formatter config missing: %v", err)
	}
}

func TestValidateTemplatesConfig(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	validation, err := ValidateTemplates(repoRoot)
	if err != nil {
		t.Fatalf("template validation failed: %v; errors=%v", err, validation.Errors)
	}
	if !validation.Valid {
		t.Fatalf("template validation should be valid: %#v", validation)
	}
	if validation.Path != CommentTemplatesPath {
		t.Fatalf("unexpected template path: %s", validation.Path)
	}

	wantIDs := map[string]bool{
		"c-file-header-professional": false,
		"c-file-header-cn":           false,
		"c-function-header-full":     false,
		"c-function-header-void":     false,
		"c-section-divider":          false,
		"c-struct-brief":             false,
		"c-enum-brief":               false,
		"c-common-includes":          false,
	}
	for _, tmpl := range validation.Templates {
		if _, ok := wantIDs[tmpl.ID]; ok {
			wantIDs[tmpl.ID] = true
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Fatalf("missing template id %s in %#v", id, validation.Templates)
		}
	}
}

func TestValidateTemplatesRejectsVersionedFileHeader(t *testing.T) {
	root := t.TempDir()
	writeSkillConfig(t, root, "BasedOnStyle: LLVM\n")
	writeFile(t, filepath.Join(root, "config", "skills", "c99-standard-c", "templates", "comment-templates.json"), `{
  "schemaVersion": 1,
  "templates": [
    {
      "id": "versioned-file-header",
      "title": "Versioned File Header",
      "description": "invalid fixture",
      "language": "c",
      "kind": "file-header",
      "body": ["/**", " * @version {{version}}", " */"],
      "variables": {
        "version": { "description": "source version", "default": "1.0.0" }
      }
    }
  ]
}
`)

	validation, err := ValidateTemplates(root)
	if err == nil || validation.Valid {
		t.Fatalf("versioned file header must fail validation: validation=%#v err=%v", validation, err)
	}

	joined := strings.Join(validation.Errors, "\n")
	for _, want := range []string{
		"file header must not expose a source version",
		"file header must not declare a version variable",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing validation error %q in %q", want, joined)
		}
	}
}

func TestSkillStatusReportsRequiredKitAssets(t *testing.T) {
	root := t.TempDir()
	writeSkillConfig(t, root, "BasedOnStyle: LLVM\n")
	writeSkillKitFixture(t, root)

	status, err := SkillStatus(root, DefaultSkillID)
	if err != nil {
		t.Fatalf("skill status failed: %v", err)
	}
	if status.KitID != DefaultKitID || !status.KitRootExists || !status.KitConfigExists ||
		!status.KitSnippetsExists || !status.KitQuickTargetExists {
		t.Fatalf("unexpected kit status: %#v", status)
	}

	quickTarget := filepath.Join(root, filepath.FromSlash(DefaultKitRoot), filepath.FromSlash(DefaultKitQuickTarget))
	if err := os.Remove(quickTarget); err != nil {
		t.Fatal(err)
	}
	status, err = SkillStatus(root, DefaultSkillID)
	if err == nil || status.KitQuickTargetExists || !strings.Contains(err.Error(), "kit quick target not found") {
		t.Fatalf("missing quick target must fail status: status=%#v err=%v", status, err)
	}
}

func TestVerifyBySkillResolvesPathsAndParsesSingleJSON(t *testing.T) {
	root := t.TempDir()
	writeSkillConfig(t, root, "BasedOnStyle: LLVM\n")
	writeSkillKitFixture(t, root)
	writeFile(t, filepath.Join(root, "fixtures", "target.json"), "{}\n")
	writeFile(t, filepath.Join(root, "overlays", "project.json"), "{}\n")

	runner := &fakeVerifyRunner{stdout: []byte(`{"schema":"cstylekit.verify.v1","ok":true,"profile":"full"}`)}
	result, err := verifyBySkill(DefaultSkillID, VerifyOptions{
		RepoRoot: root,
		Profile:  "full",
		Target:   "fixtures/target.json",
		Overlays: []string{"overlays/project.json"},
		Timings:  true,
	}, runner)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.Payload["ok"] != true || result.Profile != "full" || !runner.contextHasDeadline {
		t.Fatalf("unexpected verify result: result=%#v runner=%#v", result, runner)
	}
	if runner.kitRoot != filepath.Join(root, filepath.FromSlash(DefaultKitRoot)) {
		t.Fatalf("unexpected kit root: %s", runner.kitRoot)
	}
	wantArgs := []string{
		"verify",
		"--config", filepath.Join(root, filepath.FromSlash(DefaultKitRoot), filepath.FromSlash(DefaultKitConfig)),
		"--target", filepath.Join(root, "fixtures", "target.json"),
		"--profile", "full",
		"--json",
		"--overlay", filepath.Join(root, "overlays", "project.json"),
		"--timings",
	}
	if strings.Join(runner.args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("unexpected C Kit arguments:\nwant %#v\n got %#v", wantArgs, runner.args)
	}
}

func TestVerifyBySkillRejectsInvalidOrFailedJSON(t *testing.T) {
	root := t.TempDir()
	writeSkillConfig(t, root, "BasedOnStyle: LLVM\n")
	writeSkillKitFixture(t, root)

	for _, tc := range []struct {
		name   string
		stdout string
		runErr error
		want   string
	}{
		{name: "invalid", stdout: "not-json", want: "invalid C Kit verify JSON"},
		{name: "multiple", stdout: `{"ok":true} {"ok":true}`, want: "multiple JSON values"},
		{name: "wrong schema", stdout: `{"schema":"other","profile":"fast","ok":true}`, want: "schema must be"},
		{name: "wrong profile", stdout: `{"schema":"cstylekit.verify.v1","profile":"full","ok":true}`, want: "profile must match"},
		{name: "failed result", stdout: `{"schema":"cstylekit.verify.v1","profile":"fast","ok":false}`, runErr: errors.New("exit status 1"), want: "ok=false"},
		{name: "failed process", stdout: `{"schema":"cstylekit.verify.v1","profile":"fast","ok":true}`, runErr: errors.New("exit status 1"), want: "process failed"},
		{name: "missing ok", stdout: `{"schema":"cstylekit.verify.v1","profile":"fast"}`, want: "field ok must be boolean"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeVerifyRunner{stdout: []byte(tc.stdout), err: tc.runErr}
			_, err := verifyBySkill(DefaultSkillID, VerifyOptions{RepoRoot: root, Profile: "fast"}, runner)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted error containing %q, got %v", tc.want, err)
			}
		})
	}
}
