package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandCatalogOwnsRoutesHelpAndNamespaceContracts(t *testing.T) {
	catalog := Catalog()
	if len(catalog.Commands) == 0 || len(catalog.Help) == 0 {
		t.Fatalf("empty command catalog: %#v", catalog)
	}
	seen := map[CommandID]bool{}
	for _, command := range catalog.Commands {
		if seen[command.ID] {
			t.Fatalf("duplicate command id: %s", command.ID)
		}
		seen[command.ID] = true
		if commandRequiresSubcommand(command.Name) != command.RequiresSubcommand {
			t.Fatalf("namespace contract drift for %s", command.ID)
		}
	}
	for _, form := range catalog.Help {
		if !seen[form.Command] {
			t.Fatalf("help references unknown command: %#v", form)
		}
	}

	var help bytes.Buffer
	writeCatalogHelp(&help)
	for _, expected := range []string{
		"Formal product workflow:",
		"Compatibility commands (emit CLI_DEPRECATED):",
		"aicoding lifecycle plan",
		"aicoding powershell regex-lint --path PATH",
	} {
		if !strings.Contains(help.String(), expected) {
			t.Fatalf("catalog help missing %q: %s", expected, help.String())
		}
	}
}

func TestCommandCatalogSnapshotIsStableAndDetached(t *testing.T) {
	digest := CatalogSnapshot().Digest()
	if !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("catalog digest is missing: %q", digest)
	}
	catalog := Catalog()
	catalog.Commands[0].Name = "mutated"
	if Catalog().Commands[0].Name == "mutated" || CatalogSnapshot().Digest() != digest {
		t.Fatal("catalog was mutable through its descriptor")
	}
}

func TestCommandCatalogRejectsIncompleteRoutes(t *testing.T) {
	_, err := newCommandCatalog(
		[]commandRoute{{descriptor: CommandDescriptor{ID: "broken", Name: "broken"}}},
		[]HelpSection{{ID: HelpUsage, Title: "Usage:"}},
		[]HelpForm{{Command: "broken", Section: HelpUsage, Usage: "aicoding broken"}},
	)
	if err == nil {
		t.Fatal("catalog accepted a command without a route")
	}
}
