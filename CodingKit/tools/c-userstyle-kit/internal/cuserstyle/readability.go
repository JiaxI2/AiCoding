package cuserstyle

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	callExpressionRE = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	ifBranchRE       = regexp.MustCompile(`\bif\s*\(`)
	caseBranchRE     = regexp.MustCompile(`\bcase\b[^:\n]*:`)
	defaultBranchRE  = regexp.MustCompile(`\bdefault\s*:`)
	numberedIntentRE = regexp.MustCompile(`^\s*/\*\s*[1-9][0-9]*[.)、]\s+\S.*\*/\s*$`)
	fallthroughRE    = regexp.MustCompile(`(?i)fall[ -]?through|贯穿|继续执行下一`)
	caseExitRE       = regexp.MustCompile(`\b(?:break|continue|return)\b`)
)

var nonCallIdentifiers = map[string]struct{}{
	"_Alignof": {}, "_Generic": {}, "_Static_assert": {},
	"do": {}, "else": {}, "for": {}, "if": {}, "return": {}, "sizeof": {}, "switch": {}, "while": {},
}

type sourceReadabilityAnalysis struct {
	originalLines []string
	cleanLines    []string
	lineDepth     []int
	functions     []analyzedFunction
	summary       FileReadabilitySummary
}

type analyzedFunction struct {
	info      functionInfo
	startLine int
	openLine  int
	openCol   int
	endLine   int
	endCol    int
	calls     map[string]int
}

type caseBranchInfo struct {
	handled              bool
	hasIntent            bool
	willFallThrough      bool
	hasFallthroughIntent bool
}

func (analysis sourceReadabilityAnalysis) functionAtStart(startLine int) (FunctionReadability, analyzedFunction, bool) {
	for index, function := range analysis.functions {
		if function.startLine == startLine && index < len(analysis.summary.Functions) {
			return analysis.summary.Functions[index], function, true
		}
	}
	return FunctionReadability{}, analyzedFunction{}, false
}

// analyzeReadability performs the lexical readability analysis used by lint and verify.
// Semantic findings are returned as manual-review items and never become gate errors here.
func analyzeReadability(path string, data []byte, cfg Config) FileReadabilitySummary {
	return analyzeSourceReadability(path, data, cfg).summary
}

func analyzeSourceReadability(path string, data []byte, cfg Config) sourceReadabilityAnalysis {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	originalLines := strings.Split(text, "\n")
	cleanText := maskCSource(text)
	cleanLines := strings.Split(cleanText, "\n")
	blankPreprocessorLines(originalLines, cleanLines)

	analysis := sourceReadabilityAnalysis{
		originalLines: originalLines,
		cleanLines:    cleanLines,
		lineDepth:     calculateLineDepth(cleanLines),
		summary: FileReadabilitySummary{
			File:         filepath.ToSlash(path),
			Analyzer:     "lexical",
			Functions:    make([]FunctionReadability, 0),
			ManualReview: make([]ReviewItem, 0),
		},
	}

	functionMacros := collectFunctionLikeMacros(originalLines)
	for lineIndex := 0; lineIndex < len(originalLines); lineIndex++ {
		fn, ok := parseFunctionAt(originalLines, lineIndex)
		if !ok || fn.Prototype {
			continue
		}
		openLine, openCol, ok := findOpeningBrace(cleanLines, lineIndex)
		if !ok {
			continue
		}
		endLine, endCol, ok := findClosingBrace(cleanLines, openLine, openCol)
		if !ok {
			continue
		}

		body := functionBodyText(cleanLines, openLine, openCol, endLine, endCol)
		effectiveLines, maxNesting := measureFunctionBody(cleanLines, openLine, openCol, endLine, endCol)
		branchCount := len(ifBranchRE.FindAllStringIndex(body, -1)) +
			len(caseBranchRE.FindAllStringIndex(body, -1)) +
			len(defaultBranchRE.FindAllStringIndex(body, -1))
		calls := countDirectCalls(body, functionMacros)
		callees := sortedMapKeys(calls)
		policy := cfg.Readability.ComplexFunction
		complex := effectiveLines >= policy.MinEffectiveLines &&
			(branchCount >= policy.MinBranches || maxNesting >= policy.MinNesting)

		functionSummary := FunctionReadability{
			Name:                   fn.Name,
			Line:                   fn.Line,
			Static:                 fn.Static,
			EffectiveLines:         effectiveLines,
			BranchCount:            branchCount,
			MaxNesting:             maxNesting,
			FanOut:                 len(callees),
			Callees:                callees,
			Complex:                complex,
			NumberedFlowDocumented: hasSequentialNumberedFlow(precedingDoxygen(originalLines, lineIndex)),
		}
		analysis.functions = append(analysis.functions, analyzedFunction{
			info:      fn,
			startLine: lineIndex,
			openLine:  openLine,
			openCol:   openCol,
			endLine:   endLine,
			endCol:    endCol,
			calls:     calls,
		})
		analysis.summary.Functions = append(analysis.summary.Functions, functionSummary)
		lineIndex = endLine
	}

	completeCallGraph(&analysis, cleanText, cfg)
	sort.SliceStable(analysis.summary.Functions, func(i, j int) bool {
		if analysis.summary.Functions[i].Line == analysis.summary.Functions[j].Line {
			return analysis.summary.Functions[i].Name < analysis.summary.Functions[j].Name
		}
		return analysis.summary.Functions[i].Line < analysis.summary.Functions[j].Line
	})
	sort.SliceStable(analysis.summary.ManualReview, func(i, j int) bool {
		if analysis.summary.ManualReview[i].Line == analysis.summary.ManualReview[j].Line {
			return analysis.summary.ManualReview[i].Rule < analysis.summary.ManualReview[j].Rule
		}
		return analysis.summary.ManualReview[i].Line < analysis.summary.ManualReview[j].Line
	})
	return analysis
}

func completeCallGraph(analysis *sourceReadabilityAnalysis, cleanText string, cfg Config) {
	incoming := make(map[string]int)
	for _, fn := range analysis.functions {
		for callee, count := range fn.calls {
			incoming[callee] += count
		}
	}
	for index := range analysis.summary.Functions {
		functionSummary := &analysis.summary.Functions[index]
		functionSummary.IncomingDirectCalls = incoming[functionSummary.Name]
		if functionSummary.Complex && cfg.Readability.ReportNonObviousBranches {
			analysis.summary.ManualReview = append(analysis.summary.ManualReview, ReviewItem{
				File:     analysis.summary.File,
				Line:     functionSummary.Line,
				Function: functionSummary.Name,
				Rule:     "review.comment.logical-blocks",
				Message:  "人工确认编号概览与函数内主要逻辑块、非显然 if/else 的真实意图一致。",
			})
		}
		if !cfg.Readability.ReportSingleCallStaticHelpers || !functionSummary.Static ||
			functionSummary.IncomingDirectCalls != 1 || hasAddressTaken(cleanText, functionSummary.Name) {
			continue
		}
		if index < len(analysis.functions) && analysis.functions[index].calls[functionSummary.Name] > 0 {
			continue
		}
		functionSummary.SingleCallStaticHelper = true
		analysis.summary.ManualReview = append(analysis.summary.ManualReview, ReviewItem{
			File:     analysis.summary.File,
			Line:     functionSummary.Line,
			Function: functionSummary.Name,
			Rule:     "review.function.single-call-static-helper",
			Message:  "该静态函数只有一个直接调用点；人工确认其是否代表独立职责，不能仅凭调用次数决定合并。",
		})
	}
}

func readabilityDiagnostics(path string, analysis sourceReadabilityAnalysis, ranges []LineRange, cfg Config, changedOnly bool) []Diagnostic {
	var diagnostics []Diagnostic
	for index, functionSummary := range analysis.summary.Functions {
		if index >= len(analysis.functions) {
			break
		}
		fn := analysis.functions[index]
		if changedOnly && !spanSelected(fn.startLine+1, fn.endLine+1, ranges) {
			continue
		}
		if functionSummary.Complex && cfg.Readability.ComplexFunction.RequireNumberedFlow &&
			!functionSummary.NumberedFlowDocumented {
			diagnostics = append(diagnostics, diag(path, functionSummary.Line,
				"documentation.function-flow",
				"复杂函数的定义文档必须包含从 1、2 开始的连续编号流程概览。"))
		}
	}
	if cfg.Readability.RequireNumberedIntentCommentPlacement {
		diagnostics = append(diagnostics, validateNumberedIntentPlacement(path, analysis, ranges, changedOnly)...)
	}
	return diagnostics
}

func validateNumberedIntentPlacement(path string, analysis sourceReadabilityAnalysis, ranges []LineRange, changedOnly bool) []Diagnostic {
	var diagnostics []Diagnostic
	for _, fn := range analysis.functions {
		for index := fn.openLine + 1; index < fn.endLine; index++ {
			if !numberedIntentRE.MatchString(analysis.originalLines[index]) ||
				(changedOnly && !lineSelected(index+1, ranges)) {
				continue
			}
			previous := index - 1
			if previous > fn.openLine && strings.TrimSpace(analysis.originalLines[previous]) != "" &&
				!isCaseLabel(strings.TrimSpace(analysis.cleanLines[previous])) &&
				strings.TrimSpace(analysis.cleanLines[previous]) != "{" {
				diagnostics = append(diagnostics, diag(path, index+1, "comment.numbered-intent-placement",
					"编号逻辑块注释与上一代码块之间必须保留空行。"))
			}
			next := index + 1
			if next >= fn.endLine || strings.TrimSpace(analysis.originalLines[next]) == "" ||
				leadingIndent(analysis.originalLines[index]) != leadingIndent(analysis.originalLines[next]) {
				diagnostics = append(diagnostics, diag(path, index+1, "comment.numbered-intent-placement",
					"编号逻辑块注释必须与下方代码紧邻并保持相同缩进。"))
			}
		}
	}
	return diagnostics
}

func inspectCaseBranch(analysis sourceReadabilityAnalysis, index int) caseBranchInfo {
	info := caseBranchInfo{}
	if index < 0 || index >= len(analysis.cleanLines) {
		return info
	}
	labelDepth := analysis.lineDepth[index]
	commentBeforeCode := false
	bodyParts := make([]string, 0)
	boundaryIsCase := false

	originalAfterColon := textAfterColon(analysis.originalLines[index])
	cleanAfterColon := strings.TrimSpace(textAfterColon(analysis.cleanLines[index]))
	if strings.HasPrefix(strings.TrimSpace(originalAfterColon), "/*") {
		commentBeforeCode = true
	}
	if cleanAfterColon != "" {
		info.handled = true
		bodyParts = append(bodyParts, cleanAfterColon)
	}

	for lineIndex := index + 1; lineIndex < len(analysis.cleanLines); lineIndex++ {
		cleanTrim := strings.TrimSpace(analysis.cleanLines[lineIndex])
		originalTrim := strings.TrimSpace(analysis.originalLines[lineIndex])
		if analysis.lineDepth[lineIndex] < labelDepth {
			break
		}
		if analysis.lineDepth[lineIndex] == labelDepth && isCaseLabel(cleanTrim) {
			boundaryIsCase = true
			break
		}
		if cleanTrim == "" {
			if !info.handled && strings.HasPrefix(originalTrim, "/*") {
				commentBeforeCode = true
			}
			if fallthroughRE.MatchString(originalTrim) {
				info.hasFallthroughIntent = true
			}
			continue
		}
		if cleanTrim == "{" || cleanTrim == "}" {
			continue
		}
		info.handled = true
		bodyParts = append(bodyParts, cleanTrim)
	}
	info.hasIntent = info.handled && commentBeforeCode
	body := strings.Join(bodyParts, "\n")
	info.willFallThrough = info.handled && boundaryIsCase && !caseExitRE.MatchString(body)
	return info
}

func maskCSource(source string) string {
	const (
		stateCode = iota
		stateBlockComment
		stateLineComment
		stateString
		stateCharacter
	)
	state := stateCode
	escaped := false
	var builder strings.Builder
	builder.Grow(len(source))
	for index := 0; index < len(source); index++ {
		character := source[index]
		next := byte(0)
		if index+1 < len(source) {
			next = source[index+1]
		}
		switch state {
		case stateCode:
			switch {
			case character == '/' && next == '*':
				builder.WriteString("  ")
				index++
				state = stateBlockComment
			case character == '/' && next == '/':
				builder.WriteString("  ")
				index++
				state = stateLineComment
			case character == '"':
				builder.WriteByte(' ')
				state = stateString
				escaped = false
			case character == '\'':
				builder.WriteByte(' ')
				state = stateCharacter
				escaped = false
			default:
				builder.WriteByte(character)
			}
		case stateBlockComment:
			if character == '*' && next == '/' {
				builder.WriteString("  ")
				index++
				state = stateCode
			} else if character == '\n' {
				builder.WriteByte('\n')
			} else {
				builder.WriteByte(' ')
			}
		case stateLineComment:
			if character == '\n' {
				builder.WriteByte('\n')
				state = stateCode
			} else {
				builder.WriteByte(' ')
			}
		case stateString, stateCharacter:
			terminator := byte('"')
			if state == stateCharacter {
				terminator = '\''
			}
			if character == '\n' {
				builder.WriteByte('\n')
				state = stateCode
				escaped = false
			} else {
				builder.WriteByte(' ')
				if !escaped && character == terminator {
					state = stateCode
				}
				if character == '\\' && !escaped {
					escaped = true
				} else {
					escaped = false
				}
			}
		}
	}
	return builder.String()
}

func blankPreprocessorLines(originalLines, cleanLines []string) {
	continuation := false
	for index := range cleanLines {
		original := ""
		if index < len(originalLines) {
			original = originalLines[index]
		}
		isDirective := continuation || strings.HasPrefix(strings.TrimSpace(original), "#")
		if !isDirective {
			continue
		}
		cleanLines[index] = strings.Repeat(" ", len(cleanLines[index]))
		continuation = strings.HasSuffix(strings.TrimSpace(original), "\\")
	}
}

func calculateLineDepth(lines []string) []int {
	depths := make([]int, len(lines))
	depth := 0
	for index, line := range lines {
		depths[index] = depth
		for _, character := range line {
			if character == '{' {
				depth++
			} else if character == '}' && depth > 0 {
				depth--
			}
		}
	}
	return depths
}

func collectFunctionLikeMacros(lines []string) map[string]struct{} {
	macros := make(map[string]struct{})
	for _, line := range lines {
		match := macroRE.FindStringSubmatch(line)
		if match != nil && match[2] != "" {
			macros[match[1]] = struct{}{}
		}
	}
	return macros
}

func findOpeningBrace(lines []string, start int) (int, int, bool) {
	limit := start + 8
	if limit > len(lines) {
		limit = len(lines)
	}
	for lineIndex := start; lineIndex < limit; lineIndex++ {
		if column := strings.IndexByte(lines[lineIndex], '{'); column >= 0 {
			return lineIndex, column, true
		}
	}
	return 0, 0, false
}

func findClosingBrace(lines []string, openLine, openCol int) (int, int, bool) {
	depth := 0
	for lineIndex := openLine; lineIndex < len(lines); lineIndex++ {
		startColumn := 0
		if lineIndex == openLine {
			startColumn = openCol
		}
		for column := startColumn; column < len(lines[lineIndex]); column++ {
			switch lines[lineIndex][column] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return lineIndex, column, true
				}
			}
		}
	}
	return 0, 0, false
}

func functionBodyText(lines []string, openLine, openCol, endLine, endCol int) string {
	parts := make([]string, 0, endLine-openLine+1)
	for lineIndex := openLine; lineIndex <= endLine; lineIndex++ {
		line := lines[lineIndex]
		start := 0
		end := len(line)
		if lineIndex == openLine {
			start = openCol + 1
		}
		if lineIndex == endLine {
			end = endCol
		}
		if start > end {
			start = end
		}
		parts = append(parts, line[start:end])
	}
	return strings.Join(parts, "\n")
}

func measureFunctionBody(lines []string, openLine, openCol, endLine, endCol int) (int, int) {
	body := functionBodyText(lines, openLine, openCol, endLine, endCol)
	effectiveLines := 0
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim != "" && trim != "{" && trim != "}" {
			effectiveLines++
		}
	}
	depth := 0
	maxNesting := 0
	for _, character := range body {
		if character == '{' {
			depth++
			if depth > maxNesting {
				maxNesting = depth
			}
		} else if character == '}' && depth > 0 {
			depth--
		}
	}
	return effectiveLines, maxNesting
}

func countDirectCalls(body string, functionMacros map[string]struct{}) map[string]int {
	calls := make(map[string]int)
	for _, match := range callExpressionRE.FindAllStringSubmatch(body, -1) {
		name := match[1]
		if _, excluded := nonCallIdentifiers[name]; excluded {
			continue
		}
		if _, isMacro := functionMacros[name]; isMacro {
			continue
		}
		calls[name]++
	}
	return calls
}

func sortedMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func hasAddressTaken(cleanSource, name string) bool {
	for index := 0; index < len(cleanSource); index++ {
		if cleanSource[index] != '&' {
			continue
		}
		cursor := index + 1
		for cursor < len(cleanSource) && (cleanSource[cursor] == ' ' || cleanSource[cursor] == '\t') {
			cursor++
		}
		if strings.HasPrefix(cleanSource[cursor:], name) {
			end := cursor + len(name)
			if end == len(cleanSource) || !isIdentifierCharacter(cleanSource[end]) {
				return true
			}
		}
	}
	return false
}

func isIdentifierCharacter(character byte) bool {
	return character == '_' || character >= '0' && character <= '9' ||
		character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func spanSelected(startLine, endLine int, ranges []LineRange) bool {
	for _, lineRange := range ranges {
		if startLine <= lineRange.End && endLine >= lineRange.Start {
			return true
		}
	}
	return false
}

func leadingIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func textAfterColon(line string) string {
	if index := strings.IndexByte(line, ':'); index >= 0 {
		return line[index+1:]
	}
	return ""
}
