package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

type commandExecutor func(context.Context, string, string, []string) ([]byte, []byte, error)

func runRuntimeSkillAdapter(ctx context.Context, repo string, opts Options, execute commandExecutor) AdapterResult {
	result := AdapterResult{
		ID:     ScopeRuntimeSkill,
		Action: opts.Action,
		DryRun: opts.DryRun,
		OK:     false,
		Status: "failed",
	}
	sourceRepository := opts.SourceRepository
	sourceExists := false
	if sourceRepository == "" {
		sourceRepository, sourceExists = resolveRuntimeSourceRepository(repo)
	} else {
		sourceRepository = absolutePath(repo, sourceRepository)
		sourceExists = isDirectory(sourceRepository)
	}
	opts.SourceRepository = sourceRepository
	if digest, digestErr := runtimeSkillInputDigest(repo, sourceRepository, sourceExists); digestErr == nil {
		result.InputDigest = digest
	} else {
		result.Warnings = append(result.Warnings, "runtime Skill input snapshot is unavailable: "+digestErr.Error())
	}
	if !sourceExists {
		warning := "runtime Skill source repository could not be resolved"
		if sourceRepository != "" {
			warning += ": " + sourceRepository
		}
		result.Warnings = append(result.Warnings, warning)
		if !opts.DryRun && (opts.Action == "install" || opts.Action == "update" || opts.Action == "uninstall") {
			result.Errors = []string{warning + "; use --source-repository"}
			return result
		}
	}
	script, scriptArgs, err := runtimeSkillCommand(repo, opts)
	if err != nil {
		result.Errors = []string{err.Error()}
		return result
	}
	executable := findPowerShell()
	args := []string{"-NoProfile"}
	if runtime.GOOS == "windows" {
		args = append(args, "-ExecutionPolicy", "Bypass")
	}
	args = append(args, "-File", script)
	args = append(args, scriptArgs...)

	stdout, stderr, runErr := execute(ctx, repo, executable, args)
	stdout = bytes.TrimSpace(bytes.TrimPrefix(stdout, []byte{0xef, 0xbb, 0xbf}))
	stderr = bytes.TrimSpace(stderr)
	if !json.Valid(stdout) {
		if len(stderr) > 0 {
			result.Errors = append(result.Errors, string(stderr))
		}
		if runErr != nil {
			result.Errors = append(result.Errors, runErr.Error())
		}
		if len(stdout) > 0 {
			result.Errors = append(result.Errors, "runtime Skill command returned invalid JSON: "+string(stdout))
		}
		if len(result.Errors) == 0 {
			result.Errors = []string{"runtime Skill command returned no JSON"}
		}
		return result
	}

	raw := append(json.RawMessage(nil), stdout...)
	result.Data = raw
	var envelope struct {
		OK       *bool    `json:"ok"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout, &envelope); err != nil {
		result.Errors = []string{"cannot decode runtime Skill result: " + err.Error()}
		return result
	}
	result.Warnings = append(result.Warnings, envelope.Warnings...)
	result.OK = runErr == nil
	if envelope.OK != nil {
		result.OK = result.OK && *envelope.OK
	}
	if !result.OK {
		if envelope.OK != nil && !*envelope.OK {
			result.Errors = append(result.Errors, "runtime Skill audit reported drift")
		}
		if len(stderr) > 0 {
			result.Errors = append(result.Errors, string(stderr))
		}
		if runErr != nil && len(result.Errors) == 0 {
			result.Errors = append(result.Errors, runErr.Error())
		}
		return result
	}
	if opts.DryRun {
		result.Status = "planned"
	} else if opts.Action == "install" || opts.Action == "update" || opts.Action == "uninstall" {
		result.Status = "applied"
	} else {
		result.Status = "ok"
	}
	return result
}

func runtimeSkillInputDigest(repo, sourceRepository string, sourceExists bool) (string, error) {
	data, err := os.ReadFile(filepath.Join(repo, "config", "codex-kit.json"))
	if err != nil {
		return "", err
	}
	var config interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}
	sourceCommit := ""
	if sourceExists {
		command := exec.Command("git", "-C", sourceRepository, "rev-parse", "HEAD")
		if output, commitErr := command.Output(); commitErr == nil {
			sourceCommit = strings.TrimSpace(string(output))
		}
	}
	snapshot, err := registryobject.NewSnapshot("runtime-skill-registry", struct {
		Config       interface{} `json:"config"`
		SourceCommit string      `json:"sourceCommit,omitempty"`
	}{Config: config, SourceCommit: sourceCommit})
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

func resolveRuntimeSourceRepository(repo string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(repo, "config", "codex-kit.json"))
	if err != nil {
		return "", false
	}
	var config struct {
		SkillRuntime struct {
			SourceRepositoryEnv     string `json:"sourceRepositoryEnv"`
			DefaultSourceRepository string `json:"defaultSourceRepository"`
		} `json:"skillRuntime"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", false
	}
	if envName := strings.TrimSpace(config.SkillRuntime.SourceRepositoryEnv); envName != "" {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			path := absolutePath(repo, value)
			return path, isDirectory(path)
		}
	}
	relative := strings.TrimSpace(config.SkillRuntime.DefaultSourceRepository)
	if relative == "" {
		return "", false
	}
	repoCandidate := absolutePath(repo, relative)
	if isDirectory(repoCandidate) {
		return repoCandidate, true
	}
	if commonRoot := gitCommonRepositoryRoot(repo); commonRoot != "" {
		commonCandidate := absolutePath(commonRoot, relative)
		if isDirectory(commonCandidate) {
			return commonCandidate, true
		}
	}
	return repoCandidate, false
}

func gitCommonRepositoryRoot(repo string) string {
	cmd := exec.Command("git", "-C", repo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	commonDir := strings.TrimSpace(string(output))
	if commonDir == "" {
		return ""
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(repo, commonDir)
	}
	return filepath.Dir(filepath.Clean(commonDir))
}

func absolutePath(root, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	path, err := filepath.Abs(filepath.Join(root, value))
	if err != nil {
		return filepath.Clean(filepath.Join(root, value))
	}
	return path
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runtimeSkillCommand(repo string, opts Options) (string, []string, error) {
	switch opts.Action {
	case "install", "update", "uninstall":
		profile := opts.RuntimeProfile
		if opts.Action == "uninstall" {
			profile = "runtime"
		}
		if profile == "" {
			return "", nil, errors.New("runtime Skill lifecycle requires --runtime-profile")
		}
		if profile == "skill-development" && strings.TrimSpace(opts.RuntimeSkill) == "" {
			return "", nil, errors.New("skill-development runtime profile requires --runtime-skill")
		}
		args := []string{"-Profile", profile, "-StandaloneRoot", opts.StandaloneRoot}
		args = appendRuntimeSelection(args, opts)
		if opts.MigrateUnmanaged {
			args = append(args, "-MigrateUnmanaged")
		}
		if opts.DryRun {
			args = append(args, "-DryRun")
		}
		args = append(args, "-Json")
		return filepath.Join(repo, "tools", "specialty", "set-codex-skill-profile.ps1"), args, nil
	case "status", "doctor", "verify":
		args := []string{"-StandaloneRoot", opts.StandaloneRoot}
		if opts.RuntimeProfile != "" {
			args = append(args, "-ExpectedProfile", opts.RuntimeProfile)
		}
		args = appendRuntimeSelection(args, opts)
		if opts.Action == "verify" {
			args = append(args, "-Strict")
		}
		args = append(args, "-Json")
		return filepath.Join(repo, "tools", "specialty", "audit-runtime-skills.ps1"), args, nil
	default:
		return "", nil, errors.New("unsupported runtime Skill lifecycle action: " + opts.Action)
	}
}

func appendRuntimeSelection(args []string, opts Options) []string {
	if opts.RuntimeSkill != "" {
		args = append(args, "-Skill", opts.RuntimeSkill)
	}
	if opts.SourceRepository != "" {
		args = append(args, "-SourceRepository", opts.SourceRepository)
	}
	return args
}

func findPowerShell() string {
	for _, name := range []string{"pwsh", "powershell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return "pwsh"
}

func defaultCommandExecutor(ctx context.Context, dir, executable string, args []string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
