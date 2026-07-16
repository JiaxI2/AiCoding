package mcpcontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type installState struct {
	SchemaVersion int       `json:"schemaVersion"`
	ComponentID   string    `json:"componentId"`
	Version       string    `json:"version"`
	InstalledAt   time.Time `json:"installedAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	CodexConfig   string    `json:"codexConfig"`
}

func Status(repo, codexPath string, entries []RegistryEntry) []StatusResult {
	configPath, configErr := ResolveCodexConfig(codexPath)
	results := make([]StatusResult, 0, len(entries))
	for _, entry := range entries {
		component, err := LoadComponent(repo, entry.Manifest)
		if err != nil {
			results = append(results, StatusResult{ID: entry.ID, OK: false, Errors: []string{err.Error()}})
			continue
		}
		root := componentRoot(repo, component)
		venv := venvPython(root)
		state := statePath(repo, component.ID)
		result := StatusResult{
			ID:              component.ID,
			OK:              true,
			Root:            root,
			RootExists:      platform.IsDir(root),
			VenvPython:      venv,
			Installed:       platform.IsFile(venv),
			CodexConfigPath: configPath,
			StatePath:       state,
			StateExists:     platform.IsFile(state),
		}
		if configErr != nil {
			result.OK = false
			result.Errors = append(result.Errors, configErr.Error())
		} else {
			managed, collision, readErr := managedBlockStatus(configPath, component.Codex.ServerName)
			if readErr != nil {
				result.OK = false
				result.Errors = append(result.Errors, readErr.Error())
			}
			result.Registered = managed
			result.UnmanagedCollision = collision
			if collision {
				result.OK = false
				result.Errors = append(result.Errors, "same-name unmanaged Codex MCP entry exists")
			}
		}
		if !result.RootExists {
			result.OK = false
			result.Errors = append(result.Errors, "component root is missing")
		}
		if result.Installed != result.StateExists {
			result.Warnings = append(result.Warnings, "venv and install state are inconsistent")
		}
		results = append(results, result)
	}
	return results
}

func RunLifecycle(repo, codexPath string, entries []RegistryEntry, action string, dryRun bool) []LifecycleResult {
	results := make([]LifecycleResult, 0, len(entries))
	for _, entry := range entries {
		component, err := LoadComponent(repo, entry.Manifest)
		if err != nil {
			results = append(results, LifecycleResult{
				ID:     entry.ID,
				Action: action,
				DryRun: dryRun,
				OK:     false,
				Status: "failed",
				Errors: []string{err.Error()},
			})
			continue
		}
		results = append(results, runLifecycle(repo, codexPath, component, action, dryRun))
	}
	return results
}

func runLifecycle(repo, codexPath string, component Component, action string, dryRun bool) LifecycleResult {
	result := LifecycleResult{
		ID:        component.ID,
		Action:    action,
		DryRun:    dryRun,
		OK:        false,
		Status:    "failed",
		StatePath: statePath(repo, component.ID),
	}
	configPath, err := ResolveCodexConfig(codexPath)
	if err != nil {
		result.Errors = []string{err.Error()}
		return result
	}
	result.CodexConfigPath = configPath
	root := componentRoot(repo, component)
	result.VenvPython = venvPython(root)
	if !platform.IsDir(root) {
		result.Errors = []string{"component root is missing: " + root}
		return result
	}
	switch strings.ToLower(action) {
	case "install", "update":
		if dryRun {
			result.OK = true
			result.Status = "planned"
			return result
		}
		python, findErr := findPython(component.Runtime.MinimumPython, component.Runtime.PythonEnvVar)
		if findErr != nil {
			result.Errors = []string{findErr.Error()}
			return result
		}
		result.Python = python
		if !platform.IsFile(result.VenvPython) {
			if _, runErr := runNative(root, python, "-m", "venv", ".venv"); runErr != nil {
				result.Errors = []string{"create venv: " + runErr.Error()}
				return result
			}
		}
		if _, runErr := runNative(root, result.VenvPython, "-m", "pip", "install", "-r", component.Runtime.Requirements); runErr != nil {
			result.Errors = []string{"install requirements: " + runErr.Error()}
			return result
		}
		installArgs := []string{"-m", "pip", "install"}
		installArgs = append(installArgs, component.Runtime.PackageInstall...)
		if _, runErr := runNative(root, result.VenvPython, installArgs...); runErr != nil {
			result.Errors = []string{"install component package: " + runErr.Error()}
			return result
		}
		backup, configErr := writeManagedBlock(configPath, component, result.VenvPython, root)
		if configErr != nil {
			result.Errors = []string{configErr.Error()}
			return result
		}
		result.BackupPath = backup
		if stateErr := writeInstallState(result.StatePath, component, configPath); stateErr != nil {
			result.Errors = []string{stateErr.Error()}
			return result
		}
		result.OK = true
		result.Status = "installed"
		if strings.EqualFold(action, "update") {
			result.Status = "updated"
		}
		return result
	case "uninstall":
		if dryRun {
			result.OK = true
			result.Status = "planned"
			return result
		}
		stagedVenv, stageErr := stageOwnedVenv(root)
		if stageErr != nil {
			result.Errors = []string{stageErr.Error()}
			return result
		}
		backup, configErr := removeManagedBlock(configPath, component.Codex.ServerName)
		if configErr != nil {
			if restoreErr := restoreStagedVenv(root, stagedVenv); restoreErr != nil {
				result.Errors = []string{configErr.Error(), "restore staged venv: " + restoreErr.Error()}
				return result
			}
			result.Errors = []string{configErr.Error()}
			return result
		}
		result.BackupPath = backup
		if removeErr := removeStagedVenv(root, stagedVenv); removeErr != nil {
			result.Errors = []string{removeErr.Error()}
			if restoreErr := restoreConfigBackup(configPath, backup); restoreErr != nil {
				result.Errors = append(result.Errors, "restore Codex config: "+restoreErr.Error())
			}
			if restoreErr := restoreStagedVenv(root, stagedVenv); restoreErr != nil {
				result.Errors = append(result.Errors, "restore staged venv: "+restoreErr.Error())
			}
			return result
		}
		_ = os.RemoveAll(filepath.Dir(result.StatePath))
		result.OK = true
		result.Status = "uninstalled"
		return result
	default:
		result.Errors = []string{"unsupported MCP lifecycle action: " + action}
		return result
	}
}

func componentRoot(repo string, component Component) string {
	return platform.RepoPath(repo, component.Runtime.Root)
}

func venvPython(root string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(root, ".venv", "Scripts", "python.exe")
	}
	return filepath.Join(root, ".venv", "bin", "python")
}

func statePath(repo, id string) string {
	return platform.RepoPath(repo, filepath.ToSlash(filepath.Join(".aicoding", "state", "mcp", id, "install-state.json")))
}

func findPython(minimum, overrideEnvVar string) (string, error) {
	candidates := []string{}
	if value := os.Getenv(overrideEnvVar); overrideEnvVar != "" && value != "" {
		candidates = append(candidates, value)
	}
	if value, err := exec.LookPath("python"); err == nil {
		candidates = append(candidates, value)
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			matches, _ := filepath.Glob(filepath.Join(local, "Programs", "Python", "Python3*", "python.exe"))
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			candidates = append(candidates, matches...)
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil || seen[strings.ToLower(absolute)] || !platform.IsFile(absolute) {
			continue
		}
		seen[strings.ToLower(absolute)] = true
		output, err := exec.Command(absolute, "-c", "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')").CombinedOutput()
		if err != nil {
			continue
		}
		version := strings.TrimSpace(string(output))
		if versionAtLeast(version, minimum) {
			return absolute, nil
		}
	}
	return "", fmt.Errorf("Python %s or newer was not found", minimum)
}

func versionAtLeast(actual, minimum string) bool {
	parse := func(value string) (int, int, bool) {
		parts := strings.Split(value, ".")
		if len(parts) < 2 {
			return 0, 0, false
		}
		major, err1 := strconv.Atoi(parts[0])
		minor, err2 := strconv.Atoi(parts[1])
		return major, minor, err1 == nil && err2 == nil
	}
	actualMajor, actualMinor, actualOK := parse(actual)
	minimumMajor, minimumMinor, minimumOK := parse(minimum)
	if !actualOK || !minimumOK {
		return false
	}
	return actualMajor > minimumMajor || (actualMajor == minimumMajor && actualMinor >= minimumMinor)
}

func runNative(dir, executable string, args ...string) (string, error) {
	command := exec.Command(executable, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text != "" {
			return text, fmt.Errorf("%w: %s", err, tail(text, 4000))
		}
		return text, err
	}
	return text, nil
}

func writeInstallState(path string, component Component, configPath string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	now := time.Now().UTC()
	state := installState{
		SchemaVersion: 1,
		ComponentID:   component.ID,
		Version:       component.Version,
		InstalledAt:   now,
		UpdatedAt:     now,
		CodexConfig:   configPath,
	}
	if data, err := os.ReadFile(path); err == nil {
		var previous installState
		if json.Unmarshal(data, &previous) == nil && !previous.InstalledAt.IsZero() {
			state.InstalledAt = previous.InstalledAt
		}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func managedBlockStatus(configPath, serverName string) (bool, bool, error) {
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	text := string(data)
	begin, end := managedMarkers(serverName)
	managed := strings.Contains(text, begin) && strings.Contains(text, end)
	section := "[mcp_servers." + serverName + "]"
	collision := strings.Contains(text, section) && !managed
	return managed, collision, nil
}

func writeManagedBlock(configPath string, component Component, python, root string) (string, error) {
	data, mode, err := readConfig(configPath)
	if err != nil {
		return "", err
	}
	text := string(data)
	begin, end := managedMarkers(component.Codex.ServerName)
	start := strings.Index(text, begin)
	finish := strings.Index(text, end)
	if (start >= 0) != (finish >= 0) {
		return "", errors.New("managed MCP config markers are incomplete")
	}
	section := "[mcp_servers." + component.Codex.ServerName + "]"
	if start < 0 && strings.Contains(text, section) {
		return "", errors.New("same-name unmanaged Codex MCP entry exists; refusing to overwrite")
	}
	block := renderManagedBlock(component, python, root)
	updated := ""
	if start >= 0 {
		finish += len(end)
		for finish < len(text) && (text[finish] == '\r' || text[finish] == '\n') {
			finish++
		}
		updated = text[:start] + block + text[finish:]
	} else {
		separator := ""
		if text != "" && !strings.HasSuffix(text, "\n") {
			separator = "\n"
		}
		if text != "" {
			separator += "\n"
		}
		updated = text + separator + block
	}
	return writeConfigWithBackup(configPath, data, []byte(updated), mode)
}

func removeManagedBlock(configPath, serverName string) (string, error) {
	data, mode, err := readConfig(configPath)
	if err != nil {
		return "", err
	}
	text := string(data)
	begin, end := managedMarkers(serverName)
	start := strings.Index(text, begin)
	finish := strings.Index(text, end)
	section := "[mcp_servers." + serverName + "]"
	if start < 0 || finish < 0 {
		if strings.Contains(text, section) {
			return "", errors.New("same-name unmanaged Codex MCP entry exists; refusing to remove")
		}
		return "", nil
	}
	finish += len(end)
	for finish < len(text) && (text[finish] == '\r' || text[finish] == '\n') {
		finish++
	}
	updated := strings.TrimRight(text[:start]+text[finish:], "\r\n")
	if updated != "" {
		updated += "\n"
	}
	return writeConfigWithBackup(configPath, data, []byte(updated), mode)
}

func managedMarkers(serverName string) (string, string) {
	return "# BEGIN AICODING MCP " + serverName, "# END AICODING MCP " + serverName
}

func renderManagedBlock(component Component, python, root string) string {
	begin, end := managedMarkers(component.Codex.ServerName)
	args := make([]string, 0, len(component.Runtime.ServerArgs))
	for _, arg := range component.Runtime.ServerArgs {
		args = append(args, strconv.Quote(arg))
	}
	envBlock := ""
	if len(component.Runtime.Env) > 0 {
		keys := make([]string, 0, len(component.Runtime.Env))
		for key := range component.Runtime.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines := []string{"", "[mcp_servers." + component.Codex.ServerName + ".env]"}
		for _, key := range keys {
			value := strings.ReplaceAll(component.Runtime.Env[key], "${componentRoot}", root)
			lines = append(lines, key+" = "+strconv.Quote(value))
		}
		envBlock = strings.Join(lines, "\n") + "\n"
	}
	return fmt.Sprintf(
		"%s\n[mcp_servers.%s]\ncommand = %s\nargs = [%s]\ncwd = %s\nstartup_timeout_sec = %d\ntool_timeout_sec = %d\n%s%s\n",
		begin,
		component.Codex.ServerName,
		strconv.Quote(python),
		strings.Join(args, ", "),
		strconv.Quote(root),
		component.Codex.StartupTimeoutSec,
		component.Codex.ToolTimeoutSec,
		envBlock,
		end,
	)
}

func readConfig(path string) ([]byte, os.FileMode, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []byte{}, 0o600, nil
	}
	if err != nil {
		return nil, 0, err
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return data, mode, nil
}

func writeConfigWithBackup(path string, original, updated []byte, mode os.FileMode) (string, error) {
	if string(original) == string(updated) {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	backup := ""
	if len(original) > 0 {
		backup = fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405.000000000"))
		if err := os.WriteFile(backup, original, mode); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(path, updated, mode); err != nil {
		return backup, err
	}
	return backup, nil
}

func stageOwnedVenv(root string) (string, error) {
	target := filepath.Join(root, ".venv")
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if filepath.Dir(absoluteTarget) != absoluteRoot || filepath.Base(absoluteTarget) != ".venv" {
		return "", errors.New("refusing to stage venv outside component root")
	}
	if _, err := os.Stat(absoluteTarget); os.IsNotExist(err) {
		return "", nil
	}
	staged := filepath.Join(absoluteRoot, ".venv.uninstalling")
	if platform.Exists(staged) {
		return "", errors.New("staged venv already exists: " + staged)
	}
	if err := os.Rename(absoluteTarget, staged); err != nil {
		return "", fmt.Errorf("stage venv for uninstall; stop active component processes and retry: %w", err)
	}
	return staged, nil
}

func restoreStagedVenv(root, staged string) error {
	if staged == "" {
		return nil
	}
	target := filepath.Join(root, ".venv")
	if platform.Exists(target) {
		return errors.New("cannot restore staged venv because .venv already exists")
	}
	return os.Rename(staged, target)
}

func removeStagedVenv(root, staged string) error {
	if staged == "" {
		return nil
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absoluteStaged, err := filepath.Abs(staged)
	if err != nil {
		return err
	}
	if filepath.Dir(absoluteStaged) != absoluteRoot || filepath.Base(absoluteStaged) != ".venv.uninstalling" {
		return errors.New("refusing to remove staged venv outside component root")
	}
	return os.RemoveAll(absoluteStaged)
}

func restoreConfigBackup(configPath, backup string) error {
	if backup == "" {
		return nil
	}
	data, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(configPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(configPath, data, mode)
}
