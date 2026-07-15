package cuserstyle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return Config{}, err
	}

	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	if cfg.Schema != "c-userstyle" {
		return Config{}, fmt.Errorf("schema must be c-userstyle")
	}
	if cfg.Standard != "c99" {
		return Config{}, fmt.Errorf("standard must be c99")
	}
	if cfg.Reference.Pages != 61 || cfg.Reference.ExpectedClauses != 139 {
		return Config{}, fmt.Errorf("reference must declare 61 pages and 139 clauses")
	}
	if cfg.Reference.PDF == "" || cfg.Reference.Markdown == "" || cfg.Reference.RuleCatalog == "" {
		return Config{}, fmt.Errorf("reference pdf, markdown and ruleCatalog are required")
	}
	if cfg.Style.IndentWidth <= 0 {
		cfg.Style.IndentWidth = 4
	}
	if cfg.Style.Continuation <= 0 {
		cfg.Style.Continuation = 4
	}
	if cfg.Style.ColumnLimit <= 0 {
		cfg.Style.ColumnLimit = 100
	}
	if cfg.Safety.MaxParameters <= 0 {
		cfg.Safety.MaxParameters = 5
	}
	if cfg.Safety.MaxParameters > 5 {
		return Config{}, fmt.Errorf("safety maxParameters must not exceed 5")
	}
	if cfg.Docs.EmployeeIDPolicy == "" {
		cfg.Docs.EmployeeIDPolicy = "whenProvided"
	}
	if !oneOf(cfg.Docs.EmployeeIDPolicy, "omit", "whenProvided", "required") {
		return Config{}, fmt.Errorf("documentation employeeIdPolicy must be omit, whenProvided or required")
	}
	if cfg.Docs.ModificationHistoryPolicy == "" {
		cfg.Docs.ModificationHistoryPolicy = "disabled"
	}
	if !oneOf(cfg.Docs.ModificationHistoryPolicy, "disabled", "maintenance-release", "required") {
		return Config{}, fmt.Errorf("documentation modificationHistoryPolicy must be disabled, maintenance-release or required")
	}
	if cfg.Macro.SimpleObjectCommentStyle == "" {
		cfg.Macro.SimpleObjectCommentStyle = "block"
	}
	if !oneOf(cfg.Macro.SimpleObjectCommentStyle, "block", "doxygen", "either") {
		return Config{}, fmt.Errorf("macros simpleObjectCommentStyle must be block, doxygen or either")
	}
	if cfg.Readability.ComplexFunction.MinEffectiveLines <= 0 {
		cfg.Readability.ComplexFunction.MinEffectiveLines = 20
	}
	if cfg.Readability.ComplexFunction.MinBranches <= 0 {
		cfg.Readability.ComplexFunction.MinBranches = 3
	}
	if cfg.Readability.ComplexFunction.MinNesting <= 0 {
		cfg.Readability.ComplexFunction.MinNesting = 3
	}
	if cfg.Hook.MaxDiagnostics <= 0 {
		cfg.Hook.MaxDiagnostics = 80
	}
	if cfg.Template.ModuleName == "" || cfg.Template.FileStem == "" || cfg.Template.ContextType == "" {
		return Config{}, fmt.Errorf("template moduleName, fileStem and contextType are required")
	}
	if cfg.Template.HeaderGuard == "" {
		cfg.Template.HeaderGuard = cfg.Template.ModuleName + "_H"
	}
	if cfg.Gates.GCC.LanguageStandard != "c99" || cfg.Gates.Clang.LanguageStandard != "c99" {
		return Config{}, fmt.Errorf("gcc and clang gates must use c99")
	}
	if cfg.Gates.HeaderC.LanguageStandard != "c99" ||
		cfg.Gates.HeaderCXX.LanguageStandard != "c++17" {
		return Config{}, fmt.Errorf("header gates must use c99 and c++17")
	}
	return cfg, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return err
				}
				key := keyTok.(string)
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := walkValue(dec, walk); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case '[':
			for dec.More() {
				if err := walkValue(dec, walk); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		}
		return nil
	}
	if err := walk(); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func walkValue(dec *json.Decoder, walk func() error) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); ok {
		switch d {
		case '{':
			seen := map[string]struct{}{}
			for dec.More() {
				k, err := dec.Token()
				if err != nil {
					return err
				}
				key := k.(string)
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := walkValue(dec, walk); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case '[':
			for dec.More() {
				if err := walkValue(dec, walk); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		}
	}
	return nil
}

func isExcluded(path string, cfg Config) bool {
	p := filepath.ToSlash(path)
	for _, segment := range cfg.Scope.Exclude {
		segment = strings.Trim(segment, "/")
		if segment != "" && (p == segment || strings.HasPrefix(p, segment+"/") || strings.Contains(p, "/"+segment+"/")) {
			return true
		}
	}
	return false
}
