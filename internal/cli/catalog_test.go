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
		if command.LatencyClass == "" {
			t.Fatalf("command %s has no latency class", command.ID)
		}
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
		"aicoding lifecycle plan",
		"aicoding validation check --profile Smoke|Full|Release --target HEAD|INDEX [--bind-alias]",
		"aicoding validation explain --profile Smoke|Full|Release --target HEAD|INDEX",
		"aicoding plan check (--staged | --paths PATH ...)",
		"aicoding plan approve --id ID",
		"aicoding doctor pwsh [--repo-root PATH] [--json]",
		"aicoding powershell regex-lint --path PATH",
	} {
		if !strings.Contains(help.String(), expected) {
			t.Fatalf("catalog help missing %q: %s", expected, help.String())
		}
	}
	for _, removed := range []string{
		"Compatibility commands",
		"CLI_DEPRECATED",
		"aicoding smoke",
		"aicoding test full|release",
		"aicoding status --all",
	} {
		if strings.Contains(help.String(), removed) {
			t.Fatalf("catalog help still exposes removed compatibility form %q: %s", removed, help.String())
		}
	}
	validationForms := 0
	workForms := 0
	planForms := 0
	for _, form := range catalog.Help {
		if form.Command == CommandValidation {
			validationForms++
		}
		if form.Command == CommandWork {
			workForms++
		}
		if form.Command == CommandPlan {
			planForms++
		}
	}
	if validationForms != 5 {
		t.Fatalf("validation help form count = %d, want 5", validationForms)
	}
	if workForms != 4 {
		t.Fatalf("work help form count = %d, want 4", workForms)
	}
	if planForms != 4 {
		t.Fatalf("plan help form count = %d, want 4", planForms)
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
		[]commandRoute{{descriptor: CommandDescriptor{ID: "broken", Name: "broken", LatencyClass: LatencyWork}}},
		[]HelpSection{{ID: HelpUsage, Title: "Usage:"}},
		[]HelpForm{{Command: "broken", Section: HelpUsage, Usage: "aicoding broken"}},
	)
	if err == nil {
		t.Fatal("catalog accepted a command without a route")
	}
}

func TestCommandCatalogRejectsMissingLatencyClass(t *testing.T) {
	_, err := newCommandCatalog(
		[]commandRoute{{descriptor: CommandDescriptor{ID: "broken", Name: "broken"}, handler: runBootstrap}},
		[]HelpSection{{ID: HelpUsage, Title: "Usage:"}},
		[]HelpForm{{Command: "broken", Section: HelpUsage, Usage: "aicoding broken"}},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid latency class") {
		t.Fatalf("catalog accepted a command without LatencyClass: %v", err)
	}
}

func TestCommandCatalogRejectsGitPorcelainVerbs(t *testing.T) {
	// docs/architecture/GIT_REUSE_BOUNDARY.md §9 reserves Git porcelain verbs.
	// Section 8 explicitly retains tag and fresh-clone because their AiCoding
	// meanings are policy audit and a registered verification workflow rather
	// than Git porcelain aliases.
	forbidden := map[string]struct{}{
		"add": {}, "am": {}, "apply": {}, "bisect": {}, "blame": {},
		"branch": {}, "checkout": {}, "cherry-pick": {}, "clone": {},
		"commit": {}, "diff": {}, "fetch": {}, "init": {}, "log": {},
		"merge": {}, "pull": {}, "push": {}, "rebase": {}, "remote": {},
		"reset": {}, "restore": {}, "revert": {}, "show": {}, "stash": {},
		"submodule": {}, "switch": {}, "worktree": {},
	}
	for _, command := range Catalog().Commands {
		for _, name := range append([]string{command.Name}, command.Aliases...) {
			if _, exists := forbidden[strings.ToLower(name)]; exists {
				t.Fatalf("command catalog exposes reserved Git porcelain verb %q through %s", name, command.ID)
			}
		}
	}
}
