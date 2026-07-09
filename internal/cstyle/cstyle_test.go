package cstyle

import (
	"os"
	"os/exec"
	"path/filepath"
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
