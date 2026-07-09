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

func TestCollectFilesAllExcludesVendorAndGenerated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".clang-format"), "BasedOnStyle: LLVM\n")
	writeFile(t, filepath.Join(root, "src", "foc.c"), "int main(void){return 0;}\n")
	writeFile(t, filepath.Join(root, "include", "foc.h"), "int foc(void);\n")
	writeFile(t, filepath.Join(root, "vendor", "x.c"), "int vendor(void);\n")
	writeFile(t, filepath.Join(root, "generated", "x.h"), "int generated(void);\n")

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
	if got["vendor/x.c"] || got["generated/x.h"] {
		t.Fatalf("vendor/generated files must be excluded, got %#v", files)
	}
}

func TestRunCheckDetectsDriftThenFormatFixes(t *testing.T) {
	if _, err := exec.LookPath("clang-format"); err != nil {
		t.Skip("clang-format not available")
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".clang-format"), "BasedOnStyle: LLVM\nIndentWidth: 4\nBreakBeforeBraces: Allman\n")
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
