package cuserstyle

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type snippetStringList []string

type snippetDefinition struct {
	Prefix      snippetStringList `json:"prefix"`
	Body        snippetStringList `json:"body"`
	Description string            `json:"description"`
	Scope       string            `json:"scope,omitempty"`
}

type snippetValues map[string]string

var snippetDefaultPlaceholder = regexp.MustCompile(`\$\{([0-9]+):([^{}]*)\}`)
var snippetBracedPlaceholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*|[0-9]+)\}`)
var snippetSimplePlaceholder = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*|[0-9]+)`)

func (values *snippetValues) String() string {
	if values == nil {
		return ""
	}
	keys := make([]string, 0, len(*values))
	for key := range *values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+(*values)[key])
	}
	return strings.Join(parts, ",")
}

func (values *snippetValues) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if (len(parts) != 2) || (parts[0] == "") {
		return fmt.Errorf("--set requires KEY=VALUE")
	}
	if *values == nil {
		*values = make(snippetValues)
	}
	(*values)[parts[0]] = parts[1]
	return nil
}

func (values *snippetStringList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*values = snippetStringList{single}
		return nil
	}

	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return fmt.Errorf("expected a string or string array: %w", err)
	}
	*values = multiple
	return nil
}

func RunSnippet(args []string) error {
	fs := flag.NewFlagSet("snippet", flag.ContinueOnError)
	snippetsPath := fs.String("snippets", "", "VS Code snippets JSON")
	name := fs.String("name", "", "snippet name")
	list := fs.Bool("list", false, "list snippet names")
	target := fs.String("target", "", "target filename used by TM_FILENAME")
	out := fs.String("out", "", "write rendered content to this file")
	values := make(snippetValues)
	fs.Var(&values, "set", "override a variable or tabstop with KEY=VALUE")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *snippetsPath == "" {
		return fmt.Errorf("--snippets is required")
	}

	catalog, err := loadSnippetCatalog(*snippetsPath)
	if err != nil {
		return err
	}
	if *list {
		if *name != "" {
			return fmt.Errorf("--list and --name cannot be used together")
		}
		for _, snippetName := range sortedSnippetNames(catalog) {
			fmt.Println(snippetName)
		}
		return nil
	}
	if *name == "" {
		return fmt.Errorf("--name is required unless --list is used")
	}

	snippet, ok := catalog[*name]
	if !ok {
		return fmt.Errorf("snippet %q was not found", *name)
	}
	targetName := *target
	if (targetName == "") && (*out != "") {
		targetName = filepath.Base(*out)
	}
	if targetName == "" {
		targetName = "snippet.c"
	}
	content := renderSnippet(snippet, targetName, values, time.Now())
	if *out == "" {
		fmt.Print(content)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := writeAtomic(*out, []byte(content)); err != nil {
		return err
	}
	fmt.Println(*out)
	return nil
}

func loadSnippetCatalog(path string) (map[string]snippetDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rawCatalog map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawCatalog); err != nil {
		return nil, fmt.Errorf("parse snippets %s: %w", path, err)
	}
	if len(rawCatalog) == 0 {
		return nil, fmt.Errorf("snippet catalog %s is empty", path)
	}

	catalog := make(map[string]snippetDefinition, len(rawCatalog))
	for name, raw := range rawCatalog {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		var snippet snippetDefinition
		if err := decoder.Decode(&snippet); err != nil {
			return nil, fmt.Errorf("snippet %q: %w", name, err)
		}
		if (len(snippet.Prefix) == 0) || (len(snippet.Body) == 0) ||
			(snippet.Description == "") {
			return nil, fmt.Errorf("snippet %q requires prefix, body, and description", name)
		}
		catalog[name] = snippet
	}
	return catalog, nil
}

func sortedSnippetNames(catalog map[string]snippetDefinition) []string {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func renderSnippet(snippet snippetDefinition, target string, overrides snippetValues, now time.Time) string {
	values := snippetValues{
		"0":             "",
		"TM_FILENAME":   filepath.Base(target),
		"CURRENT_YEAR":  now.Format("2006"),
		"CURRENT_MONTH": now.Format("01"),
		"CURRENT_DATE":  now.Format("02"),
	}
	for key, value := range overrides {
		values[key] = value
	}

	content := strings.Join(snippet.Body, "\n")
	content = snippetDefaultPlaceholder.ReplaceAllStringFunc(content, func(match string) string {
		parts := snippetDefaultPlaceholder.FindStringSubmatch(match)
		if value, ok := values[parts[1]]; ok {
			return value
		}
		return parts[2]
	})
	content = snippetBracedPlaceholder.ReplaceAllStringFunc(content, func(match string) string {
		parts := snippetBracedPlaceholder.FindStringSubmatch(match)
		if value, ok := values[parts[1]]; ok {
			return value
		}
		return match
	})
	content = snippetSimplePlaceholder.ReplaceAllStringFunc(content, func(match string) string {
		parts := snippetSimplePlaceholder.FindStringSubmatch(match)
		if value, ok := values[parts[1]]; ok {
			return value
		}
		return match
	})
	return strings.TrimRight(content, "\n") + "\n"
}
