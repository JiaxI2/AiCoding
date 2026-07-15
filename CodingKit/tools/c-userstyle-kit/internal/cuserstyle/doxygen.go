package cuserstyle

import (
	"regexp"
	"strconv"
	"strings"
)

type functionInfo struct {
	Name       string
	ReturnType string
	Parameters []string
	Static     bool
	Prototype  bool
	Line       int
}

var (
	functionLineRE      = regexp.MustCompile(`^\s*(static\s+)?([A-Za-z_][A-Za-z0-9_\s\*]*?)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^;{}]*)\)\s*(?:;|\{)?\s*$`)
	pointerReturnNameRE = regexp.MustCompile(`(\*+)\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	doxygenBriefRE      = regexp.MustCompile(`(?m)^\s*\*\s*@brief\s+\S`)
	doxygenReturnRE     = regexp.MustCompile(`(?m)^\s*\*\s*@return\s+\S`)
	doxygenDetailsRE    = regexp.MustCompile(`(?m)^\s*\*\s*@details(?:\s|$)`)
	numberedFlowItemRE  = regexp.MustCompile(`^\s*\*\s*([1-9][0-9]*)[.)、]\s+\S`)
	doxygenTagRE        = regexp.MustCompile(`^\s*\*\s*@([A-Za-z_][A-Za-z0-9_]*)\b`)
)

func parseFunctionLine(line string, lineNo int) (functionInfo, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "typedef ") || strings.Contains(trimmed, "(*") {
		return functionInfo{}, false
	}
	// Normalize the legal TYPE *Function(...) spelling without weakening the
	// return-type/name boundary in functionLineRE.
	normalized := pointerReturnNameRE.ReplaceAllString(line, `$1 $2(`)
	m := functionLineRE.FindStringSubmatch(normalized)
	if m == nil {
		return functionInfo{}, false
	}
	prefix := strings.TrimSpace(m[2])
	prefixKeyword := strings.Fields(prefix)[0]
	switch prefixKeyword {
	case "do", "else", "for", "if", "return", "sizeof", "switch", "while":
		return functionInfo{}, false
	}
	params := parseParameterNames(m[4])
	return functionInfo{
		Name: m[3], ReturnType: prefix, Parameters: params,
		Static:    strings.TrimSpace(m[1]) != "",
		Prototype: strings.HasSuffix(strings.TrimSpace(line), ";"),
		Line:      lineNo,
	}, true
}

func parseFunctionAt(lines []string, index int) (functionInfo, bool) {
	const maxSignatureLines = 8
	var parts []string

	for offset := 0; offset < maxSignatureLines && (index+offset) < len(lines); offset++ {
		trim := strings.TrimSpace(lines[index+offset])
		if trim == "" || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "/*") ||
			strings.HasPrefix(trim, "*") {
			if offset == 0 {
				return functionInfo{}, false
			}
			break
		}
		parts = append(parts, trim)
		if strings.HasSuffix(trim, ";") || strings.HasSuffix(trim, "{") {
			break
		}
		if offset > 0 && strings.Contains(trim, "=") {
			break
		}
	}
	if len(parts) == 0 {
		return functionInfo{}, false
	}
	return parseFunctionLine(strings.Join(parts, " "), index+1)
}

func parseParameterNames(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "void" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "..." {
			continue
		}
		fields := regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`).FindAllString(p, -1)
		if len(fields) > 0 {
			name := fields[len(fields)-1]
			if name != "const" && name != "volatile" {
				out = append(out, name)
			}
		}
	}
	return out
}

func precedingDoxygen(lines []string, index int) string {
	i := index - 1
	for i >= 0 && strings.TrimSpace(lines[i]) == "" {
		i--
	}
	if i < 0 || strings.TrimSpace(lines[i]) != "*/" {
		return ""
	}
	end := i
	for i >= 0 {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "/**") {
			return strings.Join(lines[i:end+1], "\n")
		}
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "/*") {
			return ""
		}
		i--
	}
	return ""
}

func hasSequentialNumberedFlow(block string) bool {
	inDetails := false
	expected := 1
	matched := 0
	for _, line := range strings.Split(block, "\n") {
		if tag := doxygenTagRE.FindStringSubmatch(line); tag != nil {
			if tag[1] == "details" {
				if inDetails {
					return false
				}
				inDetails = true
				continue
			}
			if inDetails {
				break
			}
			continue
		}
		if !inDetails {
			continue
		}
		match := numberedFlowItemRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			return false
		}
		if number != expected {
			return false
		}
		matched++
		expected++
	}
	return matched >= 2
}

func validateFunctionDocs(path string, lines []string, index int, fn functionInfo, cfg Config) []Diagnostic {
	if !cfg.Docs.AllFunctions {
		return nil
	}
	block := precedingDoxygen(lines, index)
	if fn.Static && fn.Prototype && cfg.Docs.RequireBarePrivatePrototype {
		if block != "" {
			return []Diagnostic{diag(path, fn.Line, "documentation.private-prototype",
				"静态函数前置声明只注册原型，完整文档应放在函数定义前。")}
		}
		return nil
	}
	if block == "" {
		return []Diagnostic{diag(path, fn.Line, "documentation.function", "函数声明或定义前缺少 Doxygen 文档。")}
	}
	var ds []Diagnostic
	if cfg.Docs.RequireBrief && !doxygenBriefRE.MatchString(block) {
		ds = append(ds, diag(path, fn.Line, "documentation.brief", "函数 Doxygen 缺少非空 @brief。"))
	}
	if cfg.Docs.RequireParamTags {
		for _, p := range fn.Parameters {
			pattern := `(?m)^\s*\*\s*@param(?:\[(?:in|out|in,out)\])?\s+` + regexp.QuoteMeta(p) + `\b`
			if !regexp.MustCompile(pattern).MatchString(block) {
				ds = append(ds, diag(path, fn.Line, "documentation.param", "参数 "+p+" 缺少 @param 文档。"))
				continue
			}
			if cfg.Docs.RequireParamDirection {
				dirPattern := `(?m)^\s*\*\s*@param\[(?:in|out|in,out)\]\s+` + regexp.QuoteMeta(p) + `\b`
				if !regexp.MustCompile(dirPattern).MatchString(block) {
					ds = append(ds, diag(path, fn.Line, "documentation.param-direction", "参数 "+p+" 缺少输入/输出方向。"))
				}
			}
		}
	}
	if cfg.Docs.RequireReturnTag && strings.TrimSpace(fn.ReturnType) != "void" {
		if !doxygenReturnRE.MatchString(block) {
			ds = append(ds, diag(path, fn.Line, "documentation.return", "非 void 函数缺少非空 @return。"))
		}
	}
	if !fn.Prototype && cfg.Docs.RequireDefinitionDetails &&
		!doxygenDetailsRE.MatchString(block) {
		ds = append(ds, diag(path, fn.Line, "documentation.definition-details",
			"函数定义文档缺少实现步骤、理由和约束的 @details。"))
	}
	if fn.Prototype && !fn.Static && cfg.Docs.RequirePublicPerformance &&
		!strings.Contains(block, "性能") {
		ds = append(ds, diag(path, fn.Line, "documentation.performance",
			"公开声明文档必须说明性能或执行上界。"))
	}
	if fn.Prototype && !fn.Static && cfg.Docs.RequirePublicReentrancy &&
		!strings.Contains(block, "可重入") && !strings.Contains(block, "并发") &&
		!strings.Contains(block, "互斥") && !strings.Contains(block, "中断") {
		ds = append(ds, diag(path, fn.Line, "documentation.reentrancy",
			"公开声明文档必须说明可重入、并发、互斥或中断约束。"))
	}
	return ds
}
