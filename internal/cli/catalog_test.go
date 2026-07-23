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

func TestCommandCatalogProjectsSubcommandAliasesHelpAndQuickstarts(t *testing.T) {
	const (
		commandID CommandID    = "demo"
		runID     SubcommandID = "demo.run"
	)
	catalog, err := newCommandCatalog(
		[]commandRoute{{descriptor: CommandDescriptor{
			ID: commandID, Name: "demo", Aliases: []string{"d"}, RequiresSubcommand: true, LatencyClass: LatencyFast,
			Subcommands: []SubcommandDescriptor{{ID: runID, Name: "run", Aliases: []string{"quick"}}},
		}, handler: runBootstrap}},
		[]HelpSection{{ID: HelpUsage, Title: "Usage:"}},
		[]HelpForm{help(commandID, HelpUsage, "[--json]", runID)},
		[]QuickstartForm{{Operation: "demo", Command: commandID, Path: []SubcommandID{runID}, Args: []string{"--json"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	route, ok := catalog.lookup("d")
	if !ok {
		t.Fatal("top-level alias was not projected")
	}
	normalized, err := catalog.prepareInvocation(route, []string{"quick", "--json"})
	if err != nil || len(normalized) != 2 || normalized[0] != "run" {
		t.Fatalf("subcommand alias projection = %#v, %v", normalized, err)
	}
	resolved, err := catalog.resolveSubcommandID(commandID, "quick")
	if err != nil || resolved != runID {
		t.Fatalf("subcommand route projection = %q, %v", resolved, err)
	}
	if got := catalog.descriptor.Help[0].Usage; got != "aicoding demo run [--json]" {
		t.Fatalf("help projection = %q", got)
	}
	tokens, err := catalog.invocationTokens(commandID, []SubcommandID{runID}, []string{"--json"})
	if err != nil || strings.Join(tokens, " ") != "aicoding demo run --json" {
		t.Fatalf("quickstart projection = %#v, %v", tokens, err)
	}
}

func TestCommandCatalogRejectsProfileHelpOutsideProductVocabulary(t *testing.T) {
	for _, vocabulary := range []string{"Smoke|Lifecycle", "Smoke|Full|Release|Canary"} {
		_, err := newCommandCatalog(
			[]commandRoute{{descriptor: CommandDescriptor{ID: "broken", Name: "broken", LatencyClass: LatencyFast}, handler: runBootstrap}},
			[]HelpSection{{ID: HelpUsage, Title: "Usage:"}},
			[]HelpForm{{Command: "broken", Section: HelpUsage, Usage: "aicoding broken --profile " + vocabulary}},
		)
		if err == nil || !strings.Contains(err.Error(), "aicoding broken") || !strings.Contains(err.Error(), vocabulary) {
			t.Fatalf("profile vocabulary %q was not rejected with command context: %v", vocabulary, err)
		}
	}
}

func TestCatalogRegistersKitTestAndProfileFreeQuickstart(t *testing.T) {
	kitCommand, ok := findCommandByID(Catalog().Commands, CommandKit)
	if !ok {
		t.Fatal("kit command is missing")
	}
	kitTest, ok := findSubcommandByID(kitCommand.Subcommands, SubKitTest)
	if !ok || kitTest.Name != "test" {
		t.Fatalf("kit test is not formally registered: %#v", kitTest)
	}
	for _, form := range Catalog().Quickstarts {
		if form.Operation != "test" {
			continue
		}
		tokens, err := commands.invocationTokens(form.Command, form.Path, form.Args)
		if err != nil {
			t.Fatal(err)
		}
		command := strings.Join(tokens, " ")
		if command != "aicoding kit test --kit {kit} --json" || strings.Contains(command, "--profile") {
			t.Fatalf("kit test quickstart is not catalog-derived and profile-free: %q", command)
		}
		return
	}
	t.Fatal("kit test quickstart is missing")
}
