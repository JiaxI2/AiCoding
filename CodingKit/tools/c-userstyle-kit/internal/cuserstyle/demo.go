package cuserstyle

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/* templates/simple/* templates/advanced/* templates/tests/*
var goldenDemoTemplates embed.FS

type demoFile struct {
	Template     string
	Output       string
	Replacements []string
}

var goldenDemoFiles = []demoFile{
	{Template: "templates/simple/demo.h", Output: "demo.h"},
	{Template: "templates/simple/demo.c", Output: "demo.c"},
	{Template: "templates/advanced/README.md", Output: "advanced/README.md"},
	{
		Template: "templates/demo.h",
		Output:   "advanced/state_machine.h",
		Replacements: []string{
			"demo_test.c", "advanced_test.c",
			"demo.c", "state_machine.c",
			"demo.h", "state_machine.h",
			"DEMO_H", "ADVANCED_STATE_MACHINE_H",
		},
	},
	{
		Template: "templates/demo.c",
		Output:   "advanced/state_machine.c",
		Replacements: []string{
			"demo_test.c", "advanced_test.c",
			"demo.c", "state_machine.c",
			"demo.h", "state_machine.h",
		},
	},
	{
		Template: "templates/demo_protocol.h",
		Output:   "advanced/protocol.h",
		Replacements: []string{
			"demo_test.c", "advanced_test.c",
			"demo_protocol.c", "protocol.c",
			"demo_protocol.h", "protocol.h",
			"demo_pool.h", "fixed_pool.h",
			"demo.h", "state_machine.h",
			"DEMO_PROTOCOL_H", "ADVANCED_PROTOCOL_H",
		},
	},
	{
		Template: "templates/demo_protocol.c",
		Output:   "advanced/protocol.c",
		Replacements: []string{
			"demo_test.c", "advanced_test.c",
			"demo_protocol.c", "protocol.c",
			"demo_protocol.h", "protocol.h",
		},
	},
	{
		Template: "templates/demo_pool.h",
		Output:   "advanced/fixed_pool.h",
		Replacements: []string{
			"demo_test.c", "advanced_test.c",
			"demo_pool.c", "fixed_pool.c",
			"demo_pool.h", "fixed_pool.h",
			"demo_protocol.h", "protocol.h",
			"demo.h", "state_machine.h",
			"DEMO_POOL_H", "ADVANCED_FIXED_POOL_H",
		},
	},
	{
		Template: "templates/demo_pool.c",
		Output:   "advanced/fixed_pool.c",
		Replacements: []string{
			"demo_test.c", "advanced_test.c",
			"demo_pool.c", "fixed_pool.c",
			"demo_pool.h", "fixed_pool.h",
		},
	},
	{Template: "templates/tests/demo_test.c", Output: "advanced/tests/advanced_test.c"},
}

var legacyDemoFiles = []demoFile{
	{Template: "templates/demo_protocol.h", Output: "demo_protocol.h"},
	{Template: "templates/demo_protocol.c", Output: "demo_protocol.c"},
	{Template: "templates/demo_pool.h", Output: "demo_pool.h"},
	{Template: "templates/demo_pool.c", Output: "demo_pool.c"},
	{Template: "templates/tests/demo_test.c", Output: "tests/demo_test.c"},
}

func RunDemo(args []string) error {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	configPath := fs.String("config", "", "configuration")
	out := fs.String("out", ".", "output directory")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}
	generated, err := generateDemo(cfg, *out)
	if err != nil {
		return err
	}

	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    true,
			"files": generated,
		})
	}
	for _, path := range generated {
		fmt.Println(path)
	}
	return nil
}

func renderHeader(cfg Config) string {
	return renderDemoFile(goldenDemoFiles[0], cfg)
}

func renderSource(cfg Config) string {
	return renderDemoFile(goldenDemoFiles[1], cfg)
}

func generateDemo(cfg Config, out string) ([]string, error) {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	if err := removeUnmodifiedLegacyDemoFiles(cfg, out); err != nil {
		return nil, err
	}

	generated := make([]string, 0, len(goldenDemoFiles))
	for _, file := range goldenDemoFiles {
		path := filepath.Join(out, filepath.FromSlash(file.Output))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := writeAtomic(path, []byte(renderDemoFile(file, cfg))); err != nil {
			return nil, err
		}
		generated = append(generated, path)
	}
	return generated, nil
}

func renderDemoFile(file demoFile, cfg Config) string {
	data, err := goldenDemoTemplates.ReadFile(file.Template)
	if err != nil {
		panic(err)
	}
	content := string(data)
	if len(file.Replacements) > 0 {
		content = strings.NewReplacer(file.Replacements...).Replace(content)
	}
	return applyNewline(content, cfg.Style.Newline)
}

func removeUnmodifiedLegacyDemoFiles(cfg Config, out string) error {
	for _, file := range legacyDemoFiles {
		path := filepath.Join(out, filepath.FromSlash(file.Output))
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		matchesLegacyTemplate := string(content) == renderDemoFile(file, cfg)
		if file.Output == "tests/demo_test.c" {
			legacyText := string(content)
			matchesLegacyTemplate = strings.Contains(legacyText, "@file demo_test.c") &&
				strings.Contains(legacyText, "#include \"demo_pool.h\"") &&
				strings.Contains(legacyText, "#include \"demo_protocol.h\"")
		}
		if matchesLegacyTemplate {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	testsPath := filepath.Join(out, "tests")
	entries, err := os.ReadDir(testsPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.Remove(testsPath)
	}
	return nil
}

func applyNewline(s, mode string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if strings.EqualFold(mode, "crlf") {
		return strings.ReplaceAll(s, "\n", "\r\n")
	}
	return s
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
