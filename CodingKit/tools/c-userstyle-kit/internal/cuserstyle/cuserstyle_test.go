package cuserstyle

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func configPath() string {
	return filepath.Join("..", "..", "examples", "c-kit.json")
}

func TestGeneratedDemoPasses(t *testing.T) {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range goldenDemoFiles {
		name := filepath.ToSlash(file.Output)
		if !strings.HasSuffix(name, ".c") && !strings.HasSuffix(name, ".h") {
			continue
		}
		content := renderDemoFile(file, cfg)
		if ds := lintContent(name, []byte(content), nil, cfg, false); len(ds) != 0 {
			t.Fatalf("%s: %+v", name, ds)
		}
	}
}

func TestGeneratedDemoLayoutSeparatesAdvancedExamples(t *testing.T) {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	legacy := legacyDemoFiles[0]
	legacyPath := filepath.Join(out, filepath.FromSlash(legacy.Output))
	if err := os.WriteFile(legacyPath, []byte(renderDemoFile(legacy, cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	generated, err := generateDemo(cfg, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(generated) != len(goldenDemoFiles) {
		t.Fatalf("expected %d generated files, got %d", len(goldenDemoFiles), len(generated))
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy root-level advanced file still exists: %s", legacyPath)
	}

	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	rootNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		rootNames = append(rootNames, entry.Name())
	}
	if strings.Join(rootNames, ",") != "advanced,demo.c,demo.h" {
		t.Fatalf("unexpected public demo root layout: %v", rootNames)
	}
}

func TestAdvancedExampleNamesAreRendered(t *testing.T) {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		t.Fatal(err)
	}
	stateHeader := renderDemoFile(goldenDemoFiles[3], cfg)
	stateSource := renderDemoFile(goldenDemoFiles[4], cfg)
	if !strings.Contains(stateHeader, "#ifndef ADVANCED_STATE_MACHINE_H") ||
		!strings.Contains(stateSource, "#include \"state_machine.h\"") {
		t.Fatal("advanced state-machine names were not rendered consistently")
	}
}

func TestSnippetCatalogAndRendering(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "c-snippets.json")
	catalog, err := loadSnippetCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 9 {
		t.Fatalf("expected 9 snippets, got %d", len(catalog))
	}
	macroSnippet := catalog["C Simple Object Macro"]
	if len(macroSnippet.Body) == 0 || !strings.HasPrefix(macroSnippet.Body[0], "/* ") ||
		strings.HasPrefix(macroSnippet.Body[0], "/**") {
		t.Fatalf("simple object macro snippet must start with an ordinary one-line block comment: %+v", macroSnippet)
	}
	snippet := catalog["C File Header (CN)"]
	rendered := renderSnippet(
		snippet,
		"sensor.c",
		snippetValues{"3": "HU JIAXUAN", "6": "ExampleCompany"},
		time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC),
	)
	if !strings.Contains(rendered, "@file      sensor.c") ||
		!strings.Contains(rendered, "@date      2026-07-15") ||
		strings.Contains(rendered, "$0") {
		t.Fatalf("unexpected rendered snippet:\n%s", rendered)
	}
}

func TestEmbeddedAndExampleSnippetsMatch(t *testing.T) {
	example, err := os.ReadFile(filepath.Join("..", "..", "examples", "c-snippets.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(example) != defaultSnippetsJSON {
		t.Fatal("embedded init snippets and examples/c-snippets.json differ")
	}
}

func TestInitInstallsAndPreservesSnippetCustomization(t *testing.T) {
	root := t.TempDir()
	if err := RunInit([]string{"--root", root}); err != nil {
		t.Fatal(err)
	}
	snippetsPath := filepath.Join(root, "UserCfg", "UserStyle", "c-snippets.json")
	if _, err := os.Stat(snippetsPath); err != nil {
		t.Fatal(err)
	}
	custom := []byte("{\"Custom\":{\"prefix\":\"x\",\"body\":[\"x\"],\"description\":\"x\"}}\n")
	if err := os.WriteFile(snippetsPath, custom, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunInit([]string{"--root", root}); err == nil {
		t.Fatal("expected init without --force to preserve existing user files")
	}
	actual, err := os.ReadFile(snippetsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(custom) {
		t.Fatal("init changed an existing user snippet catalog without --force")
	}
}

func TestNegativeFixtureReportsExpectedRules(t *testing.T) {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		t.Fatal(err)
	}
	fixtureRoot := filepath.Join("..", "..", "tests", "fixtures", "lint")
	source, err := os.ReadFile(filepath.Join(fixtureRoot, "negative_rules.c"))
	if err != nil {
		t.Fatal(err)
	}
	expectedData, err := os.ReadFile(filepath.Join(fixtureRoot, "negative_rules.expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		RequiredRuleIDs []string `json:"requiredRuleIds"`
	}
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatal(err)
	}
	diagnostics := lintContent("negative_rules.c", source, nil, cfg, false)
	for _, rule := range expected.RequiredRuleIDs {
		if !hasRule(diagnostics, rule) {
			t.Fatalf("expected %s diagnostic: %+v", rule, diagnostics)
		}
	}
}

func TestMissingBriefFails(t *testing.T) {
	cfg, _ := LoadConfig(configPath())
	src := `/**
 * @file bad.c
 * @brief test
 */

/**
 * @param[in] value value
 * @return value
 */
int32_t DEMO_Bad(int32_t value)
{
    return value;
}
`
	ds := lintContent("bad.c", []byte(src), nil, cfg, false)
	if !hasRule(ds, "documentation.brief") {
		t.Fatalf("expected missing brief diagnostic: %+v", ds)
	}
}

func TestMissingParameterDocumentationFails(t *testing.T) {
	cfg, _ := LoadConfig(configPath())
	src := `/**
 * @file bad.c
 * @brief test
 */

/**
 * @brief test
 * @return value
 */
int32_t DEMO_Bad(int32_t value)
{
    return value;
}
`
	ds := lintContent("bad.c", []byte(src), nil, cfg, false)
	if !hasRule(ds, "documentation.param") {
		t.Fatalf("expected missing parameter diagnostic: %+v", ds)
	}
}

func TestUnsafeBodyFails(t *testing.T) {
	cfg, _ := LoadConfig(configPath())
	src := `/**
 * @file bad.c
 * @brief test
 */

/**
 * @brief test
 * @return result
 */
bool DEMO_Bad(void)
{
    void* buffer = malloc(8U);
    while (true)
    {
    }
    return buffer != NULL;
}
`
	ds := lintContent("bad.c", []byte(src), nil, cfg, false)
	if !hasRule(ds, "embedded.dynamic-allocation") || !hasRule(ds, "control.unbounded-loop") {
		t.Fatalf("expected body diagnostics: %+v", ds)
	}
}

func TestConfigPolicyDefaults(t *testing.T) {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Docs.EmployeeIDPolicy = ""
	cfg.Docs.ModificationHistoryPolicy = ""
	cfg.Macro.SimpleObjectCommentStyle = ""
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "c-kit.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Docs.EmployeeIDPolicy != "whenProvided" ||
		loaded.Docs.ModificationHistoryPolicy != "disabled" ||
		loaded.Macro.SimpleObjectCommentStyle != "block" {
		t.Fatalf("unexpected defaults: %+v %+v", loaded.Docs, loaded.Macro)
	}
}

func TestFileMetadataPolicies(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Docs.FileHeader = true
	cfg.Docs.EmployeeIDPolicy = "whenProvided"
	cfg.Docs.ModificationHistoryPolicy = "disabled"
	valid := `/**
 * @file policy.c
 * @brief policy test
 */
`
	if ds := lintContent("policy.c", []byte(valid), nil, cfg, false); hasRule(ds, "documentation.employee-id.placeholder") {
		t.Fatalf("omitted employee id must pass whenProvided: %+v", ds)
	}
	placeholder := `/**
 * @file policy.c
 * @brief policy test
 * @employee_id 不适用
 */
`
	if ds := lintContent("policy.c", []byte(placeholder), nil, cfg, false); !hasRule(ds, "documentation.employee-id.placeholder") {
		t.Fatalf("expected placeholder employee id diagnostic: %+v", ds)
	}
	history := `/**
 * @file policy.c
 * @brief policy test
 * 修改记录：2026-07-15，初始版本。
 */
`
	if ds := lintContent("policy.c", []byte(history), nil, cfg, false); !hasRule(ds, "documentation.modification-history.forbidden") {
		t.Fatalf("expected source history diagnostic: %+v", ds)
	}
	for _, marker := range []string{
		" * Modification: initial version.\n",
		" * Modification History: initial version.\n",
		" * 修改历史：2026-07-15，初始版本。\n",
	} {
		source := strings.Replace(valid, " */\n", marker+" */\n", 1)
		if ds := lintContent("policy.c", []byte(source), nil, cfg, false); !hasRule(ds, "documentation.modification-history.forbidden") {
			t.Fatalf("expected source history diagnostic for %q: %+v", marker, ds)
		}
	}

	cfg.Docs.EmployeeIDPolicy = "omit"
	actualEmployee := `/**
 * @file policy.c
 * @brief policy test
 * @employee_id 004201
 */
`
	if ds := lintContent("policy.c", []byte(actualEmployee), nil, cfg, false); !hasRule(ds, "documentation.employee-id.forbidden") {
		t.Fatalf("expected forbidden employee id diagnostic: %+v", ds)
	}
	cfg.Docs.EmployeeIDPolicy = "required"
	if ds := lintContent("policy.c", []byte(valid), nil, cfg, false); !hasRule(ds, "documentation.employee-id.missing") {
		t.Fatalf("expected required employee id diagnostic: %+v", ds)
	}
	if ds := lintContent("policy.c", []byte(actualEmployee), nil, cfg, false); hasRule(ds, "documentation.employee-id.missing") || hasRule(ds, "documentation.employee-id.placeholder") {
		t.Fatalf("actual required employee id must pass: %+v", ds)
	}

	cfg.Docs.ModificationHistoryPolicy = "maintenance-release"
	if ds := lintContent("policy.c", []byte(history), nil, cfg, false); hasRule(ds, "documentation.modification-history.forbidden") {
		t.Fatalf("maintenance-release must explicitly allow source history: %+v", ds)
	}
	cfg.Docs.ModificationHistoryPolicy = "required"
	if ds := lintContent("policy.c", []byte(valid), nil, cfg, false); !hasRule(ds, "documentation.modification-history.missing") {
		t.Fatalf("expected required modification history diagnostic: %+v", ds)
	}
}

func TestSimpleObjectMacroUsesOrdinaryBlockComment(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Macro.RequireDocumentation = true
	cfg.Macro.SimpleObjectCommentStyle = "block"
	valid := "/* 最大映射数量。 */\n#define DEMO_MAX_MAPPINGS 8U\n"
	if ds := lintContent("macro.c", []byte(valid), nil, cfg, false); len(ds) != 0 {
		t.Fatalf("ordinary one-line block comment must pass: %+v", ds)
	}
	invalid := "/** @brief 最大映射数量。 */\n#define DEMO_MAX_MAPPINGS 8U\n"
	if ds := lintContent("macro.c", []byte(invalid), nil, cfg, false); !hasRule(ds, "documentation.simple-macro-comment") {
		t.Fatalf("expected simple macro comment diagnostic: %+v", ds)
	}
}

func TestComplexFunctionRequiresSequentialNumberedFlow(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Readability.ComplexFunction = ComplexFunctionPolicy{
		MinEffectiveLines:   1,
		MinBranches:         1,
		MinNesting:          1,
		RequireNumberedFlow: true,
	}
	valid := `/**
 * @brief 执行带分支的流程。
 *
 * @details
 * 1. 检查输入是否满足处理条件。
 * 2. 返回处理结果或安全默认值。
 */
int32_t DEMO_NumberedFlow(int32_t value)
{
    if (value > 0)
    {
        return value;
    }
    return 0;
}
`
	summary := analyzeReadability("flow.c", []byte(valid), cfg)
	if len(summary.Functions) != 1 || !summary.Functions[0].Complex ||
		!summary.Functions[0].NumberedFlowDocumented {
		t.Fatalf("unexpected readability summary: %+v", summary)
	}
	if ds := lintContent("flow.c", []byte(valid), nil, cfg, false); hasRule(ds, "documentation.function-flow") {
		t.Fatalf("sequential 1/2 flow must pass: %+v", ds)
	}
	invalid := strings.Replace(valid,
		" * 1. 检查输入是否满足处理条件。\n * 2. 返回处理结果或安全默认值。\n",
		" * 检查输入并返回结果。\n", 1)
	if ds := lintContent("flow.c", []byte(invalid), nil, cfg, false); !hasRule(ds, "documentation.function-flow") {
		t.Fatalf("expected numbered flow diagnostic: %+v", ds)
	}
}

func TestSequentialNumberedFlowScansCompleteDetailsSequence(t *testing.T) {
	tests := []struct {
		name  string
		block string
		want  bool
	}{
		{
			name: "continuous details",
			block: `/**
 * @details
 * 1. 检查输入。
 * 2. 发布结果。
 * @param[in] value 输入。
 */`,
			want: true,
		},
		{
			name: "gap after two",
			block: `/**
 * @details
 * 1. 检查输入。
 * 2. 执行计算。
 * 4. 发布结果。
 */`,
			want: false,
		},
		{
			name: "duplicate",
			block: `/**
 * @details
 * 1. 检查输入。
 * 2. 执行计算。
 * 2. 发布结果。
 */`,
			want: false,
		},
		{
			name: "numbers outside details",
			block: `/**
 * @brief 说明。
 * 1. 检查输入。
 * 2. 发布结果。
 */`,
			want: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasSequentialNumberedFlow(test.block); got != test.want {
				t.Fatalf("hasSequentialNumberedFlow() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNumberedIntentCommentPlacementIsStructuralOnly(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Readability.RequireNumberedIntentCommentPlacement = true
	valid := `void DEMO_NumberedBlock(void)
{
    int32_t value = 0;

    /* 1. 发布本地计算结果。 */
    value = 1;
}
`
	if ds := lintContent("placement.c", []byte(valid), nil, cfg, false); hasRule(ds, "comment.numbered-intent-placement") {
		t.Fatalf("properly placed numbered comment must pass: %+v", ds)
	}
	invalid := strings.Replace(valid, "    int32_t value = 0;\n\n    /* 1.",
		"    int32_t value = 0;\n    /* 1.", 1)
	if ds := lintContent("placement.c", []byte(invalid), nil, cfg, false); !hasRule(ds, "comment.numbered-intent-placement") {
		t.Fatalf("expected numbered comment placement diagnostic: %+v", ds)
	}
	ordinary := strings.Replace(invalid, "/* 1. 发布本地计算结果。 */", "/* 发布本地计算结果。 */", 1)
	if ds := lintContent("placement.c", []byte(ordinary), nil, cfg, false); hasRule(ds, "comment.numbered-intent-placement") {
		t.Fatalf("ordinary logical comments remain a manual-review concern: %+v", ds)
	}
}

func TestHandledCaseAllowsConsecutiveEmptyLabels(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Docs.RequireCaseIntentComment = true
	cfg.Flow.RequireSwitchDefault = true
	cfg.Flow.RequireCaseBreak = true
	valid := `void DEMO_HandleCase(int32_t value)
{
    switch (value)
    {
        case 0:
        case 1:
            /* 两个值共享同一项处理。 */
            break;

        default:
            /* 其余值保持默认处理。 */
            break;
    }
}
`
	if ds := lintContent("case.c", []byte(valid), nil, cfg, false); hasRule(ds, "comment.case-intent") {
		t.Fatalf("consecutive empty label must not require a duplicate comment: %+v", ds)
	}
	invalid := strings.Replace(valid, "            /* 两个值共享同一项处理。 */\n", "", 1)
	ds := lintContent("case.c", []byte(invalid), nil, cfg, false)
	if countRule(ds, "comment.case-intent") != 1 {
		t.Fatalf("expected exactly one handled-case diagnostic: %+v", ds)
	}
}

func TestParseFunctionLineAcceptsPointerReturnSpacing(t *testing.T) {
	tests := []struct {
		name       string
		signature  string
		returnType string
	}{
		{name: "pointer attached", signature: "object_t *DEMO_Find(void)", returnType: "object_t *"},
		{name: "double pointer attached", signature: "object_t **DEMO_Find(void)", returnType: "object_t **"},
		{name: "pointer separated", signature: "object_t * DEMO_Find(void)", returnType: "object_t *"},
		{name: "non pointer", signature: "object_t DEMO_Find(void)", returnType: "object_t"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			function, ok := parseFunctionLine(test.signature, 7)
			if !ok {
				t.Fatalf("expected function signature to parse: %q", test.signature)
			}
			if function.Name != "DEMO_Find" || function.ReturnType != test.returnType || function.Line != 7 {
				t.Fatalf("unexpected function parse: %+v", function)
			}
		})
	}
}

func TestParseFunctionLineRejectsPointerExpressionStatements(t *testing.T) {
	statements := []string{
		"return *DEMO_Find();",
		"sizeof *DEMO_Find();",
	}
	for _, statement := range statements {
		if function, ok := parseFunctionLine(statement, 1); ok {
			t.Fatalf("expression statement parsed as function: %+v", function)
		}
	}
}

func TestPointerReturningFunctionParticipatesInReadabilityCallGraph(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Readability.ReportSingleCallStaticHelpers = true
	source := `typedef struct object object_t;

static object_t *DEMO_Find(void);

static object_t *DEMO_Find(void)
{
    return 0;
}

void DEMO_Run(void)
{
    (void)DEMO_Find();
}
`

	summary := analyzeReadability("pointer-return.c", []byte(source), cfg)
	found := false
	for _, function := range summary.Functions {
		if function.Name == "DEMO_Find" {
			found = function.Static && function.IncomingDirectCalls == 1 &&
				function.SingleCallStaticHelper
		}
	}
	if !found || !hasReviewRule(summary.ManualReview, "review.function.single-call-static-helper") {
		t.Fatalf("expected pointer-return helper in call graph: %+v", summary)
	}
}

func TestSingleCallStaticHelperIsManualReviewOnly(t *testing.T) {
	cfg := focusedLintConfig(t)
	cfg.Readability.ReportSingleCallStaticHelpers = true
	source := `static void DEMO_Helper(void);

static void DEMO_Helper(void)
{
}

void DEMO_Run(void)
{
    DEMO_Helper();
}
`
	summary := analyzeReadability("helper.c", []byte(source), cfg)
	found := false
	for _, function := range summary.Functions {
		if function.Name == "DEMO_Helper" {
			found = function.SingleCallStaticHelper && function.IncomingDirectCalls == 1
		}
	}
	if !found || !hasReviewRule(summary.ManualReview, "review.function.single-call-static-helper") {
		t.Fatalf("expected advisory-only helper candidate: %+v", summary)
	}
	if ds := lintContent("helper.c", []byte(source), nil, cfg, false); hasRule(ds, "review.function.single-call-static-helper") {
		t.Fatalf("manual helper review must not become a lint error: %+v", ds)
	}
}

func TestRunLintJSONIncludesReadability(t *testing.T) {
	cfg := focusedLintConfig(t)
	root := t.TempDir()
	configFile := filepath.Join(root, "c-kit.json")
	configData, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, configData, 0o644); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(root, "summary.c")
	if err := os.WriteFile(sourceFile, []byte("void DEMO_Run(void)\n{\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	runErr := RunLint([]string{
		"--config", configFile,
		"--scope", "files",
		"--file", sourceFile,
		"--json",
		"--summary",
	})
	closeErr := writer.Close()
	os.Stdout = originalStdout
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("lint summary failed: %v\n%s", runErr, output)
	}
	var result struct {
		OK          bool `json:"ok"`
		Readability struct {
			Files []FileReadabilitySummary `json:"files"`
		} `json:"readability"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid lint JSON: %v\n%s", err, output)
	}
	if !result.OK || len(result.Readability.Files) != 1 ||
		len(result.Readability.Files[0].Functions) != 1 {
		t.Fatalf("unexpected lint readability JSON: %+v", result)
	}
}

func focusedLintConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := LoadConfig(configPath())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Docs.FileHeader = false
	cfg.Docs.RequireFileMetadata = false
	cfg.Docs.AllFunctions = false
	cfg.Docs.RequireGlobalVariableDetail = false
	cfg.Docs.RequireExternC = false
	cfg.Docs.EmployeeIDPolicy = "whenProvided"
	cfg.Docs.ModificationHistoryPolicy = "disabled"
	cfg.Macro.RequireDocumentation = false
	cfg.Readability.RequireNumberedIntentCommentPlacement = false
	cfg.Readability.ComplexFunction.RequireNumberedFlow = false
	return cfg
}

func countRule(ds []Diagnostic, rule string) int {
	count := 0
	for _, diagnostic := range ds {
		if diagnostic.Rule == rule {
			count++
		}
	}
	return count
}

func hasReviewRule(items []ReviewItem, rule string) bool {
	for _, item := range items {
		if item.Rule == rule {
			return true
		}
	}
	return false
}

func hasRule(ds []Diagnostic, rule string) bool {
	for _, d := range ds {
		if d.Rule == rule {
			return true
		}
	}
	return false
}

func BenchmarkLintGeneratedSource(b *testing.B) {
	cfg, _ := LoadConfig(configPath())
	data := []byte(renderSource(cfg))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lintContent("demo.c", data, nil, cfg, false)
	}
}
