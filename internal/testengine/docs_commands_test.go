package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureDiagramsAcceptTypedCatalogCommands(t *testing.T) {
	repo := architectureDiagramFixture(t)
	if err := checkArchitectureDiagrams(repo); err != nil {
		t.Fatal(err)
	}
}

func TestArchitectureDiagramsRejectUnknownCommandWithLocation(t *testing.T) {
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
	repo := architectureDiagramFixture(t)
	var source strings.Builder
	source.WriteString("```mermaid\nflowchart LR\n")
	for index := 0; index < architectureDiagramNodeBudget+1; index++ {
		source.WriteString("  N")
		source.WriteString(strings.Repeat("X", index))
		source.WriteString("[\"node\"]\n")
	}
	source.WriteString("```\n")
	writeArchitectureFixtureFile(t, repo, "README.md", source.String())

	err := checkArchitectureDiagrams(repo)
	if err == nil || !strings.Contains(err.Error(), "21 explicit nodes") {
		t.Fatalf("node budget error = %v", err)
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
	for _, document := range architectureDiagramDocuments {
		content := strings.Repeat("```mermaid\nflowchart LR\n  OK[\"aicoding verify\"]\n```\n", document.count)
		writeArchitectureFixtureFile(t, repo, document.path, content)
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
