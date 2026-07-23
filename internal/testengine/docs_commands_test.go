package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureDiagramsAcceptTypedCatalogCommands(t *testing.T) {
	t.Parallel()
	repo := architectureDiagramFixture(t)
	if err := checkArchitectureDiagrams(repo); err != nil {
		t.Fatal(err)
	}
}

func TestArchitectureDiagramsRejectUnknownCommandWithLocation(t *testing.T) {
	t.Parallel()
	repo := architectureDiagramFixture(t)
	writeArchitectureFixtureFile(t, repo, "docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md", "```mermaid\nflowchart LR\n  BAD[\"aicoding nonexistent\"]\n```\n\n```mermaid\nflowchart LR\n  OK[\"aicoding verify\"]\n```\n")

	err := checkArchitectureDiagrams(repo)
	if err == nil {
		t.Fatal("unknown diagram command must fail")
	}
	for _, want := range []string{"LOOP_ENGINEERING_ARCHITECTURE.md:3", "aicoding nonexistent", "typed catalog"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
}

func TestArchitectureDiagramsEnforceNodeBudget(t *testing.T) {
	t.Parallel()
	repo := architectureDiagramFixture(t)
	var source strings.Builder
	source.WriteString("```mermaid\nflowchart LR\n")
	for index := 0; index < architectureDiagramNodeBudget+1; index++ {
		source.WriteString("  N")
		source.WriteString(strings.Repeat("X", index))
		source.WriteString("[\"node\"]\n")
	}
	source.WriteString("```\n")
	writeArchitectureFixtureFile(t, repo, "docs/architecture/PRIMITIVE_CONSTITUTION.md", source.String())

	err := checkArchitectureDiagrams(repo)
	if err == nil || !strings.Contains(err.Error(), "21 explicit nodes") {
		t.Fatalf("node budget error = %v", err)
	}
}

func TestReadmeArchitectureDiagramRejectsMissingThemeVariant(t *testing.T) {
	t.Parallel()
	repo := architectureDiagramFixture(t)
	writeArchitectureFixtureFile(t, repo, "README.md", `<img src="docs/assets/aicoding-overview-light.svg#gh-light-mode-only">`)

	err := checkArchitectureDiagrams(repo)
	if err == nil || !strings.Contains(err.Error(), "aicoding-overview-dark.svg#gh-dark-mode-only") {
		t.Fatalf("missing dark SVG variant error = %v", err)
	}
}

func TestRepositoryArchitectureDiagrams(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkArchitectureDiagrams(repo); err != nil {
		t.Fatal(err)
	}
}

func architectureDiagramFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeArchitectureFixtureFile(t, repo, "internal/cli/catalog.go", `package cli

var commands = []CommandDescriptor{
	CommandDescriptor{Name: "verify", Aliases: []string{"check"}},
}

type CommandDescriptor struct {
	Name string
	Aliases []string
}
`)
	for _, document := range mermaidArchitectureDiagramDocuments {
		content := strings.Repeat("```mermaid\nflowchart LR\n  OK[\"aicoding verify\"]\n```\n", document.count)
		writeArchitectureFixtureFile(t, repo, document.path, content)
	}
	writeArchitectureFixtureFile(t, repo, "README.md", `<img src="docs/assets/aicoding-overview-light.svg#gh-light-mode-only">
<img src="docs/assets/aicoding-overview-dark.svg#gh-dark-mode-only">
`)
	for _, variant := range readmeSVGArchitectureVariants {
		writeArchitectureFixtureFile(t, repo, variant.path, `<svg><title>VMCP_entry</title><text>aicoding verify</text></svg>`)
	}
	return repo
}

func writeArchitectureFixtureFile(t *testing.T, repo string, rel string, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
