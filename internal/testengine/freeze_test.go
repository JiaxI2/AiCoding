package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFreezeChecksCurrentRepository(t *testing.T) {
	repo := filepath.Join("..", "..")
	checks := []struct {
		id    string
		check func() error
	}{
		{id: "FREEZE-001", check: func() error { return checkFrozenSchemas(repo) }},
		{id: "FREEZE-002", check: func() error { return checkUniqueProductionType(repo, "internal/report", "Result") }},
		{id: "FREEZE-003", check: func() error { return checkUniqueProductionType(repo, "internal/validationevidence", "Receipt") }},
		{id: "FREEZE-004", check: func() error { return checkLoopWorkCatalog(repo) }},
		{id: "FREEZE-005", check: func() error { return checkLoopDecideSignature(repo) }},
		{id: "FREEZE-006", check: func() error { return checkValidationFingerprintFields(repo) }},
		{id: "FREEZE-007", check: func() error { return checkKitManifestSourceOptional(repo) }},
		{id: "FREEZE-008", check: func() error { return checkTypedSubcommandCatalog(repo) }},
		{id: "FREEZE-009", check: func() error { return checkProductProfileVocabulary(repo) }},
	}
	for _, check := range checks {
		t.Run(check.id, func(t *testing.T) {
			if err := check.check(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUniqueProductionTypeRejectsDuplicate(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "internal/report/result.go", "package report\n\ntype Result struct{}\n")
	writeFreezeTestFile(t, repo, "internal/report/nested/duplicate.go", "package nested\n\ntype Result struct{}\n")
	err := checkUniqueProductionType(repo, "internal/report", "Result")
	if err == nil || !strings.Contains(err.Error(), "found 2") {
		t.Fatalf("duplicate Result was not rejected: %v", err)
	}
}

func TestRegistryContainsFreezeGates(t *testing.T) {
	t.Parallel()
	cfg, err := NormalizeConfig(Config{Repo: t.TempDir(), Profile: ProfileSmoke})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]TestCase{}
	for _, testCase := range Registry(cfg) {
		if strings.HasPrefix(testCase.ID, "FREEZE-") {
			found[testCase.ID] = testCase
		}
	}
	for _, id := range []string{"FREEZE-001", "FREEZE-002", "FREEZE-003", "FREEZE-004", "FREEZE-005", "FREEZE-006", "FREEZE-007", "FREEZE-008", "FREEZE-009"} {
		testCase, ok := found[id]
		if !ok || testCase.Kind != "static" || testCase.Severity != Required || len(testCase.Profiles) != len(allProfiles()) {
			t.Fatalf("invalid %s registry case: %#v", id, testCase)
		}
	}
}

func TestLoopWorkCatalogRejectsExecutionCommand(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "internal/cli/catalog.go", "package cli\n\nvar forbidden = \"aicoding work run --file SPEC.json\"\n")
	err := checkLoopWorkCatalog(repo)
	if err == nil || !strings.Contains(err.Error(), "work run") {
		t.Fatalf("work run was not rejected: %v", err)
	}
}

func TestLoopDecideSignatureRejectsFifthParameter(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "internal/loopkit/transition/transition.go", `package transition

func Decide(spec workspec.Spec, history []Attempt, gates []GateStatus, now time.Time, force bool) (Decision, error) {
	return Decision{}, nil
}
`)
	err := checkLoopDecideSignature(repo)
	if err == nil || !strings.Contains(err.Error(), "signature changed") {
		t.Fatalf("Decide signature drift was not rejected: %v", err)
	}
}

func TestValidationFingerprintRejectsFieldDrift(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "internal/validationevidence/model.go", "package validationevidence\n\ntype Fingerprint struct { Identity string }\n")
	err := checkValidationFingerprintFields(repo)
	if err == nil || !strings.Contains(err.Error(), "field list changed") {
		t.Fatalf("Fingerprint field drift was not rejected: %v", err)
	}
}

func TestKitManifestSourceRejectsRequired(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "config/schemas/kit-manifest.schema.json", `{"required":["source"],"properties":{"source":{}}}`)
	err := checkKitManifestSourceOptional(repo)
	if err == nil || !strings.Contains(err.Error(), "must remain optional") {
		t.Fatalf("required source was not rejected: %v", err)
	}
}

func TestTypedSubcommandCatalogRejectsExternalRoute(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeFreezeTestFile(t, repo, "internal/cli/catalog.go", `package cli

type commandRoute struct { descriptor CommandDescriptor; handler func([]string) }
type CommandDescriptor struct { Subcommands []SubcommandDescriptor }
type SubcommandDescriptor struct{}
type typedCatalog struct{}
func (typedCatalog) prepareInvocation(route any, args []string) ([]string, error) { return args, nil }
func mustCommandCatalog(routes []commandRoute) typedCatalog { return typedCatalog{} }
var commands = mustCommandCatalog([]commandRoute{
	{descriptor: CommandDescriptor{Subcommands: []SubcommandDescriptor{{}}}, handler: runKit},
})
var resolveCatalogSubcommandID func(any, ...string) (any, error)
`)
	writeFreezeTestFile(t, repo, "internal/cli/cli.go", `package cli

func Execute(commandArgs []string) {
	commandArgs, _ = commands.prepareInvocation(nil, commandArgs)
}
func runKit(args []string) {
	_, _ = resolveCatalogSubcommandID(nil, args[0])
	switch args[0] {
	case "shadow":
	}
}
`)
	err := checkTypedSubcommandCatalog(repo)
	if err == nil || !strings.Contains(err.Error(), "runKit") || !strings.Contains(err.Error(), "shadow") {
		t.Fatalf("catalog-external subcommand route was not rejected: %v", err)
	}
}

func TestProductProfileVocabularyRejectsIndependentHelp(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeProfileFreezeFixture(t, repo, `[]string{"Smoke", "Full", "Release"}`, `fs.String("profile", "Smoke", "Smoke, Full or Lifecycle")`)
	err := checkProductProfileVocabulary(repo)
	if err == nil || !strings.Contains(err.Error(), "runKit") || !strings.Contains(err.Error(), "outside the product vocabulary") {
		t.Fatalf("independent --profile help was not rejected: %v", err)
	}
}

func TestProductProfileVocabularyRejectsFourthValue(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeProfileFreezeFixture(t, repo, `[]string{"Smoke", "Full", "Release", "Canary"}`, `fs.String("profile", "Smoke", productProfileHelp())`)
	err := checkProductProfileVocabulary(repo)
	if err == nil || !strings.Contains(err.Error(), "Canary") || !strings.Contains(err.Error(), "want [Smoke Full Release]") {
		t.Fatalf("fourth --profile vocabulary was not rejected: %v", err)
	}
}

func writeProfileFreezeFixture(t *testing.T, repo, vocabulary, flagCall string) {
	t.Helper()
	writeFreezeTestFile(t, repo, "internal/cli/catalog.go", "package cli\n\nvar productProfileVocabulary = "+vocabulary+"\nfunc productProfileHelp() string { return \"\" }\n")
	writeFreezeTestFile(t, repo, "internal/cli/test.go", `package cli

func normalizeTestProfile(value string) {
	for range productProfileVocabulary {}
}
func runKit(fs interface{ String(string, string, string) *string }) {
	`+flagCall+`
}
`)
}

func writeFreezeTestFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
