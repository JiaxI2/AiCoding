package governance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

const dependencyGovernancePath = "config/dependency-governance.json"

type dependencyPolicy struct {
	SchemaVersion        int                   `json:"schemaVersion"`
	Name                 string                `json:"name"`
	Direction            string                `json:"direction"`
	Layers               []dependencyLayer     `json:"layers"`
	ReservedNamespaces   []reservedNamespace   `json:"reservedNamespaces"`
	Scan                 dependencyScan        `json:"scan"`
	VersionVisibility    versionVisibility     `json:"versionVisibility"`
	KitRegistry          dependencyRegistry    `json:"kitRegistry"`
	MCPRegistry          dependencyRegistry    `json:"mcpRegistry"`
	Skills               dependencySkillPolicy `json:"skills"`
	ExternalDependencies []externalDependency  `json:"externalDependencies"`
	GitProcessBoundary   gitProcessBoundary    `json:"gitProcessBoundary"`
	GoPackageBoundaries  []goPackageBoundary   `json:"goPackageBoundaries"`
}

type dependencyLayer struct {
	ID          string `json:"id"`
	Rank        int    `json:"rank"`
	Description string `json:"description"`
}

type reservedNamespace struct {
	Value      string `json:"value"`
	OwnerLayer string `json:"ownerLayer"`
}

type dependencyScan struct {
	Extensions         []string `json:"extensions"`
	FileNames          []string `json:"fileNames"`
	ExcludeDirectories []string `json:"excludeDirectories"`
}

type versionVisibility struct {
	IdentityPattern               string         `json:"identityPattern"`
	CodeSelfVersionPattern        string         `json:"codeSelfVersionPattern"`
	CodeSelfVersionAllowedSymbols []string       `json:"codeSelfVersionAllowedSymbols"`
	ReadmeBodyVersionPattern      string         `json:"readmeBodyVersionPattern"`
	CodeExtensions                []string       `json:"codeExtensions"`
	CodeFileNames                 []string       `json:"codeFileNames"`
	DocumentationDirectories      []string       `json:"documentationDirectories"`
	AuthorityFiles                []string       `json:"authorityFiles"`
	ReadmeFiles                   []string       `json:"readmeFiles"`
	ReadmeBadges                  []versionBadge `json:"readmeBadges"`
}

type versionBadge struct {
	Label          string `json:"label"`
	ImageFragment  string `json:"imageFragment"`
	Target         string `json:"target"`
	Authority      string `json:"authority"`
	DisplayVersion string `json:"displayVersion"`
	Manifest       string `json:"manifest"`
}

type dependencyRegistry struct {
	Path         string              `json:"path"`
	IDPattern    string              `json:"idPattern"`
	PromptPolicy string              `json:"promptPolicy"`
	Bindings     []dependencyBinding `json:"bindings"`
}

type dependencyBinding struct {
	ID               string               `json:"id"`
	Layer            string               `json:"layer"`
	PlatformAgnostic bool                 `json:"platformAgnostic"`
	Roots            []string             `json:"roots"`
	DependsOn        []string             `json:"dependsOn"`
	Exception        *dependencyException `json:"exception"`
}

type dependencyException struct {
	Reason   string `json:"reason"`
	Owner    string `json:"owner"`
	ReviewBy string `json:"reviewBy"`
}

type dependencySkillPolicy struct {
	RuntimeConfig               string   `json:"runtimeConfig"`
	PluginRoot                  string   `json:"pluginRoot"`
	PluginLayer                 string   `json:"pluginLayer"`
	PluginRequiredPrefix        string   `json:"pluginRequiredPrefix"`
	StandaloneLayer             string   `json:"standaloneLayer"`
	StandaloneForbiddenPrefixes []string `json:"standaloneForbiddenPrefixes"`
}

type externalDependency struct {
	ID    string `json:"id"`
	Layer string `json:"layer"`
}

type goPackageBoundary struct {
	Path             string   `json:"path"`
	ForbiddenImports []string `json:"forbiddenImports"`
}

type gitProcessBoundary struct {
	OwnerPackage     string   `json:"ownerPackage"`
	ScanRoots        []string `json:"scanRoots"`
	AllowedImporters []string `json:"allowedImporters"`
}

type dependencyRegistryFile struct {
	Kits       []dependencyRegistryEntry `json:"kits"`
	Components []dependencyRegistryEntry `json:"components"`
}

type dependencyRegistryEntry struct {
	ID       string `json:"id"`
	Manifest string `json:"manifest"`
}

type dependencyComponentManifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Runtime     struct {
		Module     string   `json:"module"`
		ServerArgs []string `json:"serverArgs"`
	} `json:"runtime"`
	Codex struct {
		ServerName string `json:"serverName"`
	} `json:"codex"`
}

type dependencyCodexKit struct {
	Profiles map[string]struct {
		StandaloneSkills []string `json:"standaloneSkills"`
	} `json:"profiles"`
	StandaloneSkillRegistry struct {
		Skills      []string          `json:"skills"`
		SourcePaths map[string]string `json:"sourcePaths"`
	} `json:"standaloneSkillRegistry"`
}

type DependencyReport struct {
	SchemaVersion int               `json:"schemaVersion"`
	Config        string            `json:"config"`
	Direction     string            `json:"direction"`
	Checks        []DependencyCheck `json:"checks"`
	Errors        []string          `json:"errors"`
	Warnings      []string          `json:"warnings,omitempty"`
}

type DependencyCheck struct {
	Name     string   `json:"name"`
	OK       bool     `json:"ok"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// CheckDependencies validates layer direction, registry classification,
// reserved namespaces and lower-layer platform independence.
func CheckDependencies(repo string) DependencyReport {
	report := DependencyReport{
		SchemaVersion: 1,
		Config:        dependencyGovernancePath,
		Checks:        []DependencyCheck{},
		Errors:        []string{},
	}
	policy, err := loadDependencyPolicy(repo)
	if err != nil {
		report.addDependencyCheck("load dependency policy", []string{err.Error()}, nil)
		return report
	}
	report.Direction = policy.Direction

	layers, layerErrors := dependencyLayers(policy)
	report.addDependencyCheck("layer model", layerErrors, nil)

	bindings, bindingErrors := dependencyBindings(policy, layers)
	report.addDependencyCheck("binding declarations", bindingErrors, nil)

	kitEntries, kitRegistryErrors := loadDependencyRegistry(repo, policy.KitRegistry.Path, "kits")
	kitCoverageErrors := append(kitRegistryErrors, dependencyRegistryCoverage("kit", kitEntries, policy.KitRegistry.Bindings)...)
	report.addDependencyCheck("kit registry coverage", kitCoverageErrors, nil)

	mcpEntries, mcpRegistryErrors := loadDependencyRegistry(repo, policy.MCPRegistry.Path, "components")
	mcpCoverageErrors := append(mcpRegistryErrors, dependencyRegistryCoverage("MCP", mcpEntries, policy.MCPRegistry.Bindings)...)
	if policy.MCPRegistry.IDPattern != "" {
		idPattern, compileErr := regexp.Compile(policy.MCPRegistry.IDPattern)
		if compileErr != nil {
			mcpCoverageErrors = append(mcpCoverageErrors, "invalid MCP idPattern: "+compileErr.Error())
		} else {
			for _, entry := range mcpEntries {
				if !idPattern.MatchString(entry.ID) {
					mcpCoverageErrors = append(mcpCoverageErrors, "MCP component id does not match policy: "+entry.ID)
				}
			}
		}
	}
	report.addDependencyCheck("MCP registry coverage", mcpCoverageErrors, nil)

	report.addDependencyCheck("declared dependency direction", checkDependencyDirection(bindings, layers), nil)
	report.addDependencyCheck("lower-layer platform independence", checkPlatformAgnosticRoots(repo, policy, layers), nil)
	report.addDependencyCheck("MCP component identity", checkMCPComponentIdentity(repo, policy, mcpEntries), nil)
	report.addDependencyCheck("MCP and Skill responsibility boundary", checkMCPPromptPolicy(repo, policy), nil)
	report.addDependencyCheck("Skill naming and exposure", checkSkillDependencyPolicy(repo, policy, layers), nil)
	report.addDependencyCheck("asset identity version opacity", checkAssetVersionOpacity(repo, policy), nil)
	report.addDependencyCheck("README version badge authority", checkReadmeVersionBadges(repo, policy), nil)
	report.addDependencyCheck("orthogonal Go package boundaries", checkGoPackageBoundaries(repo, policy.GoPackageBoundaries), nil)
	report.addDependencyCheck("git process ownership", checkGitProcessOwnership(repo, policy.GitProcessBoundary), nil)
	report.addDependencyCheck("gitx importer allowlist", checkGitxImporterAllowlist(repo, policy.GitProcessBoundary), nil)

	for _, rel := range []string{
		"config/schemas/dependency-governance.schema.json",
		policy.KitRegistry.Path,
		policy.MCPRegistry.Path,
		policy.Skills.RuntimeConfig,
	} {
		if !platform.IsFile(platform.RepoPath(repo, rel)) {
			report.addDependencyCheck("required governance files", []string{"missing " + rel}, nil)
			break
		}
	}
	return report
}

func checkGoPackageBoundaries(repo string, boundaries []goPackageBoundary) []string {
	errs := []string{}
	for _, boundary := range boundaries {
		rel := filepath.ToSlash(filepath.Clean(boundary.Path))
		if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || !strings.HasPrefix(rel, "internal/") {
			errs = append(errs, "Go package boundary path must stay under internal/: "+boundary.Path)
			continue
		}
		root := platform.RepoPath(repo, rel)
		if !platform.IsDir(root) {
			errs = append(errs, "Go package boundary directory is missing: "+rel)
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if parseErr != nil {
				return parseErr
			}
			for _, imported := range file.Imports {
				value, unquoteErr := strconv.Unquote(imported.Path.Value)
				if unquoteErr != nil {
					return unquoteErr
				}
				for _, forbidden := range boundary.ForbiddenImports {
					prefix := "github.com/JiaxI2/AiCoding/" + strings.TrimSuffix(filepath.ToSlash(forbidden), "/")
					if value == prefix || strings.HasPrefix(value, prefix+"/") {
						relFile, _ := filepath.Rel(repo, path)
						errs = append(errs, filepath.ToSlash(relFile)+" imports forbidden package "+value)
					}
				}
			}
			return nil
		})
		if walkErr != nil {
			errs = append(errs, "cannot inspect Go package boundary "+rel+": "+walkErr.Error())
		}
	}
	return errs
}

func checkGitProcessOwnership(repo string, boundary gitProcessBoundary) []string {
	if errs := validateGitProcessBoundary(boundary); len(errs) != 0 {
		return errs
	}
	return inspectGitBoundaryFiles(repo, boundary, true, func(path, relFile string) []string {
		startsGit, err := startsLiteralGitProcess(path)
		if err != nil {
			return []string{relFile + ": " + err.Error()}
		}
		if startsGit {
			return []string{relFile + " starts git process outside " + boundary.OwnerPackage}
		}
		return nil
	})
}

func checkGitxImporterAllowlist(repo string, boundary gitProcessBoundary) []string {
	if errs := validateGitProcessBoundary(boundary); len(errs) != 0 {
		return errs
	}
	allowed := make(map[string]struct{}, len(boundary.AllowedImporters))
	for _, importer := range boundary.AllowedImporters {
		allowed[filepath.ToSlash(filepath.Clean(importer))] = struct{}{}
	}
	return inspectGitBoundaryFiles(repo, boundary, false, func(path, relFile string) []string {
		importsGitx, err := importsInternalGitx(path)
		if err != nil {
			return []string{relFile + ": " + err.Error()}
		}
		if !importsGitx {
			return nil
		}
		importer := filepath.ToSlash(filepath.Dir(relFile))
		if _, ok := allowed[importer]; !ok {
			return []string{relFile + " imports internal/gitx from non-allowlisted package " + importer}
		}
		return nil
	})
}

func validateGitProcessBoundary(boundary gitProcessBoundary) []string {
	if strings.TrimSpace(boundary.OwnerPackage) == "" || len(boundary.ScanRoots) == 0 || len(boundary.AllowedImporters) == 0 {
		return []string{"gitProcessBoundary policy is missing or incomplete"}
	}
	owner := filepath.ToSlash(filepath.Clean(boundary.OwnerPackage))
	if owner == "." || strings.HasPrefix(owner, "../") || !strings.HasPrefix(owner, "internal/") {
		return []string{"gitProcessBoundary ownerPackage must stay under internal/: " + boundary.OwnerPackage}
	}
	return nil
}

func inspectGitBoundaryFiles(repo string, boundary gitProcessBoundary, skipOwner bool, inspect func(string, string) []string) []string {
	errs := []string{}
	owner := filepath.ToSlash(filepath.Clean(boundary.OwnerPackage))
	for _, scanRoot := range boundary.ScanRoots {
		root := filepath.ToSlash(filepath.Clean(scanRoot))
		if root == "." || filepath.IsAbs(scanRoot) || strings.HasPrefix(root, "../") {
			errs = append(errs, "gitProcessBoundary scan root must stay inside the repository: "+scanRoot)
			continue
		}
		absoluteRoot := platform.RepoPath(repo, root)
		if !platform.IsDir(absoluteRoot) {
			errs = append(errs, "gitProcessBoundary scan root is missing: "+root)
			continue
		}
		walkErr := filepath.WalkDir(absoluteRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relFile, relErr := filepath.Rel(repo, path)
			if relErr != nil {
				return relErr
			}
			relFile = filepath.ToSlash(relFile)
			if entry.IsDir() {
				if skipOwner && relFile == owner {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			errs = append(errs, inspect(path, relFile)...)
			return nil
		})
		if walkErr != nil {
			errs = append(errs, "cannot inspect git process boundary "+root+": "+walkErr.Error())
		}
	}
	return errs
}

func startsLiteralGitProcess(path string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return false, err
	}
	execNames := map[string]struct{}{}
	dotImport := false
	for _, imported := range file.Imports {
		value, unquoteErr := strconv.Unquote(imported.Path.Value)
		if unquoteErr != nil {
			return false, unquoteErr
		}
		if value != "os/exec" {
			continue
		}
		if imported.Name == nil {
			execNames["exec"] = struct{}{}
			continue
		}
		switch imported.Name.Name {
		case ".":
			dotImport = true
		case "_":
		default:
			execNames[imported.Name.Name] = struct{}{}
		}
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		argument := -1
		switch function := call.Fun.(type) {
		case *ast.SelectorExpr:
			identifier, ok := function.X.(*ast.Ident)
			if !ok {
				return true
			}
			if _, ok := execNames[identifier.Name]; !ok {
				return true
			}
			switch function.Sel.Name {
			case "Command":
				argument = 0
			case "CommandContext":
				argument = 1
			}
		case *ast.Ident:
			if !dotImport {
				return true
			}
			switch function.Name {
			case "Command":
				argument = 0
			case "CommandContext":
				argument = 1
			}
		}
		if argument < 0 || len(call.Args) <= argument {
			return true
		}
		literal, ok := call.Args[argument].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, unquoteErr := strconv.Unquote(literal.Value)
		if unquoteErr == nil && value == "git" {
			found = true
		}
		return !found
	})
	return found, nil
}

func importsInternalGitx(path string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return false, err
	}
	const gitxImport = "github.com/JiaxI2/AiCoding/internal/gitx"
	for _, imported := range file.Imports {
		value, unquoteErr := strconv.Unquote(imported.Path.Value)
		if unquoteErr != nil {
			return false, unquoteErr
		}
		if value == gitxImport || strings.HasPrefix(value, gitxImport+"/") {
			return true, nil
		}
	}
	return false, nil
}

func (r *DependencyReport) addDependencyCheck(name string, errs, warnings []string) {
	errs = uniqueLayoutErrors(errs)
	warnings = uniqueLayoutErrors(warnings)
	sort.Strings(errs)
	sort.Strings(warnings)
	r.Checks = append(r.Checks, DependencyCheck{Name: name, OK: len(errs) == 0, Errors: errs, Warnings: warnings})
	for _, err := range errs {
		r.Errors = append(r.Errors, name+": "+err)
	}
	for _, warning := range warnings {
		r.Warnings = append(r.Warnings, name+": "+warning)
	}
}

func loadDependencyPolicy(repo string) (dependencyPolicy, error) {
	var policy dependencyPolicy
	data, err := os.ReadFile(platform.RepoPath(repo, dependencyGovernancePath))
	if err != nil {
		return policy, err
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		return policy, err
	}
	if policy.SchemaVersion != 1 {
		return policy, fmt.Errorf("dependency policy schemaVersion must be 1")
	}
	if policy.Direction != "higher-rank-may-depend-on-equal-or-lower-rank" {
		return policy, fmt.Errorf("unsupported dependency direction: %s", policy.Direction)
	}
	return policy, nil
}

func dependencyLayers(policy dependencyPolicy) (map[string]int, []string) {
	layers := map[string]int{}
	ranks := map[int]string{}
	errs := []string{}
	for _, layer := range policy.Layers {
		if layer.ID == "" {
			errs = append(errs, "layer id is required")
			continue
		}
		if _, ok := layers[layer.ID]; ok {
			errs = append(errs, "duplicate layer id: "+layer.ID)
		}
		if previous, ok := ranks[layer.Rank]; ok {
			errs = append(errs, fmt.Sprintf("duplicate layer rank %d: %s and %s", layer.Rank, previous, layer.ID))
		}
		layers[layer.ID] = layer.Rank
		ranks[layer.Rank] = layer.ID
	}
	for _, reserved := range policy.ReservedNamespaces {
		if _, ok := layers[reserved.OwnerLayer]; !ok {
			errs = append(errs, "reserved namespace references unknown owner layer: "+reserved.OwnerLayer)
		}
	}
	return layers, errs
}

func dependencyBindings(policy dependencyPolicy, layers map[string]int) (map[string]dependencyBinding, []string) {
	bindings := map[string]dependencyBinding{}
	errs := []string{}
	add := func(kind string, binding dependencyBinding) {
		key := kind + ":" + binding.ID
		if _, ok := bindings[key]; ok {
			errs = append(errs, "duplicate binding: "+key)
		}
		if _, ok := layers[binding.Layer]; !ok {
			errs = append(errs, key+" references unknown layer "+binding.Layer)
		}
		if binding.PlatformAgnostic && binding.Exception != nil {
			errs = append(errs, key+" cannot be platform agnostic and declare an exception")
		}
		if binding.Layer == "capability" && !binding.PlatformAgnostic {
			if binding.Exception == nil {
				errs = append(errs, key+" capability exception requires reason, owner and reviewBy")
			} else {
				if binding.Exception.Reason == "" || binding.Exception.Owner == "" || binding.Exception.ReviewBy == "" {
					errs = append(errs, key+" capability exception is incomplete")
				} else if reviewBy, err := time.Parse("2006-01-02", binding.Exception.ReviewBy); err != nil {
					errs = append(errs, key+" exception reviewBy is invalid")
				} else if time.Now().After(reviewBy.Add(24 * time.Hour)) {
					errs = append(errs, key+" capability exception review is overdue: "+binding.Exception.ReviewBy)
				}
			}
		}
		bindings[key] = binding
	}
	for _, binding := range policy.KitRegistry.Bindings {
		add("kit", binding)
	}
	for _, binding := range policy.MCPRegistry.Bindings {
		add("mcp", binding)
	}
	for _, external := range policy.ExternalDependencies {
		if _, ok := layers[external.Layer]; !ok {
			errs = append(errs, external.ID+" references unknown layer "+external.Layer)
			continue
		}
		if _, ok := bindings[external.ID]; ok {
			errs = append(errs, "duplicate external dependency: "+external.ID)
			continue
		}
		bindings[external.ID] = dependencyBinding{ID: external.ID, Layer: external.Layer, PlatformAgnostic: true}
	}
	return bindings, errs
}

func loadDependencyRegistry(repo, rel, field string) ([]dependencyRegistryEntry, []string) {
	data, err := os.ReadFile(platform.RepoPath(repo, rel))
	if err != nil {
		return nil, []string{err.Error()}
	}
	var registry dependencyRegistryFile
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, []string{err.Error()}
	}
	if field == "kits" {
		return registry.Kits, nil
	}
	return registry.Components, nil
}

func dependencyRegistryCoverage(kind string, entries []dependencyRegistryEntry, bindings []dependencyBinding) []string {
	entryIDs := map[string]bool{}
	bindingIDs := map[string]bool{}
	errs := []string{}
	for _, entry := range entries {
		if entryIDs[entry.ID] {
			errs = append(errs, "duplicate "+kind+" registry id: "+entry.ID)
		}
		entryIDs[entry.ID] = true
	}
	for _, binding := range bindings {
		if bindingIDs[binding.ID] {
			errs = append(errs, "duplicate "+kind+" policy binding: "+binding.ID)
		}
		bindingIDs[binding.ID] = true
	}
	for id := range entryIDs {
		if !bindingIDs[id] {
			errs = append(errs, kind+" registry entry lacks dependency binding: "+id)
		}
	}
	for id := range bindingIDs {
		if !entryIDs[id] {
			errs = append(errs, kind+" dependency binding is not registered: "+id)
		}
	}
	return errs
}

func checkDependencyDirection(bindings map[string]dependencyBinding, layers map[string]int) []string {
	errs := []string{}
	for sourceID, source := range bindings {
		sourceRank, ok := layers[source.Layer]
		if !ok {
			continue
		}
		for _, targetID := range source.DependsOn {
			target, exists := bindings[targetID]
			if !exists {
				errs = append(errs, sourceID+" depends on undeclared component "+targetID)
				continue
			}
			targetRank, ok := layers[target.Layer]
			if ok && sourceRank < targetRank {
				errs = append(errs, fmt.Sprintf("%s (%s) must not depend on higher layer %s (%s)", sourceID, source.Layer, targetID, target.Layer))
			}
		}
	}
	return errs
}

func checkPlatformAgnosticRoots(repo string, policy dependencyPolicy, layers map[string]int) []string {
	errs := []string{}
	for kind, bindings := range map[string][]dependencyBinding{
		"kit": policy.KitRegistry.Bindings,
		"mcp": policy.MCPRegistry.Bindings,
	} {
		for _, binding := range bindings {
			if !binding.PlatformAgnostic {
				continue
			}
			for _, reserved := range policy.ReservedNamespaces {
				if layers[binding.Layer] < layers[reserved.OwnerLayer] && strings.Contains(strings.ToLower(binding.ID), strings.ToLower(reserved.Value)) {
					errs = append(errs, kind+":"+binding.ID+" uses upper-layer namespace "+reserved.Value)
				}
			}
			for _, root := range binding.Roots {
				errs = append(errs, scanPlatformAgnosticRoot(repo, kind+":"+binding.ID, root, policy)...)
			}
		}
	}
	return errs
}

func scanPlatformAgnosticRoot(repo, bindingID, root string, policy dependencyPolicy) []string {
	rootPath := platform.RepoPath(repo, root)
	if !platform.IsDir(rootPath) {
		return []string{bindingID + " root is missing: " + root}
	}
	excluded := stringSet(policy.Scan.ExcludeDirectories)
	extensions := stringSetLower(policy.Scan.Extensions)
	fileNames := stringSet(policy.Scan.FileNames)
	errs := []string{}
	walkErr := filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != rootPath && (excluded[entry.Name()] || strings.HasSuffix(strings.ToLower(entry.Name()), ".egg-info")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !extensions[strings.ToLower(filepath.Ext(entry.Name()))] && !fileNames[entry.Name()] {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		line := 0
		for scanner.Scan() {
			line++
			text := scanner.Text()
			lower := strings.ToLower(text)
			for _, reserved := range policy.ReservedNamespaces {
				if strings.Contains(lower, strings.ToLower(reserved.Value)) {
					rel, _ := filepath.Rel(repo, path)
					errs = append(errs, fmt.Sprintf("%s contains upper-layer namespace %q at %s:%d", bindingID, reserved.Value, filepath.ToSlash(rel), line))
					break
				}
			}
		}
		return scanner.Err()
	})
	if walkErr != nil {
		errs = append(errs, bindingID+" scan failed: "+walkErr.Error())
	}
	return errs
}

func checkMCPComponentIdentity(repo string, policy dependencyPolicy, entries []dependencyRegistryEntry) []string {
	bindings := map[string]dependencyBinding{}
	for _, binding := range policy.MCPRegistry.Bindings {
		bindings[binding.ID] = binding
	}
	errs := []string{}
	for _, entry := range entries {
		binding, ok := bindings[entry.ID]
		if !ok || !binding.PlatformAgnostic {
			continue
		}
		data, err := os.ReadFile(platform.RepoPath(repo, entry.Manifest))
		if err != nil {
			errs = append(errs, entry.ID+": "+err.Error())
			continue
		}
		var manifest dependencyComponentManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			errs = append(errs, entry.ID+": "+err.Error())
			continue
		}
		identity := strings.Join([]string{
			manifest.ID,
			manifest.Name,
			manifest.Description,
			manifest.Runtime.Module,
			strings.Join(manifest.Runtime.ServerArgs, " "),
			manifest.Codex.ServerName,
		}, "\n")
		for _, reserved := range policy.ReservedNamespaces {
			if strings.Contains(strings.ToLower(identity), strings.ToLower(reserved.Value)) {
				errs = append(errs, entry.ID+" component identity contains upper-layer namespace "+reserved.Value)
			}
		}
	}
	return errs
}

func checkMCPPromptPolicy(repo string, policy dependencyPolicy) []string {
	if policy.MCPRegistry.PromptPolicy != "forbid-workflow-prompts" {
		return []string{"unsupported MCP prompt policy: " + policy.MCPRegistry.PromptPolicy}
	}
	errs := []string{}
	excluded := stringSet(policy.Scan.ExcludeDirectories)
	for _, binding := range policy.MCPRegistry.Bindings {
		if !binding.PlatformAgnostic {
			continue
		}
		for _, root := range binding.Roots {
			rootPath := platform.RepoPath(repo, root)
			walkErr := filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					if path != rootPath && (excluded[entry.Name()] || strings.HasSuffix(strings.ToLower(entry.Name()), ".egg-info")) {
						return filepath.SkipDir
					}
					if path != rootPath && entry.Name() == "prompts" {
						entries, readErr := os.ReadDir(path)
						if readErr != nil {
							return readErr
						}
						if len(entries) > 0 {
							rel, _ := filepath.Rel(repo, path)
							errs = append(errs, "capability MCP must not own workflow prompt directory: "+filepath.ToSlash(rel))
						}
						return filepath.SkipDir
					}
					return nil
				}
				if strings.ToLower(filepath.Ext(entry.Name())) != ".py" {
					return nil
				}
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				text := string(data)
				if strings.Contains(text, "@server.prompt") || strings.Contains(text, ".prompt(") {
					rel, _ := filepath.Rel(repo, path)
					errs = append(errs, "capability MCP must not register workflow prompts: "+filepath.ToSlash(rel))
				}
				return nil
			})
			if walkErr != nil {
				errs = append(errs, binding.ID+" prompt scan failed: "+walkErr.Error())
			}
		}
	}
	return errs
}

func checkSkillDependencyPolicy(repo string, policy dependencyPolicy, layers map[string]int) []string {
	errs := []string{}
	if _, ok := layers[policy.Skills.PluginLayer]; !ok {
		errs = append(errs, "plugin Skill layer is unknown: "+policy.Skills.PluginLayer)
	}
	if _, ok := layers[policy.Skills.StandaloneLayer]; !ok {
		errs = append(errs, "standalone Skill layer is unknown: "+policy.Skills.StandaloneLayer)
	}
	data, err := os.ReadFile(platform.RepoPath(repo, policy.Skills.RuntimeConfig))
	if err != nil {
		return append(errs, err.Error())
	}
	var config dependencyCodexKit
	if err := json.Unmarshal(data, &config); err != nil {
		return append(errs, err.Error())
	}
	registered := map[string]bool{}
	for _, name := range config.StandaloneSkillRegistry.Skills {
		registered[name] = true
		for _, prefix := range policy.Skills.StandaloneForbiddenPrefixes {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				errs = append(errs, "standalone Skill must not use platform prefix: "+name)
			}
		}
	}
	for profile, profileConfig := range config.Profiles {
		for _, name := range profileConfig.StandaloneSkills {
			if !registered[name] {
				errs = append(errs, "profile "+profile+" references unregistered standalone Skill: "+name)
			}
		}
	}
	for name := range config.StandaloneSkillRegistry.SourcePaths {
		if !registered[name] {
			errs = append(errs, "standalone Skill sourcePaths has unregistered key: "+name)
		}
	}

	pluginRoot := platform.RepoPath(repo, policy.Skills.PluginRoot)
	if platform.IsDir(pluginRoot) {
		entries, readErr := os.ReadDir(pluginRoot)
		if readErr != nil {
			errs = append(errs, readErr.Error())
		} else {
			for _, entry := range entries {
				if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				if !strings.HasPrefix(strings.ToLower(entry.Name()), strings.ToLower(policy.Skills.PluginRequiredPrefix)) {
					errs = append(errs, "plugin Skill must use platform prefix: "+entry.Name())
				}
			}
		}
	}
	return errs
}

func checkAssetVersionOpacity(repo string, policy dependencyPolicy) []string {
	identityPattern, err := regexp.Compile(policy.VersionVisibility.IdentityPattern)
	if err != nil {
		return []string{"invalid version identity pattern: " + err.Error()}
	}
	selfVersionPattern, err := regexp.Compile(policy.VersionVisibility.CodeSelfVersionPattern)
	if err != nil {
		return []string{"invalid code self-version pattern: " + err.Error()}
	}
	errs := []string{}
	for kind, bindings := range map[string][]dependencyBinding{
		"kit": policy.KitRegistry.Bindings,
		"mcp": policy.MCPRegistry.Bindings,
	} {
		for _, binding := range bindings {
			if identityPattern.MatchString(binding.ID) {
				errs = append(errs, kind+":"+binding.ID+" encodes a version in its stable id")
			}
			for _, root := range binding.Roots {
				errs = append(errs, scanAssetVersionOpacity(repo, kind+":"+binding.ID, root, policy, identityPattern, selfVersionPattern)...)
			}
		}
	}
	data, readErr := os.ReadFile(platform.RepoPath(repo, policy.Skills.RuntimeConfig))
	if readErr == nil {
		var config dependencyCodexKit
		if json.Unmarshal(data, &config) == nil {
			for _, name := range config.StandaloneSkillRegistry.Skills {
				if identityPattern.MatchString(name) {
					errs = append(errs, "standalone Skill encodes a version in its stable name: "+name)
				}
			}
		}
	}
	return errs
}

func scanAssetVersionOpacity(repo, bindingID, root string, policy dependencyPolicy, identityPattern, selfVersionPattern *regexp.Regexp) []string {
	rootPath := platform.RepoPath(repo, root)
	if !platform.IsDir(rootPath) {
		return []string{bindingID + " root is missing: " + root}
	}
	excluded := stringSet(policy.Scan.ExcludeDirectories)
	codeExtensions := stringSetLower(policy.VersionVisibility.CodeExtensions)
	codeFileNames := stringSet(policy.VersionVisibility.CodeFileNames)
	documentationDirectories := stringSet(policy.VersionVisibility.DocumentationDirectories)
	authorityFiles := stringSet(policy.VersionVisibility.AuthorityFiles)
	errs := []string{}
	walkErr := filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != rootPath && (excluded[entry.Name()] || strings.HasSuffix(strings.ToLower(entry.Name()), ".egg-info")) {
				return filepath.SkipDir
			}
			return nil
		}
		relFromRoot, _ := filepath.Rel(rootPath, path)
		segments := strings.Split(filepath.ToSlash(relFromRoot), "/")
		inDocumentation := false
		for index, segment := range segments {
			if index == len(segments)-1 {
				break
			}
			if documentationDirectories[segment] {
				inDocumentation = true
				break
			}
			if identityPattern.MatchString(segment) {
				rel, _ := filepath.Rel(repo, path)
				errs = append(errs, bindingID+" path encodes a version in stable identity: "+filepath.ToSlash(rel))
				break
			}
		}
		if !inDocumentation && !authorityFiles[entry.Name()] && identityPattern.MatchString(entry.Name()) {
			rel, _ := filepath.Rel(repo, path)
			errs = append(errs, bindingID+" file name encodes a version in stable identity: "+filepath.ToSlash(rel))
		}
		if !codeExtensions[strings.ToLower(filepath.Ext(entry.Name()))] && !codeFileNames[entry.Name()] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		line := 0
		for scanner.Scan() {
			line++
			text := scanner.Text()
			allowedSelfVersion := false
			lowerText := strings.ToLower(text)
			for _, symbol := range policy.VersionVisibility.CodeSelfVersionAllowedSymbols {
				if strings.Contains(lowerText, strings.ToLower(symbol)) {
					allowedSelfVersion = true
					break
				}
			}
			if identityPattern.MatchString(text) || (selfVersionPattern.MatchString(text) && !allowedSelfVersion) {
				rel, _ := filepath.Rel(repo, path)
				errs = append(errs, fmt.Sprintf("%s code observes an asset version at %s:%d", bindingID, filepath.ToSlash(rel), line))
			}
		}
		return scanner.Err()
	})
	if walkErr != nil {
		errs = append(errs, bindingID+" version scan failed: "+walkErr.Error())
	}
	readme := filepath.Join(rootPath, "README.md")
	if data, readErr := os.ReadFile(readme); readErr == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "# ") {
				if identityPattern.MatchString(line) {
					rel, _ := filepath.Rel(repo, readme)
					errs = append(errs, bindingID+" README title encodes an asset version: "+filepath.ToSlash(rel))
				}
				break
			}
		}
	}
	return errs
}

func checkReadmeVersionBadges(repo string, policy dependencyPolicy) []string {
	bodyVersionPattern, err := regexp.Compile(policy.VersionVisibility.ReadmeBodyVersionPattern)
	if err != nil {
		return []string{"invalid README body version pattern: " + err.Error()}
	}
	errs := []string{}
	var baseline []string
	for _, rel := range policy.VersionVisibility.ReadmeFiles {
		data, readErr := os.ReadFile(platform.RepoPath(repo, rel))
		if readErr != nil {
			errs = append(errs, readErr.Error())
			continue
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		badgeLines := []string{}
		for index, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "[![") {
				badgeLines = append(badgeLines, trimmed)
				continue
			}
			if bodyVersionPattern.MatchString(line) {
				errs = append(errs, fmt.Sprintf("%s exposes a version outside a badge at line %d", rel, index+1))
			}
		}
		if baseline == nil {
			baseline = badgeLines
		} else if strings.Join(baseline, "\n") != strings.Join(badgeLines, "\n") {
			errs = append(errs, rel+" badge block differs from "+policy.VersionVisibility.ReadmeFiles[0])
		}
		for _, badge := range policy.VersionVisibility.ReadmeBadges {
			line := findBadgeLine(badgeLines, badge.Label)
			if line == "" {
				errs = append(errs, rel+" is missing version badge "+badge.Label)
				continue
			}
			if !strings.Contains(line, badge.ImageFragment) {
				errs = append(errs, rel+" badge image does not match policy: "+badge.Label)
			}
			if !strings.Contains(line, "]("+badge.Target+")") {
				errs = append(errs, rel+" badge target does not match version authority: "+badge.Label)
			}
		}
	}
	for _, badge := range policy.VersionVisibility.ReadmeBadges {
		switch badge.Authority {
		case "local-kit":
			if badge.Manifest == "" {
				errs = append(errs, badge.Label+" local-kit badge requires a manifest")
				continue
			}
			data, readErr := os.ReadFile(platform.RepoPath(repo, badge.Manifest))
			if readErr != nil {
				errs = append(errs, readErr.Error())
				continue
			}
			var manifest struct {
				Version string `json:"version"`
			}
			if json.Unmarshal(data, &manifest) != nil || manifest.Version == "" {
				errs = append(errs, badge.Label+" manifest version is missing")
				continue
			}
			if badge.DisplayVersion != manifest.Version {
				errs = append(errs, badge.Label+" badge version does not match "+badge.Manifest)
			}
			if !platform.IsFile(platform.RepoPath(repo, badge.Target)) {
				errs = append(errs, badge.Label+" local documentation target is missing: "+badge.Target)
			}
		case "upstream-version":
			anchor := strings.TrimSuffix(badge.DisplayVersion, "+")
			if anchor == "" || !strings.Contains(badge.Target, anchor) {
				errs = append(errs, badge.Label+" upstream target is not bound to displayed version "+badge.DisplayVersion)
			}
			if !strings.HasPrefix(badge.Target, "https://") {
				errs = append(errs, badge.Label+" upstream version target must use HTTPS")
			}
		case "upstream-project", "repository-release":
			if !strings.HasPrefix(badge.Target, "https://") {
				errs = append(errs, badge.Label+" badge target must use HTTPS")
			}
		default:
			errs = append(errs, badge.Label+" uses unsupported badge authority "+badge.Authority)
		}
	}
	return errs
}

func findBadgeLine(lines []string, label string) string {
	needle := "[![" + label + "]"
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func stringSetLower(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[strings.ToLower(value)] = true
	}
	return result
}
