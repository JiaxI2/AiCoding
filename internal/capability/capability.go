package capability

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/registry"
)

const (
	CatalogPath       = "config/internal-capabilities.json"
	CapabilitiesPath  = "docs/CAPABILITIES.md"
	readmePath        = "README.md"
	readmeBeginMarker = "<!-- BEGIN GENERATED: CAPABILITIES -->"
	readmeEndMarker   = "<!-- END GENERATED: CAPABILITIES -->"
)

var (
	capabilityIDPattern      = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	capabilityPackagePattern = regexp.MustCompile(`^[a-z0-9]+$`)
	readInternalDirectory    = os.ReadDir
)

type Capability struct {
	ID              string      `json:"id"`
	Package         string      `json:"package"`
	Name            string      `json:"name"`
	Type            string      `json:"type"`
	Status          string      `json:"status"`
	Summary         string      `json:"summary"`
	PublicEntries   []string    `json:"publicEntries"`
	ArchitectureDoc string      `json:"architectureDoc,omitempty"`
	Quickstart      *Quickstart `json:"quickstart,omitempty"`
	Activation      *Activation `json:"activation,omitempty"`
	Verification    []string    `json:"verification,omitempty"`
}

type Quickstart struct {
	Steps        []string `json:"steps"`
	ExampleInput string   `json:"exampleInput,omitempty"`
}

type Activation struct {
	Kind       string `json:"kind"`
	Note       string `json:"note"`
	AgentUsage string `json:"agentUsage"`
}

type Catalog struct {
	SchemaVersion int          `json:"schemaVersion"`
	Name          string       `json:"name"`
	Digest        string       `json:"digest"`
	Capabilities  []Capability `json:"capabilities"`
}

type VerifyOptions struct {
	PublicEntryExists func(string) bool `json:"-"`
	CheckGenerated    bool              `json:"checkGenerated"`
}

type Verification struct {
	OK                            bool     `json:"ok"`
	RegistryPath                  string   `json:"registryPath"`
	Digest                        string   `json:"digest"`
	RegisteredCount               int      `json:"registeredCount"`
	InternalDirectoryCount        int      `json:"internalDirectoryCount"`
	Unregistered                  []string `json:"unregistered"`
	MissingPackages               []string `json:"missingPackages"`
	MissingDocuments              []string `json:"missingDocuments"`
	InvalidPublicEntries          []string `json:"invalidPublicEntries"`
	StableWithoutVerification     []string `json:"stableWithoutVerification"`
	StablePublicWithoutQuickstart []string `json:"stablePublicWithoutQuickstart"`
	StablePublicWithoutActivation []string `json:"stablePublicWithoutActivation"`
	GeneratedChecked              bool     `json:"generatedChecked"`
	READMEUpToDate                bool     `json:"readmeUpToDate"`
	DocumentUpToDate              bool     `json:"documentUpToDate"`
	Errors                        []string `json:"errors"`
}

type IndexRender struct {
	README   string `json:"readme"`
	Document string `json:"document"`
}

type catalogFile struct {
	SchemaVersion int          `json:"schemaVersion"`
	Name          string       `json:"name"`
	Capabilities  []Capability `json:"capabilities"`
}

func Load(repo string) (Catalog, error) {
	path := filepath.Join(repo, filepath.FromSlash(CatalogPath))
	file, err := os.Open(path)
	if err != nil {
		return Catalog{}, err
	}
	defer file.Close()

	var source catalogFile
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		return Catalog{}, fmt.Errorf("decode %s: %w", CatalogPath, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Catalog{}, fmt.Errorf("decode %s: %w", CatalogPath, err)
	}
	if err := validateCatalog(source); err != nil {
		return Catalog{}, err
	}
	snapshot, err := registry.NewSnapshot("internal-capabilities", source)
	if err != nil {
		return Catalog{}, err
	}
	catalog := Catalog{
		SchemaVersion: source.SchemaVersion,
		Name:          source.Name,
		Digest:        snapshot.Digest(),
		Capabilities:  cloneCapabilities(source.Capabilities),
	}
	sort.Slice(catalog.Capabilities, func(i, j int) bool { return catalog.Capabilities[i].ID < catalog.Capabilities[j].ID })
	return catalog, nil
}

func List(catalog Catalog, typeFilter, statusFilter string) ([]Capability, error) {
	typeFilter = strings.TrimSpace(typeFilter)
	statusFilter = strings.TrimSpace(statusFilter)
	if typeFilter != "" && !validCapabilityType(typeFilter) {
		return nil, fmt.Errorf("unsupported capability type %q", typeFilter)
	}
	if statusFilter != "" && !validCapabilityStatus(statusFilter) {
		return nil, fmt.Errorf("unsupported capability status %q", statusFilter)
	}
	selected := []Capability{}
	for _, item := range catalog.Capabilities {
		if typeFilter != "" && item.Type != typeFilter {
			continue
		}
		if statusFilter != "" && item.Status != statusFilter {
			continue
		}
		selected = append(selected, cloneCapability(item))
	}
	return selected, nil
}

func Describe(catalog Catalog, id string) (Capability, error) {
	id = strings.TrimSpace(id)
	for _, item := range catalog.Capabilities {
		if item.ID == id {
			return cloneCapability(item), nil
		}
	}
	return Capability{}, fmt.Errorf("unknown capability id %q", id)
}

func Verify(repo string, catalog Catalog, options VerifyOptions) Verification {
	result := Verification{
		RegistryPath:                  CatalogPath,
		Digest:                        catalog.Digest,
		RegisteredCount:               len(catalog.Capabilities),
		Unregistered:                  []string{},
		MissingPackages:               []string{},
		MissingDocuments:              []string{},
		InvalidPublicEntries:          []string{},
		StableWithoutVerification:     []string{},
		StablePublicWithoutQuickstart: []string{},
		StablePublicWithoutActivation: []string{},
		Errors:                        []string{},
	}
	registered := make(map[string]Capability, len(catalog.Capabilities))
	for _, item := range catalog.Capabilities {
		registered[item.Package] = item
		if info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(item.Package))); err != nil || !info.IsDir() {
			result.MissingPackages = append(result.MissingPackages, item.Package)
		}
		if item.ArchitectureDoc != "" {
			if info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(item.ArchitectureDoc))); err != nil || info.IsDir() {
				result.MissingDocuments = append(result.MissingDocuments, item.ID+": "+item.ArchitectureDoc)
			}
		}
		if item.Status == "stable" && len(item.Verification) == 0 {
			result.StableWithoutVerification = append(result.StableWithoutVerification, item.ID)
		}
		if item.Status == "stable" && len(item.PublicEntries) > 0 {
			if item.Quickstart == nil || len(item.Quickstart.Steps) == 0 {
				result.StablePublicWithoutQuickstart = append(result.StablePublicWithoutQuickstart, item.ID)
			}
			if item.Activation == nil {
				result.StablePublicWithoutActivation = append(result.StablePublicWithoutActivation, item.ID)
			}
		}
		if options.PublicEntryExists != nil {
			for _, entry := range item.PublicEntries {
				if !options.PublicEntryExists(entry) {
					result.InvalidPublicEntries = append(result.InvalidPublicEntries, item.ID+": "+entry)
				}
			}
		}
	}
	entries, err := readInternalDirectory(filepath.Join(repo, "internal"))
	if err != nil {
		result.Errors = append(result.Errors, "read internal directory: "+err.Error())
	} else {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			result.InternalDirectoryCount++
			packagePath := filepath.ToSlash(filepath.Join("internal", entry.Name()))
			if _, ok := registered[packagePath]; !ok {
				result.Unregistered = append(result.Unregistered, packagePath)
			}
		}
	}
	if options.CheckGenerated {
		result.GeneratedChecked = true
		readme, readErr := os.ReadFile(filepath.Join(repo, readmePath))
		if readErr != nil {
			result.Errors = append(result.Errors, "read README.md: "+readErr.Error())
		} else if rendered, renderErr := RenderIndex(catalog, string(readme)); renderErr != nil {
			result.Errors = append(result.Errors, renderErr.Error())
		} else {
			result.READMEUpToDate = normalizeNewlines(string(readme)) == normalizeNewlines(rendered.README)
			document, documentErr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(CapabilitiesPath)))
			result.DocumentUpToDate = documentErr == nil && normalizeNewlines(string(document)) == normalizeNewlines(rendered.Document)
		}
	}
	sort.Strings(result.Unregistered)
	sort.Strings(result.MissingPackages)
	sort.Strings(result.MissingDocuments)
	sort.Strings(result.InvalidPublicEntries)
	sort.Strings(result.StableWithoutVerification)
	sort.Strings(result.StablePublicWithoutQuickstart)
	sort.Strings(result.StablePublicWithoutActivation)
	for _, item := range result.Unregistered {
		result.Errors = append(result.Errors, "unregistered internal package: "+item)
	}
	for _, item := range result.MissingPackages {
		result.Errors = append(result.Errors, "registered package is missing: "+item)
	}
	for _, item := range result.MissingDocuments {
		result.Errors = append(result.Errors, "architecture document is missing: "+item)
	}
	for _, item := range result.InvalidPublicEntries {
		result.Errors = append(result.Errors, "public entry is absent from typed command catalog: "+item)
	}
	for _, item := range result.StableWithoutVerification {
		result.Errors = append(result.Errors, "stable capability has no verification command: "+item)
	}
	for _, item := range result.StablePublicWithoutQuickstart {
		result.Errors = append(result.Errors, "stable public capability has no quickstart: "+item)
	}
	for _, item := range result.StablePublicWithoutActivation {
		result.Errors = append(result.Errors, "stable public capability has no activation: "+item)
	}
	if result.GeneratedChecked && !result.READMEUpToDate {
		result.Errors = append(result.Errors, "README capability index is stale; run `aicoding capability index --write`")
	}
	if result.GeneratedChecked && !result.DocumentUpToDate {
		result.Errors = append(result.Errors, "docs/CAPABILITIES.md is stale; run `aicoding capability index --write`")
	}
	result.OK = len(result.Errors) == 0
	return result
}

func RenderIndex(catalog Catalog, readme string) (IndexRender, error) {
	if strings.Count(readme, readmeBeginMarker) != 1 || strings.Count(readme, readmeEndMarker) != 1 {
		return IndexRender{}, fmt.Errorf("README.md must contain exactly one capability generated block")
	}
	begin := strings.Index(readme, readmeBeginMarker)
	endRelative := strings.Index(readme[begin+len(readmeBeginMarker):], readmeEndMarker)
	if endRelative < 0 {
		return IndexRender{}, fmt.Errorf("README capability generated block is malformed")
	}
	end := begin + len(readmeBeginMarker) + endRelative + len(readmeEndMarker)
	block := renderREADMEBlock(catalog)
	if strings.Contains(readme, "\r\n") {
		block = strings.ReplaceAll(block, "\n", "\r\n")
	}
	updatedREADME := readme[:begin] + block + readme[end:]
	return IndexRender{README: updatedREADME, Document: renderCapabilitiesDocument(catalog)}, nil
}

func validateCatalog(source catalogFile) error {
	if source.SchemaVersion != 1 {
		return fmt.Errorf("%s schemaVersion must be 1", CatalogPath)
	}
	if strings.TrimSpace(source.Name) == "" || len(source.Capabilities) == 0 {
		return fmt.Errorf("%s name and capabilities are required", CatalogPath)
	}
	ids := map[string]bool{}
	packages := map[string]bool{}
	for index, item := range source.Capabilities {
		prefix := fmt.Sprintf("%s capability[%d]", CatalogPath, index)
		if !capabilityIDPattern.MatchString(item.ID) {
			return fmt.Errorf("%s has invalid id %q", prefix, item.ID)
		}
		if ids[item.ID] {
			return fmt.Errorf("duplicate capability id %q", item.ID)
		}
		ids[item.ID] = true
		if !validPackagePath(item.Package) {
			return fmt.Errorf("%s has invalid package %q", prefix, item.Package)
		}
		if packages[item.Package] {
			return fmt.Errorf("duplicate capability package %q", item.Package)
		}
		packages[item.Package] = true
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Summary) == "" {
			return fmt.Errorf("%s name and summary are required", prefix)
		}
		if !validCapabilityType(item.Type) || !validCapabilityStatus(item.Status) {
			return fmt.Errorf("%s has invalid type/status %q/%q", prefix, item.Type, item.Status)
		}
		if len(item.PublicEntries) > 0 && !validRepoRelativePath(item.ArchitectureDoc) {
			return fmt.Errorf("%s public capability requires a repository-relative architectureDoc", prefix)
		}
		if item.ArchitectureDoc != "" && !validRepoRelativePath(item.ArchitectureDoc) {
			return fmt.Errorf("%s has invalid architectureDoc %q", prefix, item.ArchitectureDoc)
		}
		publicEntries := map[string]bool{}
		for _, entry := range item.PublicEntries {
			entry = strings.TrimSpace(entry)
			if !strings.HasPrefix(entry, "aicoding ") || publicEntries[entry] {
				return fmt.Errorf("%s has invalid or duplicate public entry %q", prefix, entry)
			}
			publicEntries[entry] = true
		}
		verification := map[string]bool{}
		for _, command := range item.Verification {
			command = strings.TrimSpace(command)
			if command == "" {
				return fmt.Errorf("%s has an empty verification command", prefix)
			}
			if verification[command] {
				return fmt.Errorf("%s has duplicate verification command %q", prefix, command)
			}
			verification[command] = true
		}
		if item.Quickstart != nil {
			if len(item.Quickstart.Steps) == 0 {
				return fmt.Errorf("%s quickstart.steps must not be empty", prefix)
			}
			steps := map[string]bool{}
			for _, step := range item.Quickstart.Steps {
				step = strings.TrimSpace(step)
				if !strings.HasPrefix(step, "aicoding ") || steps[step] {
					return fmt.Errorf("%s has invalid or duplicate quickstart step %q", prefix, step)
				}
				steps[step] = true
			}
			if item.Quickstart.ExampleInput != "" && !validRepoRelativePath(item.Quickstart.ExampleInput) {
				return fmt.Errorf("%s has invalid quickstart exampleInput %q", prefix, item.Quickstart.ExampleInput)
			}
		}
		if item.Activation != nil {
			if item.Activation.Kind != "cli-entry" && item.Activation.Kind != "kit-install" {
				return fmt.Errorf("%s has invalid activation kind %q", prefix, item.Activation.Kind)
			}
			if strings.TrimSpace(item.Activation.Note) == "" || !strings.HasPrefix(strings.TrimSpace(item.Activation.AgentUsage), "aicoding ") {
				return fmt.Errorf("%s activation requires note and aicoding agentUsage", prefix)
			}
		}
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON value")
}

func validPackagePath(value string) bool {
	if !strings.HasPrefix(value, "internal/") || strings.Count(value, "/") != 1 {
		return false
	}
	name := strings.TrimPrefix(value, "internal/")
	return name != "" && capabilityPackagePattern.MatchString(name)
}

func validRepoRelativePath(value string) bool {
	if strings.TrimSpace(value) == "" || filepath.IsAbs(value) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func validCapabilityType(value string) bool {
	switch value {
	case "primitive", "domain-capability", "product-workflow", "internal-only":
		return true
	default:
		return false
	}
}

func validCapabilityStatus(value string) bool {
	switch value {
	case "experimental", "beta", "stable", "deprecated":
		return true
	default:
		return false
	}
}

func cloneCapabilities(items []Capability) []Capability {
	cloned := make([]Capability, len(items))
	for index, item := range items {
		cloned[index] = cloneCapability(item)
	}
	return cloned
}

func cloneCapability(item Capability) Capability {
	item.PublicEntries = append([]string{}, item.PublicEntries...)
	item.Verification = append([]string{}, item.Verification...)
	if item.Quickstart != nil {
		quickstart := *item.Quickstart
		quickstart.Steps = append([]string{}, item.Quickstart.Steps...)
		item.Quickstart = &quickstart
	}
	if item.Activation != nil {
		activation := *item.Activation
		item.Activation = &activation
	}
	return item
}

func renderREADMEBlock(catalog Catalog) string {
	var out strings.Builder
	out.WriteString(readmeBeginMarker)
	out.WriteString("\n\n> 此区由 `config/internal-capabilities.json` 生成（`")
	out.WriteString(catalog.Digest)
	out.WriteString("`）。完整的 ")
	out.WriteString(fmt.Sprintf("%d", len(catalog.Capabilities)))
	out.WriteString(" 项能力见 [能力索引](docs/CAPABILITIES.md)。\n\n")
	out.WriteString("| 可直接使用的能力 | 核心职责 | 快速入口 | 使用闭环 | 架构 |\n")
	out.WriteString("|---|---|---|---|---|\n")
	for _, item := range catalog.Capabilities {
		if len(item.PublicEntries) == 0 {
			continue
		}
		fmt.Fprintf(&out, "| `%s` %s | %s | %s | %s | %s |\n",
			markdownCell(item.ID), markdownCell(item.Name), markdownCell(item.Summary),
			codeList(item.PublicEntries[:1]), readmeUsageLink(item.ID), readmeArchitectureLink(item.ArchitectureDoc))
	}
	out.WriteString("\n")
	out.WriteString(readmeEndMarker)
	return out.String()
}

func renderCapabilitiesDocument(catalog Catalog) string {
	var out strings.Builder
	out.WriteString("# AiCoding 平台能力索引\n\n")
	out.WriteString("> 本文件由 `config/internal-capabilities.json` 生成，请运行 ")
	out.WriteString("`bin/aicoding.exe capability index --write` 更新。\n\n")
	out.WriteString("Registry digest: `")
	out.WriteString(catalog.Digest)
	out.WriteString("`\n\n")
	out.WriteString(fmt.Sprintf("共登记 %d 个 `internal/` 一级包；文档义务按公共入口、内部实现域和 Primitive 分级。\n\n", len(catalog.Capabilities)))
	out.WriteString("- `publicEntries` 非空：必须指向 typed command catalog 中的现存入口，并登记架构文档。\n")
	out.WriteString("- `stable` 且 `publicEntries` 非空：必须登记 quickstart 与 activation，避免只有命令没有用法。\n")
	out.WriteString("- `internal-only`：没有公共入口时可不单建架构文档，避免文档剧场。\n")
	out.WriteString("- `stable`：必须登记至少一条可执行验证命令；`beta`/`experimental` 仍需明确状态。\n\n")
	out.WriteString("| ID | Package | Type | Status | Summary | Public entries | Architecture | Verification |\n")
	out.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, item := range catalog.Capabilities {
		fmt.Fprintf(&out, "| `%s` | `%s` | `%s` | `%s` | %s | %s | %s | %s |\n",
			markdownCell(item.ID), markdownCell(item.Package), markdownCell(item.Type), markdownCell(item.Status),
			markdownCell(item.Summary), codeList(item.PublicEntries), documentArchitectureLink(item.ArchitectureDoc), codeList(item.Verification))
	}
	out.WriteString("\n## 公共能力使用闭环\n\n")
	for _, item := range catalog.Capabilities {
		if len(item.PublicEntries) == 0 {
			continue
		}
		fmt.Fprintf(&out, "<a id=\"capability-%s\"></a>\n\n", item.ID)
		fmt.Fprintf(&out, "### `%s` %s\n\n", markdownCell(item.ID), markdownCell(item.Name))
		fmt.Fprintf(&out, "- 当前状态：`%s`（`%s`）\n", markdownCell(item.Status), markdownCell(item.Type))
		fmt.Fprintf(&out, "- 是什么：%s\n", markdownCell(item.Summary))
		fmt.Fprintf(&out, "- 架构图：%s\n", documentArchitectureLink(item.ArchitectureDoc))
		out.WriteString("- 怎么用：\n")
		if item.Quickstart == nil || len(item.Quickstart.Steps) == 0 {
			out.WriteString("  - —\n")
		} else {
			for index, step := range item.Quickstart.Steps {
				fmt.Fprintf(&out, "  %d. `%s`\n", index+1, strings.ReplaceAll(markdownCell(step), "`", "\\`"))
			}
			if item.Quickstart.ExampleInput != "" {
				fmt.Fprintf(&out, "  - 示例输入：`%s`\n", markdownCell(item.Quickstart.ExampleInput))
			}
		}
		if item.Activation == nil {
			out.WriteString("- 怎么进 Agent：—\n")
		} else {
			fmt.Fprintf(&out, "- 怎么进 Agent：`%s`；%s；调用 `%s`。\n",
				markdownCell(item.Activation.Kind), markdownCell(item.Activation.Note), markdownCell(item.Activation.AgentUsage))
		}
		fmt.Fprintf(&out, "- 怎么验证：%s\n", codeList(item.Verification))
		fmt.Fprintf(&out, "- 一次查看：`bin/aicoding.exe capability describe --id %s --json`\n\n", markdownCell(item.ID))
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}

func codeList(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	encoded := make([]string, len(values))
	for index, value := range values {
		encoded[index] = "`" + strings.ReplaceAll(markdownCell(value), "`", "\\`") + "`"
	}
	return strings.Join(encoded, "<br>")
}

func readmeArchitectureLink(path string) string {
	if path == "" {
		return "—"
	}
	return "[文档](" + path + ")"
}

func readmeUsageLink(id string) string {
	return "[describe](docs/CAPABILITIES.md#capability-" + id + ")"
}

func documentArchitectureLink(path string) string {
	if path == "" {
		return "—"
	}
	target := strings.TrimPrefix(filepath.ToSlash(path), "docs/")
	return "[文档](" + target + ")"
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func normalizeNewlines(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}
