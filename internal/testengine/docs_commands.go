package testengine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const architectureDiagramNodeBudget = 20

type mermaidArchitectureDiagramDocument struct {
	path  string
	count int
}

type readmeSVGArchitectureVariant struct {
	path   string
	marker string
}

var (
	mermaidArchitectureDiagramDocuments = []mermaidArchitectureDiagramDocument{
		{path: "docs/architecture/PRIMITIVE_CONSTITUTION.md", count: 1},
		{path: "docs/architecture/AICODING_CORE_ARCHITECTURE.md", count: 1},
		{path: "docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md", count: 2},
		{path: "docs/reference/KIT_PLUGIN_VIEW.md", count: 1},
		{path: "docs/COMMANDS.md", count: 1},
	}
	readmeSVGArchitectureVariants = []readmeSVGArchitectureVariant{
		{path: "docs/assets/aicoding-overview-light.svg", marker: "#gh-light-mode-only"},
		{path: "docs/assets/aicoding-overview-dark.svg", marker: "#gh-dark-mode-only"},
	}
	diagramCommandPattern = regexp.MustCompile(`\baicoding(?:\.exe)?[ \t]+([a-z][a-z0-9-]*)\b`)
	diagramNodePattern    = regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9_]*)\s*(?:\[|\(|\{)`)
	visioSVGTitlePattern  = regexp.MustCompile(`<title>VMCP_([^<]+)</title>`)
)

type mermaidDiagram struct {
	source    string
	startLine int
}

func checkArchitectureDiagrams(repo string) error {
	commands, err := typedCatalogCommandNames(filepath.Join(repo, "internal/cli/catalog.go"))
	if err != nil {
		return err
	}
	if err := checkReadmeSVGArchitectureDiagram(repo, commands); err != nil {
		return err
	}

	for _, document := range mermaidArchitectureDiagramDocuments {
		rel := document.path
		path := filepath.Join(repo, filepath.FromSlash(rel))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rel, readErr)
		}
		diagrams, parseErr := mermaidDiagrams(string(data))
		if parseErr != nil {
			return fmt.Errorf("%s: %w", rel, parseErr)
		}
		if len(diagrams) != document.count {
			return fmt.Errorf("%s: expected exactly %d Mermaid diagram(s), found %d", rel, document.count, len(diagrams))
		}
		for _, diagram := range diagrams {
			if err := checkMermaidDiagram(rel, diagram, commands); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkReadmeSVGArchitectureDiagram(repo string, commands map[string]struct{}) error {
	readmeData, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		return fmt.Errorf("read README.md: %w", err)
	}
	readme := string(readmeData)
	var baseline []string
	for _, variant := range readmeSVGArchitectureVariants {
		reference := variant.path + variant.marker
		if strings.Count(readme, reference) != 1 {
			return fmt.Errorf("README.md: expected exactly one themed SVG architecture reference %q", reference)
		}
		data, readErr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(variant.path)))
		if readErr != nil {
			return fmt.Errorf("read %s: %w", variant.path, readErr)
		}
		source := string(data)
		if !strings.Contains(source, "<svg") || !strings.Contains(source, "</svg>") {
			return fmt.Errorf("%s: exported architecture asset is not SVG", variant.path)
		}
		nodes := visioSVGNodeIDs(source)
		if len(nodes) == 0 || len(nodes) > architectureDiagramNodeBudget {
			return fmt.Errorf("%s: Visio SVG diagram has %d explicit nodes; expected 1..%d", variant.path, len(nodes), architectureDiagramNodeBudget)
		}
		if baseline == nil {
			baseline = nodes
		} else if strings.Join(baseline, "\n") != strings.Join(nodes, "\n") {
			return fmt.Errorf("%s: themed Visio SVG node set differs from %s", variant.path, readmeSVGArchitectureVariants[0].path)
		}
		if err := checkDiagramCommands(variant.path, source, 1, commands); err != nil {
			return err
		}
	}
	return nil
}

func visioSVGNodeIDs(source string) []string {
	ids := []string{}
	for _, match := range visioSVGTitlePattern.FindAllStringSubmatch(source, -1) {
		if strings.HasPrefix(match[1], "e-") {
			continue
		}
		ids = append(ids, match[1])
	}
	return ids
}

func typedCatalogCommandNames(path string) (map[string]struct{}, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse typed command catalog: %w", err)
	}

	commands := map[string]struct{}{}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		ident, ok := literal.Type.(*ast.Ident)
		if !ok || ident.Name != "CommandDescriptor" {
			return true
		}
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := field.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Name":
				if name, ok := goStringLiteral(field.Value); ok {
					commands[name] = struct{}{}
				}
			case "Aliases":
				aliases, ok := field.Value.(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, alias := range aliases.Elts {
					if name, ok := goStringLiteral(alias); ok {
						commands[name] = struct{}{}
					}
				}
			}
		}
		return true
	})
	if len(commands) == 0 {
		return nil, fmt.Errorf("typed command catalog has no literal command names: %s", path)
	}
	return commands, nil
}

func goStringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func mermaidDiagrams(markdown string) ([]mermaidDiagram, error) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var diagrams []mermaidDiagram
	for index := 0; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "```mermaid" {
			continue
		}
		startLine := index + 2
		end := index + 1
		for end < len(lines) && strings.TrimSpace(lines[end]) != "```" {
			end++
		}
		if end == len(lines) {
			return nil, fmt.Errorf("unclosed Mermaid fence at line %d", index+1)
		}
		diagrams = append(diagrams, mermaidDiagram{
			source:    strings.Join(lines[index+1:end], "\n"),
			startLine: startLine,
		})
		index = end
	}
	return diagrams, nil
}

func checkMermaidDiagram(rel string, diagram mermaidDiagram, commands map[string]struct{}) error {
	if !hasMermaidFlowHeader(diagram.source) {
		return fmt.Errorf("%s:%d: Mermaid diagram must declare graph or flowchart", rel, diagram.startLine)
	}

	nodes := map[string]struct{}{}
	for _, match := range diagramNodePattern.FindAllStringSubmatch(diagram.source, -1) {
		nodes[match[1]] = struct{}{}
	}
	if len(nodes) == 0 || len(nodes) > architectureDiagramNodeBudget {
		return fmt.Errorf("%s:%d: Mermaid diagram has %d explicit nodes; expected 1..%d", rel, diagram.startLine, len(nodes), architectureDiagramNodeBudget)
	}

	return checkDiagramCommands(rel, diagram.source, diagram.startLine, commands)
}

func checkDiagramCommands(rel, source string, startLine int, commands map[string]struct{}) error {
	for _, match := range diagramCommandPattern.FindAllStringSubmatchIndex(source, -1) {
		command := source[match[2]:match[3]]
		if _, ok := commands[command]; ok {
			continue
		}
		line := startLine + strings.Count(source[:match[0]], "\n")
		return fmt.Errorf("%s:%d: diagram command %q is absent from internal/cli typed catalog", rel, line, "aicoding "+command)
	}
	return nil
}

func hasMermaidFlowHeader(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		fields := strings.Fields(line)
		return len(fields) > 0 && (fields[0] == "graph" || fields[0] == "flowchart")
	}
	return false
}
