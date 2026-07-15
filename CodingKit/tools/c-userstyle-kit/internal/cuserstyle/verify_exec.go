package cuserstyle

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const verifyCommandOutputLimit = 4096

type verifyCommandRunner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, cwd string, executable string, args ...string) ([]byte, error)
}

type osVerifyRunner struct{}

func (osVerifyRunner) LookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("required host compiler %s was not found: %w", name, err)
	}
	return path, nil
}

func (osVerifyRunner) Run(
	ctx context.Context,
	cwd string,
	executable string,
	args ...string,
) ([]byte, error) {
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = cwd
	command.Env = sanitizedVerifyEnvironment(os.Environ())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if ctx.Err() != nil {
		return append(stdout.Bytes(), stderr.Bytes()...),
			fmt.Errorf("command timed out: %s", filepath.Base(executable))
	}
	if err != nil {
		return append(stdout.Bytes(), stderr.Bytes()...), err
	}
	return stdout.Bytes(), nil
}

func sanitizedVerifyEnvironment(environ []string) []string {
	blocked := map[string]struct{}{
		"CCC_ADD_ARGS":                 {},
		"CCC_OVERRIDE_OPTIONS":         {},
		"CLANG_CONFIG_FILE_SYSTEM_DIR": {},
		"CLANG_CONFIG_FILE_USER_DIR":   {},
		"COLLECT_GCC_OPTIONS":          {},
		"COMPILER_PATH":                {},
		"CPATH":                        {},
		"CPLUS_INCLUDE_PATH":           {},
		"C_INCLUDE_PATH":               {},
		"DEPENDENCIES_OUTPUT":          {},
		"DYLD_INSERT_LIBRARIES":        {},
		"GCC_EXEC_PREFIX":              {},
		"GCC_SPECS":                    {},
		"LD_PRELOAD":                   {},
		"LIBRARY_PATH":                 {},
		"OBJC_INCLUDE_PATH":            {},
		"SUNPRO_DEPENDENCIES":          {},
	}
	result := make([]string, 0, len(environ))
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if _, denied := blocked[strings.ToUpper(name)]; denied {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func verifySourceSyntax(
	ctx context.Context,
	runner verifyCommandRunner,
	tool string,
	profile GateProfile,
	target verifyTarget,
	tempDir string,
	extraArgs []string,
) error {
	toolPath, err := runner.LookPath(tool)
	if err != nil {
		return err
	}
	return verifySourceSyntaxWithPath(ctx, runner, toolPath, profile, target, tempDir, extraArgs)
}

func verifySourceSyntaxWithPath(
	ctx context.Context,
	runner verifyCommandRunner,
	toolPath string,
	profile GateProfile,
	target verifyTarget,
	tempDir string,
	extraArgs []string,
) error {
	args := append([]string{}, extraArgs...)
	args = append(args, profile.Flags...)
	args = append(args, "-fsyntax-only")
	args = append(args, compileEnvironmentArgs(target, target.Candidate.Source)...)
	args = append(args, target.Candidate.Source)
	_, err := runVerifyCommand(ctx, runner, tempDir, toolPath, args...)
	return err
}

func verifyHeader(
	ctx context.Context,
	runner verifyCommandRunner,
	tool string,
	profile GateProfile,
	target verifyTarget,
	tempDir string,
	language string,
	extraArgs []string,
) error {
	toolPath, err := runner.LookPath(tool)
	if err != nil {
		return err
	}
	return verifyHeaderWithPath(
		ctx,
		runner,
		toolPath,
		profile,
		target,
		tempDir,
		language,
		extraArgs,
	)
}

func verifyHeaderWithPath(
	ctx context.Context,
	runner verifyCommandRunner,
	toolPath string,
	profile GateProfile,
	target verifyTarget,
	tempDir string,
	language string,
	extraArgs []string,
) error {
	extension := ".c"
	if language == "c++" {
		extension = ".cpp"
	}
	probePath := filepath.Join(tempDir, "header-probe"+extension)
	if err := os.WriteFile(probePath, []byte("\n"), 0o600); err != nil {
		return fmt.Errorf("write header probe: %w", err)
	}

	args := append([]string{}, extraArgs...)
	args = append(args, profile.Flags...)
	args = append(args, "-x", language, "-fsyntax-only")
	args = append(args, compileEnvironmentArgs(target, target.Candidate.Header)...)
	args = append(args, "-include", target.Candidate.Header, probePath)
	_, err := runVerifyCommand(ctx, runner, tempDir, toolPath, args...)
	return err
}

func compileAndRunHost(
	ctx context.Context,
	runner verifyCommandRunner,
	profile GateProfile,
	moduleSource string,
	target verifyTarget,
	tempDir string,
	outputName string,
) ([]byte, error) {
	if target.Host.TestSource == "" {
		return nil, fmt.Errorf("host.testSource is required")
	}
	gccPath, err := runner.LookPath("gcc")
	if err != nil {
		return nil, err
	}
	executableName := outputName
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}
	executablePath := filepath.Join(tempDir, executableName)

	args := append([]string{}, profile.Flags...)
	args = append(args, compileEnvironmentArgs(target, moduleSource)...)
	args = append(args, moduleSource)
	args = append(args, target.Host.SupportSources...)
	args = append(args, target.Host.TestSource, "-o", executablePath)
	if _, err := runVerifyCommand(ctx, runner, tempDir, gccPath, args...); err != nil {
		return nil, err
	}
	return runVerifyCommand(ctx, runner, tempDir, executablePath)
}

func compileEnvironmentArgs(target verifyTarget, moduleSource string) []string {
	directories := []string{
		filepath.Dir(moduleSource),
		filepath.Dir(target.Candidate.Header),
		filepath.Dir(target.Baseline.Header),
		filepath.Dir(target.Host.TestSource),
	}
	for _, source := range target.Host.SupportSources {
		directories = append(directories, filepath.Dir(source))
	}
	for _, header := range target.Host.SupportHeaders {
		directories = append(directories, filepath.Dir(header))
	}

	args := make([]string, 0, (len(directories)*2)+len(target.Host.Defines))
	seen := make(map[string]struct{}, len(directories))
	for _, directory := range directories {
		if directory == "" {
			continue
		}
		clean := filepath.Clean(directory)
		key := strings.ToLower(clean)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		args = append(args, "-I", clean)
	}
	for _, define := range target.Host.Defines {
		args = append(args, "-D"+define)
	}
	return args
}

func resolveClangHost(
	ctx context.Context,
	runner verifyCommandRunner,
	tool string,
	tempDir string,
) (string, []string, error) {
	toolPath, err := runner.LookPath(tool)
	if err != nil {
		return "", nil, err
	}
	version, err := runVerifyCommand(ctx, runner, tempDir, toolPath, "--version")
	if err != nil {
		return "", nil, err
	}
	if !strings.Contains(strings.ToLower(string(version)), "target: riscv") {
		return toolPath, nil, nil
	}

	const mingwSysroot = "C:/msys64/ucrt64"
	info, err := os.Stat(mingwSysroot)
	if err != nil || !info.IsDir() {
		return "", nil, fmt.Errorf(
			"%s targets RISC-V and host sysroot %s is unavailable",
			tool,
			mingwSysroot,
		)
	}
	return toolPath, []string{
		"--target=x86_64-w64-windows-gnu",
		"--sysroot=" + mingwSysroot,
	}, nil
}

func runVerifyCommand(
	ctx context.Context,
	runner verifyCommandRunner,
	cwd string,
	executable string,
	args ...string,
) ([]byte, error) {
	output, err := runner.Run(ctx, cwd, executable, args...)
	if err == nil {
		return output, nil
	}
	message := strings.TrimSpace(string(output))
	if len(message) > verifyCommandOutputLimit {
		message = message[:verifyCommandOutputLimit] + "..."
	}
	if message == "" {
		return output, fmt.Errorf("%s failed: %w", filepath.Base(executable), err)
	}
	return output, fmt.Errorf("%s failed: %w: %s", filepath.Base(executable), err, message)
}
