package cuserstyle

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoadConfigWithOverlays loads one complete base configuration and applies
// partial JSON overlays from left to right. Objects merge recursively through
// encoding/json, scalar values overwrite, and arrays replace the base value.
func LoadConfigWithOverlays(configPath string, overlayPaths []string) (Config, string, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return Config{}, "", err
	}

	for _, path := range overlayPaths {
		if err := applyConfigOverlay(&cfg, path); err != nil {
			return Config{}, "", err
		}
	}
	if err := validateEffectiveConfig(cfg); err != nil {
		return Config{}, "", err
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return Config{}, "", fmt.Errorf("encode effective config: %w", err)
	}
	digest := sha256.Sum256(data)
	return cfg, fmt.Sprintf("%X", digest), nil
}

func applyConfigOverlay(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read overlay %s: %w", path, err)
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return fmt.Errorf("invalid overlay %s: %w", path, err)
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid overlay %s: %w", path, err)
	}
	root, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid overlay %s: root must be an object", path)
	}
	if containsJSONNull(root) {
		return fmt.Errorf("invalid overlay %s: null values are not supported", path)
	}
	for _, locked := range []string{"schema", "standard", "reference"} {
		if _, exists := root[locked]; exists {
			return fmt.Errorf("invalid overlay %s: %s is locked by the base config", path, locked)
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("invalid overlay %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid overlay %s: multiple JSON values", path)
		}
		return fmt.Errorf("invalid overlay %s: %w", path, err)
	}
	return nil
}

func containsJSONNull(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case map[string]any:
		for _, child := range typed {
			if containsJSONNull(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsJSONNull(child) {
				return true
			}
		}
	}
	return false
}

func validateEffectiveConfig(cfg Config) error {
	if strings.TrimSpace(cfg.ID) == "" {
		return fmt.Errorf("effective config id is required")
	}
	if cfg.Schema != "c-userstyle" || cfg.Standard != "c99" {
		return fmt.Errorf("effective config must retain c-userstyle and c99")
	}
	if cfg.Style.IndentWidth <= 0 || cfg.Style.Continuation <= 0 || cfg.Style.ColumnLimit <= 0 {
		return fmt.Errorf("effective style widths must be positive")
	}
	if cfg.Safety.MaxParameters <= 0 || cfg.Safety.MaxParameters > 5 {
		return fmt.Errorf("effective safety maxParameters must be between 1 and 5")
	}
	if !oneOf(cfg.Docs.EmployeeIDPolicy, "omit", "whenProvided", "required") {
		return fmt.Errorf("effective documentation employeeIdPolicy is invalid")
	}
	if !oneOf(
		cfg.Docs.ModificationHistoryPolicy,
		"disabled",
		"maintenance-release",
		"required",
	) {
		return fmt.Errorf("effective documentation modificationHistoryPolicy is invalid")
	}
	if !oneOf(cfg.Macro.SimpleObjectCommentStyle, "block", "doxygen", "either") {
		return fmt.Errorf("effective macros simpleObjectCommentStyle is invalid")
	}
	if cfg.Readability.ComplexFunction.MinEffectiveLines <= 0 ||
		cfg.Readability.ComplexFunction.MinBranches <= 0 ||
		cfg.Readability.ComplexFunction.MinNesting <= 0 {
		return fmt.Errorf("effective readability complexFunction thresholds must be positive")
	}
	if cfg.Hook.MaxDiagnostics <= 0 {
		return fmt.Errorf("effective hook maxDiagnostics must be positive")
	}
	if cfg.Template.ModuleName == "" || cfg.Template.FileStem == "" ||
		cfg.Template.ContextType == "" || cfg.Template.HeaderGuard == "" {
		return fmt.Errorf(
			"effective template moduleName, fileStem, contextType and headerGuard are required",
		)
	}
	if cfg.Gates.GCC.LanguageStandard != "c99" || cfg.Gates.Clang.LanguageStandard != "c99" {
		return fmt.Errorf("effective gcc and clang gates must use c99")
	}
	if cfg.Gates.HeaderC.LanguageStandard != "c99" ||
		cfg.Gates.HeaderCXX.LanguageStandard != "c++17" {
		return fmt.Errorf("effective header gates must use c99 and c++17")
	}
	return validateVerificationGateProfiles(cfg)
}

func validateVerificationGateProfiles(cfg Config) error {
	strictC := []string{
		"-std=c99",
		"-pedantic-errors",
		"-Wall",
		"-Wextra",
		"-Werror",
		"-Wconversion",
		"-Wsign-conversion",
		"-Wshadow",
		"-Wstrict-prototypes",
		"-Wmissing-prototypes",
		"-Wvla",
		"-Wformat=2",
		"-Wundef",
		"-Wcast-qual",
		"-Wwrite-strings",
	}
	headerCRequired := []string{"-std=c99", "-pedantic-errors", "-Wall", "-Wextra", "-Werror"}
	headerCXXRequired := []string{"-std=c++17", "-pedantic-errors", "-Wall", "-Wextra", "-Werror"}
	headerCXXAllowed := []string{
		"-std=c++17",
		"-pedantic-errors",
		"-Wall",
		"-Wextra",
		"-Werror",
		"-Wconversion",
		"-Wsign-conversion",
		"-Wshadow",
		"-Wformat=2",
		"-Wundef",
		"-Wcast-qual",
		"-Wwrite-strings",
	}

	checks := []struct {
		name     string
		profile  GateProfile
		required []string
		allowed  []string
	}{
		{name: "gcc", profile: cfg.Gates.GCC, required: strictC, allowed: strictC},
		{name: "clang", profile: cfg.Gates.Clang, required: strictC, allowed: strictC},
		{name: "headerC", profile: cfg.Gates.HeaderC, required: headerCRequired, allowed: strictC},
		{
			name:     "headerCxx",
			profile:  cfg.Gates.HeaderCXX,
			required: headerCXXRequired,
			allowed:  headerCXXAllowed,
		},
	}
	for _, check := range checks {
		if err := validateVerificationGate(check.name, check.profile, check.required, check.allowed); err != nil {
			return err
		}
	}
	return nil
}

func validateVerificationGate(
	name string,
	profile GateProfile,
	required []string,
	allowed []string,
) error {
	if !profile.WarningsAsErrors {
		return fmt.Errorf("effective gates.%s warningsAsErrors must remain true", name)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, flag := range allowed {
		allowedSet[flag] = struct{}{}
	}
	actual := make(map[string]struct{}, len(profile.Flags))
	for _, flag := range profile.Flags {
		if _, exists := allowedSet[flag]; !exists {
			return fmt.Errorf("effective gates.%s flag %q is not allowed", name, flag)
		}
		if _, duplicate := actual[flag]; duplicate {
			return fmt.Errorf("effective gates.%s contains duplicate flag %q", name, flag)
		}
		actual[flag] = struct{}{}
	}
	for _, flag := range required {
		if _, exists := actual[flag]; !exists {
			return fmt.Errorf("effective gates.%s is missing required flag %q", name, flag)
		}
	}
	return nil
}
