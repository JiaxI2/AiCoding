package cuserstyle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	verifyResultSchema = "cstylekit.verify.v1"
	verifyTargetSchema = "cstylekit-verify-target-v1"
	verifyTimeout      = 5 * time.Minute
)

type verifyTarget struct {
	Schema    string          `json:"schema"`
	ID        string          `json:"id"`
	Candidate verifyCandidate `json:"candidate"`
	Baseline  verifyBaseline  `json:"baseline,omitempty"`
	Host      verifyHost      `json:"host,omitempty"`
}

type verifyCandidate struct {
	Source string `json:"source"`
	Header string `json:"header"`
}

type verifyBaseline struct {
	Source string `json:"source"`
	Header string `json:"header"`
}

type verifyHost struct {
	TestSource     string   `json:"testSource"`
	SupportSources []string `json:"supportSources"`
	SupportHeaders []string `json:"supportHeaders"`
	Defines        []string `json:"defines"`
}

type verifyOptions struct {
	ConfigPath   string
	OverlayPaths []string
	TargetPath   string
	Profile      string
	Timings      bool
}

type VerifyFileReport struct {
	Role    string `json:"role"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Managed bool   `json:"managed"`
}

type VerifyStepResult struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	Message    string   `json:"message,omitempty"`
	DurationMS *float64 `json:"durationMs,omitempty"`
}

type VerifyResult struct {
	Schema                string                 `json:"schema"`
	OK                    bool                   `json:"ok"`
	Profile               string                 `json:"profile"`
	ConfigID              string                 `json:"configId"`
	TargetID              string                 `json:"targetId"`
	EffectiveConfigSHA256 string                 `json:"effectiveConfigSha256"`
	Files                 []VerifyFileReport     `json:"files"`
	Diagnostics           []Diagnostic           `json:"diagnostics"`
	Readability           FileReadabilitySummary `json:"readability"`
	Steps                 []VerifyStepResult     `json:"steps"`
	TotalDurationMS       float64                `json:"totalDurationMs,omitempty"`
}

// RunVerify validates external C files without invoking a firmware toolchain.
func RunVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	configPath := fs.String("config", "", "complete base configuration")
	var overlays multiFlag
	fs.Var(&overlays, "overlay", "partial configuration overlay; may be repeated")
	targetPath := fs.String("target", "", "external verification target manifest")
	profile := fs.String("profile", "fast", "fast|full")
	timings := fs.Bool("timings", false, "include per-step timings")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" || *targetPath == "" {
		return fmt.Errorf("--config and --target are required")
	}
	if *profile != "fast" && *profile != "full" {
		return fmt.Errorf("unsupported verify profile %q", *profile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), verifyTimeout)
	defer cancel()
	result, err := executeVerification(ctx, verifyOptions{
		ConfigPath:   *configPath,
		OverlayPaths: overlays,
		TargetPath:   *targetPath,
		Profile:      *profile,
		Timings:      *timings,
	}, osVerifyRunner{})
	if err != nil {
		return err
	}
	if err := emitVerifyResult(result, *jsonOut, *timings); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("C UserStyle external verification failed")
	}
	return nil
}

func executeVerification(
	ctx context.Context,
	options verifyOptions,
	runner verifyCommandRunner,
) (VerifyResult, error) {
	started := time.Now()
	cfg, configHash, err := LoadConfigWithOverlays(options.ConfigPath, options.OverlayPaths)
	if err != nil {
		return VerifyResult{}, err
	}
	target, err := loadVerifyTarget(options.TargetPath)
	if err != nil {
		return VerifyResult{}, err
	}
	tempDir, err := os.MkdirTemp("", "cstylekit-verify-")
	if err != nil {
		return VerifyResult{}, fmt.Errorf("create verification temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	result := VerifyResult{
		Schema:                verifyResultSchema,
		Profile:               options.Profile,
		ConfigID:              cfg.ID,
		TargetID:              target.ID,
		EffectiveConfigSHA256: configHash,
		Files:                 []VerifyFileReport{},
		Diagnostics:           []Diagnostic{},
		Steps:                 []VerifyStepResult{},
	}
	finish := func() {
		if options.Timings {
			result.TotalDurationMS = durationMilliseconds(time.Since(started))
		}
	}

	var fileData map[string][]byte
	var snapshotTarget verifyTarget
	if !runVerifyStep(&result, "scope-hash", options.Timings, func() error {
		var collectErr error
		result.Files, fileData, snapshotTarget, collectErr = collectVerifyFiles(
			target,
			cfg,
			tempDir,
		)
		return collectErr
	}) {
		finish()
		return result, nil
	}

	lintPassed := runVerifyStep(&result, "lint", options.Timings, func() error {
		paths := []string{target.Candidate.Source, target.Candidate.Header}
		for _, path := range paths {
			result.Diagnostics = append(
				result.Diagnostics,
				lintContent(path, fileData[path], nil, cfg, false)...,
			)
		}
		if len(result.Diagnostics) > cfg.Hook.MaxDiagnostics {
			result.Diagnostics = result.Diagnostics[:cfg.Hook.MaxDiagnostics]
		}
		if len(result.Diagnostics) != 0 {
			return fmt.Errorf("reported %d diagnostic(s)", len(result.Diagnostics))
		}
		return nil
	})
	runVerifyStep(&result, "readability", options.Timings, func() error {
		result.Readability = analyzeReadability(
			target.Candidate.Source,
			fileData[target.Candidate.Source],
			cfg,
		)
		return nil
	})
	if !lintPassed {
		finish()
		return result, nil
	}

	if !runVerifyStep(&result, "gcc-c99", options.Timings, func() error {
		return verifySourceSyntax(
			ctx,
			runner,
			"gcc",
			cfg.Gates.GCC,
			snapshotTarget,
			tempDir,
			nil,
		)
	}) {
		finish()
		return result, nil
	}
	if !runVerifyStep(&result, "gcc-header-c99", options.Timings, func() error {
		return verifyHeader(
			ctx,
			runner,
			"gcc",
			cfg.Gates.HeaderC,
			snapshotTarget,
			tempDir,
			"c",
			nil,
		)
	}) {
		finish()
		return result, nil
	}

	var candidateOutput []byte
	if !runVerifyStep(&result, "candidate-host-test", options.Timings, func() error {
		if snapshotTarget.Host.TestSource == "" {
			return fmt.Errorf("host.testSource is required by profile %s", options.Profile)
		}
		var hostErr error
		candidateOutput, hostErr = compileAndRunHost(
			ctx,
			runner,
			cfg.Gates.GCC,
			snapshotTarget.Candidate.Source,
			snapshotTarget,
			tempDir,
			"candidate-host",
		)
		return hostErr
	}) {
		finish()
		return result, nil
	}

	if options.Profile == "full" {
		var clangPath string
		var clangArgs []string
		if !runVerifyStep(&result, "clang-c99", options.Timings, func() error {
			var resolveErr error
			clangPath, clangArgs, resolveErr = resolveClangHost(ctx, runner, "clang", tempDir)
			if resolveErr != nil {
				return resolveErr
			}
			return verifySourceSyntaxWithPath(
				ctx,
				runner,
				clangPath,
				cfg.Gates.Clang,
				snapshotTarget,
				tempDir,
				clangArgs,
			)
		}) {
			finish()
			return result, nil
		}
		if !runVerifyStep(&result, "clang-header-c99", options.Timings, func() error {
			return verifyHeaderWithPath(
				ctx,
				runner,
				clangPath,
				cfg.Gates.HeaderC,
				snapshotTarget,
				tempDir,
				"c",
				clangArgs,
			)
		}) {
			finish()
			return result, nil
		}
		if !runVerifyStep(&result, "gxx-header-cxx17", options.Timings, func() error {
			return verifyHeader(
				ctx,
				runner,
				"g++",
				cfg.Gates.HeaderCXX,
				snapshotTarget,
				tempDir,
				"c++",
				nil,
			)
		}) {
			finish()
			return result, nil
		}
		var clangXXPath string
		var clangXXArgs []string
		if !runVerifyStep(&result, "clangxx-header-cxx17", options.Timings, func() error {
			var resolveErr error
			clangXXPath, clangXXArgs, resolveErr = resolveClangHost(
				ctx,
				runner,
				"clang++",
				tempDir,
			)
			if resolveErr != nil {
				return resolveErr
			}
			return verifyHeaderWithPath(
				ctx,
				runner,
				clangXXPath,
				cfg.Gates.HeaderCXX,
				snapshotTarget,
				tempDir,
				"c++",
				clangXXArgs,
			)
		}) {
			finish()
			return result, nil
		}
		if !runVerifyStep(&result, "behavior-equivalence", options.Timings, func() error {
			if snapshotTarget.Baseline.Source == "" || snapshotTarget.Baseline.Header == "" {
				return fmt.Errorf("baseline.source and baseline.header are required by profile full")
			}
			baselineOutput, baselineErr := compileAndRunHost(
				ctx,
				runner,
				cfg.Gates.GCC,
				snapshotTarget.Baseline.Source,
				snapshotTarget,
				tempDir,
				"baseline-host",
			)
			if baselineErr != nil {
				return baselineErr
			}
			if !bytes.Equal(baselineOutput, candidateOutput) {
				return fmt.Errorf("baseline and candidate host output differ")
			}
			return nil
		}) {
			finish()
			return result, nil
		}
	}

	result.OK = true
	finish()
	return result, nil
}

func runVerifyStep(
	result *VerifyResult,
	id string,
	timings bool,
	operation func() error,
) bool {
	started := time.Now()
	err := operation()
	step := VerifyStepResult{ID: id, Status: "pass"}
	if timings {
		duration := durationMilliseconds(time.Since(started))
		step.DurationMS = &duration
	}
	if err != nil {
		step.Status = "fail"
		step.Message = err.Error()
	}
	result.Steps = append(result.Steps, step)
	return err == nil
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func loadVerifyTarget(path string) (verifyTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return verifyTarget{}, err
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return verifyTarget{}, fmt.Errorf("invalid verify target: %w", err)
	}
	var target verifyTarget
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&target); err != nil {
		return verifyTarget{}, fmt.Errorf("invalid verify target: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return verifyTarget{}, fmt.Errorf("invalid verify target: multiple JSON values")
		}
		return verifyTarget{}, fmt.Errorf("invalid verify target: %w", err)
	}
	if target.Schema != verifyTargetSchema {
		return verifyTarget{}, fmt.Errorf("verify target schema must be %s", verifyTargetSchema)
	}
	if strings.TrimSpace(target.ID) == "" {
		return verifyTarget{}, fmt.Errorf("verify target id is required")
	}
	if err := validateTargetFileName(target.Candidate.Source, ".c", "candidate.source"); err != nil {
		return verifyTarget{}, err
	}
	if err := validateTargetFileName(target.Candidate.Header, ".h", "candidate.header"); err != nil {
		return verifyTarget{}, err
	}
	if target.Baseline.Source != "" {
		if err := validateTargetFileName(target.Baseline.Source, ".c", "baseline.source"); err != nil {
			return verifyTarget{}, err
		}
		if err := validateTargetFileName(target.Baseline.Header, ".h", "baseline.header"); err != nil {
			return verifyTarget{}, err
		}
	} else if target.Baseline.Header != "" {
		return verifyTarget{}, fmt.Errorf("baseline.header requires baseline.source")
	}
	if target.Host.TestSource != "" {
		if err := validateTargetFileName(target.Host.TestSource, ".c", "host.testSource"); err != nil {
			return verifyTarget{}, err
		}
	}
	for _, source := range target.Host.SupportSources {
		if err := validateTargetFileName(source, ".c", "host.supportSources"); err != nil {
			return verifyTarget{}, err
		}
	}
	for _, header := range target.Host.SupportHeaders {
		if err := validateTargetFileName(header, ".h", "host.supportHeaders"); err != nil {
			return verifyTarget{}, err
		}
	}
	for _, define := range target.Host.Defines {
		if strings.TrimSpace(define) == "" || strings.ContainsAny(define, "\x00\r\n") {
			return verifyTarget{}, fmt.Errorf("host.defines contains an invalid value")
		}
	}

	baseDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return verifyTarget{}, err
	}
	target.Candidate.Source, err = resolveTargetPath(baseDir, target.Candidate.Source)
	if err != nil {
		return verifyTarget{}, err
	}
	target.Candidate.Header, err = resolveTargetPath(baseDir, target.Candidate.Header)
	if err != nil {
		return verifyTarget{}, err
	}
	if target.Baseline.Source != "" {
		target.Baseline.Source, err = resolveTargetPath(baseDir, target.Baseline.Source)
		if err != nil {
			return verifyTarget{}, err
		}
		target.Baseline.Header, err = resolveTargetPath(baseDir, target.Baseline.Header)
		if err != nil {
			return verifyTarget{}, err
		}
	}
	if target.Host.TestSource != "" {
		target.Host.TestSource, err = resolveTargetPath(baseDir, target.Host.TestSource)
		if err != nil {
			return verifyTarget{}, err
		}
	}
	for index, source := range target.Host.SupportSources {
		target.Host.SupportSources[index], err = resolveTargetPath(baseDir, source)
		if err != nil {
			return verifyTarget{}, err
		}
	}
	for index, header := range target.Host.SupportHeaders {
		target.Host.SupportHeaders[index], err = resolveTargetPath(baseDir, header)
		if err != nil {
			return verifyTarget{}, err
		}
	}
	return target, nil
}

func validateTargetFileName(path, extension, field string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if strings.ContainsAny(path, "\x00\r\n") ||
		!strings.EqualFold(filepath.Ext(path), extension) {
		return fmt.Errorf("%s must name a %s file", field, extension)
	}
	return nil
}

func resolveTargetPath(baseDir, path string) (string, error) {
	if strings.ContainsAny(path, "\x00\r\n") {
		return "", fmt.Errorf("target path contains an invalid character")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return filepath.Abs(filepath.Clean(path))
}

func collectVerifyFiles(
	target verifyTarget,
	cfg Config,
	tempDir string,
) ([]VerifyFileReport, map[string][]byte, verifyTarget, error) {
	type fileInput struct {
		role string
		path string
	}
	inputs := []fileInput{
		{role: "candidate-source", path: target.Candidate.Source},
		{role: "candidate-header", path: target.Candidate.Header},
	}
	if target.Baseline.Source != "" {
		inputs = append(inputs,
			fileInput{role: "baseline-source", path: target.Baseline.Source},
			fileInput{role: "baseline-header", path: target.Baseline.Header},
		)
	}
	if target.Host.TestSource != "" {
		inputs = append(inputs, fileInput{role: "host-test", path: target.Host.TestSource})
	}
	for _, source := range target.Host.SupportSources {
		inputs = append(inputs, fileInput{role: "host-support", path: source})
	}
	for _, header := range target.Host.SupportHeaders {
		inputs = append(inputs, fileInput{role: "host-support-header", path: header})
	}

	snapshotDir := filepath.Join(tempDir, "snapshot")
	if err := os.MkdirAll(snapshotDir, 0o700); err != nil {
		return nil, nil, verifyTarget{}, fmt.Errorf("create source snapshot directory: %w", err)
	}

	reports := make([]VerifyFileReport, 0, len(inputs))
	dataByPath := make(map[string][]byte, len(inputs))
	snapshotByPath := make(map[string]string, len(inputs))
	basenameOwners := make(map[string]string, len(inputs))
	for _, input := range inputs {
		if isExcluded(input.path, cfg) {
			return reports, dataByPath, verifyTarget{},
				fmt.Errorf("explicit target is excluded by config: %s", input.path)
		}
		data, err := os.ReadFile(input.path)
		if err != nil {
			return reports, dataByPath, verifyTarget{}, fmt.Errorf("read %s: %w", input.path, err)
		}
		digest := sha256.Sum256(data)
		reports = append(reports, VerifyFileReport{
			Role:    input.role,
			Path:    filepath.ToSlash(input.path),
			SHA256:  fmt.Sprintf("%X", digest),
			Managed: true,
		})
		dataByPath[input.path] = data

		pathKey := strings.ToLower(filepath.Clean(input.path))
		if _, exists := snapshotByPath[pathKey]; exists {
			continue
		}
		basename := filepath.Base(input.path)
		basenameKey := strings.ToLower(basename)
		if owner, collision := basenameOwners[basenameKey]; collision && owner != pathKey {
			return reports, dataByPath, verifyTarget{}, fmt.Errorf(
				"snapshot basename collision for %s and %s; rename one manifest input",
				input.path,
				owner,
			)
		}
		basenameOwners[basenameKey] = pathKey
		snapshotPath := filepath.Join(snapshotDir, basename)
		if err := os.WriteFile(snapshotPath, data, 0o444); err != nil {
			return reports, dataByPath, verifyTarget{},
				fmt.Errorf("write source snapshot %s: %w", snapshotPath, err)
		}
		snapshotByPath[pathKey] = snapshotPath
	}

	snapshotPath := func(original string) string {
		if original == "" {
			return ""
		}
		return snapshotByPath[strings.ToLower(filepath.Clean(original))]
	}
	snapshotTarget := target
	snapshotTarget.Candidate.Source = snapshotPath(target.Candidate.Source)
	snapshotTarget.Candidate.Header = snapshotPath(target.Candidate.Header)
	snapshotTarget.Baseline.Source = snapshotPath(target.Baseline.Source)
	snapshotTarget.Baseline.Header = snapshotPath(target.Baseline.Header)
	snapshotTarget.Host.TestSource = snapshotPath(target.Host.TestSource)
	snapshotTarget.Host.SupportSources = append([]string{}, target.Host.SupportSources...)
	for index, source := range target.Host.SupportSources {
		snapshotTarget.Host.SupportSources[index] = snapshotPath(source)
	}
	snapshotTarget.Host.SupportHeaders = append([]string{}, target.Host.SupportHeaders...)
	for index, header := range target.Host.SupportHeaders {
		snapshotTarget.Host.SupportHeaders[index] = snapshotPath(header)
	}
	return reports, dataByPath, snapshotTarget, nil
}

func emitVerifyResult(result VerifyResult, jsonOut, timings bool) error {
	if jsonOut {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(result)
	}
	for _, step := range result.Steps {
		line := fmt.Sprintf("%s %s", strings.ToUpper(step.Status), step.ID)
		if timings && step.DurationMS != nil {
			line += fmt.Sprintf(" %.3fms", *step.DurationMS)
		}
		if step.Message != "" {
			line += ": " + step.Message
		}
		fmt.Println(line)
	}
	if result.OK {
		fmt.Println("C UserStyle external verification passed")
	}
	return nil
}
