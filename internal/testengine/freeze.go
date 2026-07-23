package testengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var frozenSchemaPaths = []string{
	"config/schemas/cli-report.schema.json",
	"config/schemas/dependency-governance.schema.json",
	"config/schemas/kit-manifest.schema.json",
	"config/schemas/kit-registry.schema.json",
	"config/schemas/mcp-component.schema.json",
	"config/schemas/mcp-registry.schema.json",
}

func checkFrozenSchemas(repo string) error {
	return requirePaths(repo, frozenSchemaPaths...)
}

func checkUniqueProductionType(repo, root, typeName string) error {
	base := filepath.Join(repo, filepath.FromSlash(root))
	pattern := regexp.MustCompile(`(?m)^\s*type\s+` + regexp.QuoteMeta(typeName) + `\s+struct\s*\{`)
	matches := []string{}
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for range pattern.FindAll(raw, -1) {
			rel, relErr := filepath.Rel(repo, path)
			if relErr != nil {
				return relErr
			}
			matches = append(matches, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(matches)
	if len(matches) != 1 {
		return fmt.Errorf("%s must contain exactly one production type %s struct; found %d in %v", root, typeName, len(matches), matches)
	}
	return nil
}

func checkLoopWorkCatalog(repo string) error {
	raw, err := os.ReadFile(filepath.Join(repo, "internal", "cli", "catalog.go"))
	if err != nil {
		return err
	}
	for _, subcommand := range []string{"run", "prepare", "step"} {
		pattern := regexp.MustCompile(`"aicoding\s+work\s+` + regexp.QuoteMeta(subcommand) + `(?:\s|\")`)
		if pattern.Match(raw) {
			return fmt.Errorf("typed command catalog must not contain work %s", subcommand)
		}
	}
	return nil
}

func checkLoopDecideSignature(repo string) error {
	path := filepath.Join(repo, "internal", "loopkit", "transition", "transition.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return err
	}
	var matches []*ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "Decide" {
			matches = append(matches, function)
		}
	}
	if len(matches) != 1 {
		return fmt.Errorf("transition.Decide must have exactly one declaration; found %d", len(matches))
	}
	parameters, err := fieldTypes(matches[0].Type.Params)
	if err != nil {
		return err
	}
	results, err := fieldTypes(matches[0].Type.Results)
	if err != nil {
		return err
	}
	wantParameters := []string{"workspec.Spec", "[]Attempt", "[]GateStatus", "time.Time"}
	wantResults := []string{"Decision", "error"}
	if !equalStrings(parameters, wantParameters) || !equalStrings(results, wantResults) {
		return fmt.Errorf("transition.Decide signature changed: parameters=%v results=%v; want parameters=%v results=%v", parameters, results, wantParameters, wantResults)
	}
	return nil
}

func checkValidationFingerprintFields(repo string) error {
	path := filepath.Join(repo, "internal", "validationevidence", "model.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return err
	}
	want := []string{
		"Identity",
		"RepositoryID",
		"SubjectTreeOID",
		"Node",
		"NodeInputDigest",
		"Profile",
		"ValidationPlanDigest",
		"EngineSemanticDigest",
		"ConfigDigest",
		"ToolchainDigest",
		"OptionsDigest",
	}
	var matches [][]string
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Fingerprint" {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return fmt.Errorf("validationevidence.Fingerprint must remain a struct")
			}
			fields := make([]string, 0, len(structure.Fields.List))
			for _, field := range structure.Fields.List {
				for _, name := range field.Names {
					fields = append(fields, name.Name)
				}
			}
			matches = append(matches, fields)
		}
	}
	if len(matches) != 1 {
		return fmt.Errorf("validationevidence.Fingerprint must have exactly one declaration; found %d", len(matches))
	}
	if !equalStrings(matches[0], want) {
		return fmt.Errorf("validationevidence.Fingerprint field list changed: got %v; want %v", matches[0], want)
	}
	return nil
}

func checkKitManifestSourceOptional(repo string) error {
	path := filepath.Join(repo, "config", "schemas", "kit-manifest.schema.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return fmt.Errorf("parse kit manifest schema: %w", err)
	}
	if _, exists := schema.Properties["source"]; !exists {
		return fmt.Errorf("kit manifest schema must declare optional source property")
	}
	for _, required := range schema.Required {
		if required == "source" {
			return fmt.Errorf("kit manifest source must remain optional")
		}
	}
	return nil
}

func checkTypedSubcommandCatalog(repo string) error {
	cliDir := filepath.Join(repo, "internal", "cli")
	catalogPath := filepath.Join(cliDir, "catalog.go")
	catalogFile, err := parser.ParseFile(token.NewFileSet(), catalogPath, nil, 0)
	if err != nil {
		return err
	}
	handlers, err := catalogSubcommandHandlers(catalogFile)
	if err != nil {
		return err
	}
	if len(handlers) == 0 {
		return fmt.Errorf("typed command catalog registers no subcommand handlers")
	}

	files, err := filepath.Glob(filepath.Join(cliDir, "*.go"))
	if err != nil {
		return err
	}
	foundHandlers := make(map[string]bool, len(handlers))
	executeUsesGuard := false
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if function.Name.Name == "Execute" && callsSelector(function.Body, "commands", "prepareInvocation") {
				executeUsesGuard = true
				if literal, ok := directArgsStringComparison(function.Body); ok {
					return fmt.Errorf("cli.Execute contains catalog-external argv route %q", literal)
				}
			}
			if _, registered := handlers[function.Name.Name]; !registered {
				continue
			}
			foundHandlers[function.Name.Name] = true
			if !callsIdentifier(function.Body, "resolveCatalogSubcommandID") {
				return fmt.Errorf("%s routes registered subcommands without typed catalog resolution", function.Name.Name)
			}
			if literal, ok := stringCaseLiteral(function.Body); ok {
				return fmt.Errorf("%s contains catalog-external string subcommand route %q", function.Name.Name, literal)
			}
			if literal, ok := directArgsStringComparison(function.Body); ok {
				return fmt.Errorf("%s contains catalog-external args[0] route %q", function.Name.Name, literal)
			}
		}
	}
	if !executeUsesGuard {
		return fmt.Errorf("cli.Execute must route through typed catalog prepareInvocation")
	}
	for handler := range handlers {
		if !foundHandlers[handler] {
			return fmt.Errorf("typed command catalog references missing subcommand handler %s", handler)
		}
	}
	return nil
}

func catalogSubcommandHandlers(file *ast.File) (map[string]struct{}, error) {
	handlers := map[string]struct{}{}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "commands" || len(value.Values) != 1 {
				continue
			}
			call, ok := value.Values[0].(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return nil, fmt.Errorf("commands must be constructed by the typed catalog")
			}
			routes, ok := call.Args[0].(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("typed command routes must be a composite literal")
			}
			for _, element := range routes.Elts {
				route, ok := element.(*ast.CompositeLit)
				if !ok {
					continue
				}
				descriptor, _ := compositeField(route, "descriptor").(*ast.CompositeLit)
				handler, _ := compositeField(route, "handler").(*ast.Ident)
				if descriptor == nil || handler == nil {
					continue
				}
				subcommands, _ := compositeField(descriptor, "Subcommands").(*ast.CompositeLit)
				if subcommands != nil && len(subcommands.Elts) > 0 {
					handlers[handler.Name] = struct{}{}
				}
			}
			return handlers, nil
		}
	}
	return nil, fmt.Errorf("typed command catalog variable commands was not found")
}

func compositeField(literal *ast.CompositeLit, name string) ast.Expr {
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := pair.Key.(*ast.Ident)
		if ok && key.Name == name {
			return pair.Value
		}
	}
	return nil
}

func callsSelector(node ast.Node, receiver, method string) bool {
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		call, ok := current.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, receiverOK := selector.X.(*ast.Ident)
		if receiverOK && identifier.Name == receiver && selector.Sel.Name == method {
			found = true
			return false
		}
		return true
	})
	return found
}

func callsIdentifier(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		call, ok := current.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, functionOK := call.Fun.(*ast.Ident)
		if functionOK && identifier.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func stringCaseLiteral(node ast.Node) (string, bool) {
	var literal string
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		clause, ok := current.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, expression := range clause.List {
			basic, ok := expression.(*ast.BasicLit)
			if !ok || basic.Kind != token.STRING {
				continue
			}
			literal, _ = strconv.Unquote(basic.Value)
			found = true
			return false
		}
		return !found
	})
	return literal, found
}

func directArgsStringComparison(node ast.Node) (string, bool) {
	var literal string
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		binary, ok := current.(*ast.BinaryExpr)
		if !ok || (binary.Op != token.EQL && binary.Op != token.NEQ) {
			return true
		}
		left, leftString := stringLiteral(binary.X)
		right, rightString := stringLiteral(binary.Y)
		switch {
		case leftString && referencesArgumentZero(binary.Y):
			literal, found = left, true
		case rightString && referencesArgumentZero(binary.X):
			literal, found = right, true
		}
		return !found
	})
	return literal, found
}

func stringLiteral(expression ast.Expr) (string, bool) {
	basic, ok := expression.(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(basic.Value)
	return value, err == nil
}

func referencesArgumentZero(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(current ast.Node) bool {
		index, ok := current.(*ast.IndexExpr)
		if !ok {
			return true
		}
		name, nameOK := index.X.(*ast.Ident)
		value, valueOK := index.Index.(*ast.BasicLit)
		if nameOK && valueOK && (name.Name == "args" || name.Name == "commandArgs") && value.Kind == token.INT && value.Value == "0" {
			found = true
			return false
		}
		return true
	})
	return found
}

func checkProductProfileVocabulary(repo string) error {
	cliDir := filepath.Join(repo, "internal", "cli")
	files, err := filepath.Glob(filepath.Join(cliDir, "*.go"))
	if err != nil {
		return err
	}
	want := []string{"Smoke", "Full", "Release"}
	var got []string
	normalizerUsesCatalog := false
	profileFlags := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if ok && general.Tok == token.VAR {
				for _, specification := range general.Specs {
					value, ok := specification.(*ast.ValueSpec)
					if !ok || len(value.Names) != 1 || value.Names[0].Name != "productProfileVocabulary" || len(value.Values) != 1 {
						continue
					}
					literal, ok := value.Values[0].(*ast.CompositeLit)
					if !ok {
						return fmt.Errorf("product --profile vocabulary must be a literal")
					}
					for _, element := range literal.Elts {
						value, ok := stringLiteral(element)
						if !ok {
							return fmt.Errorf("product --profile vocabulary contains a non-string value")
						}
						got = append(got, value)
					}
				}
			}
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Body != nil {
				if function.Name.Name == "normalizeTestProfile" {
					ast.Inspect(function.Body, func(current ast.Node) bool {
						rangeStatement, ok := current.(*ast.RangeStmt)
						if !ok {
							return true
						}
						identifier, rangeOK := rangeStatement.X.(*ast.Ident)
						if rangeOK && identifier.Name == "productProfileVocabulary" {
							normalizerUsesCatalog = true
						}
						return true
					})
				}
				var flagError error
				ast.Inspect(function.Body, func(current ast.Node) bool {
					call, ok := current.(*ast.CallExpr)
					if !ok {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || (selector.Sel.Name != "String" && selector.Sel.Name != "StringVar") {
						return true
					}
					nameIndex := 0
					if selector.Sel.Name == "StringVar" {
						nameIndex = 1
					}
					if len(call.Args) <= nameIndex {
						return true
					}
					name, ok := stringLiteral(call.Args[nameIndex])
					if !ok || name != "profile" {
						return true
					}
					profileFlags++
					if !callsIdentifier(call, "productProfileHelp") {
						flagError = fmt.Errorf("%s declares --profile help outside the product vocabulary catalog", function.Name.Name)
						return false
					}
					return true
				})
				if flagError != nil {
					return flagError
				}
			}
		}
	}
	if !equalStrings(got, want) {
		return fmt.Errorf("product --profile vocabulary changed: got %v; want %v", got, want)
	}
	if profileFlags == 0 {
		return fmt.Errorf("no product --profile flags found")
	}
	if !normalizerUsesCatalog {
		return fmt.Errorf("normalizeTestProfile must validate against productProfileVocabulary")
	}
	return nil
}

func fieldTypes(fields *ast.FieldList) ([]string, error) {
	if fields == nil {
		return nil, nil
	}
	types := []string{}
	for _, field := range fields.List {
		var rendered bytes.Buffer
		if err := format.Node(&rendered, token.NewFileSet(), field.Type); err != nil {
			return nil, err
		}
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			types = append(types, rendered.String())
		}
	}
	return types, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
