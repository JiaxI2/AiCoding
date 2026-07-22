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
