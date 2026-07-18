package kit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type pluginBuildInfo struct {
	PluginName         string `json:"pluginName"`
	PluginVersion      string `json:"pluginVersion"`
	SourceCommit       string `json:"sourceCommit"`
	SourceTag          string `json:"sourceTag"`
	PackManifestHash   string `json:"packManifestHash"`
	PluginManifestHash string `json:"pluginManifestHash"`
	SkillsDigest       string `json:"skillsDigest"`
	HooksDigest        string `json:"hooksDigest"`
	DirtySource        bool   `json:"dirtySource"`
}

type pluginManifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type marketplaceManifest struct {
	Name    string `json:"name"`
	Plugins []struct {
		Name string `json:"name"`
	} `json:"plugins"`
}

type PlatformPluginSync struct {
	PluginID             string           `json:"pluginId"`
	Marketplace          string           `json:"marketplace"`
	PluginName           string           `json:"pluginName"`
	PluginVersion        string           `json:"pluginVersion"`
	SourcePackage        string           `json:"sourcePackage"`
	InstalledPackage     string           `json:"installedPackage"`
	SourceBuildInfo      pluginBuildInfo  `json:"sourceBuildInfo"`
	InstalledBuildInfo   *pluginBuildInfo `json:"installedBuildInfo,omitempty"`
	Installed            bool             `json:"installed"`
	Enabled              *bool            `json:"enabled,omitempty"`
	Drift                bool             `json:"drift"`
	DriftReasons         []string         `json:"driftReasons,omitempty"`
	Refreshed            bool             `json:"refreshed"`
	ManualRequired       bool             `json:"manualRequired"`
	CLIPath              string           `json:"cliPath,omitempty"`
	Commands             []string         `json:"commands,omitempty"`
	ManualCommand        string           `json:"manualCommand"`
	DeepLink             string           `json:"deepLink"`
	RepositoryLink       string           `json:"repositoryLink,omitempty"`
	RepositoryLinkStatus string           `json:"repositoryLinkStatus,omitempty"`
}

var codexLookPath = exec.LookPath

var codexPluginCommand = func(path string, args ...string) ([]byte, error) {
	cmd := exec.Command(path, args...)
	return cmd.CombinedOutput()
}

func inspectPlatformPlugin(repo string, manifest Manifest) (PlatformPluginSync, error) {
	result := PlatformPluginSync{}
	pluginRoot := manifest.Paths["pluginRoot"]
	if pluginRoot == "" {
		return result, errors.New("aicoding-platform missing paths.pluginRoot")
	}
	result.SourcePackage = platform.RepoPath(repo, pluginRoot)
	if !platform.IsDir(result.SourcePackage) {
		return result, fmt.Errorf("missing plugin package: %s", pluginRoot)
	}

	var plugin pluginManifest
	if err := readPluginJSON(filepath.Join(result.SourcePackage, ".codex-plugin", "plugin.json"), &plugin); err != nil {
		return result, fmt.Errorf("read plugin manifest: %w", err)
	}
	if plugin.Name == "" || plugin.Version == "" {
		return result, errors.New("plugin manifest requires name and version")
	}

	marketplaceRel := manifest.Paths["marketplace"]
	if marketplaceRel == "" {
		return result, errors.New("aicoding-platform missing paths.marketplace")
	}
	marketplacePath := platform.RepoPath(repo, marketplaceRel)
	var marketplace marketplaceManifest
	if err := readPluginJSON(marketplacePath, &marketplace); err != nil {
		return result, fmt.Errorf("read marketplace manifest: %w", err)
	}
	if marketplace.Name == "" {
		return result, errors.New("marketplace manifest requires name")
	}
	foundPlugin := false
	for _, entry := range marketplace.Plugins {
		if entry.Name == plugin.Name {
			foundPlugin = true
			break
		}
	}
	if !foundPlugin {
		return result, fmt.Errorf("marketplace %s does not contain plugin %s", marketplace.Name, plugin.Name)
	}

	if err := readPluginJSON(filepath.Join(result.SourcePackage, "BUILDINFO.json"), &result.SourceBuildInfo); err != nil {
		return result, fmt.Errorf("read source plugin BUILDINFO.json: %w", err)
	}
	if result.SourceBuildInfo.PluginName != plugin.Name || result.SourceBuildInfo.PluginVersion != plugin.Version {
		return result, errors.New("source plugin BUILDINFO.json does not match plugin manifest identity")
	}
	if result.SourceBuildInfo.DirtySource {
		return result, errors.New("source plugin BUILDINFO.json reports dirtySource=true")
	}

	codexHome, err := resolveCodexHome()
	if err != nil {
		return result, err
	}
	result.Marketplace = marketplace.Name
	result.PluginName = plugin.Name
	result.PluginVersion = plugin.Version
	result.PluginID = plugin.Name + "@" + marketplace.Name
	result.InstalledPackage = filepath.Join(codexHome, "plugins", "cache", marketplace.Name, plugin.Name, plugin.Version)
	result.ManualCommand = "codex plugin add " + result.PluginID + " --json"
	result.DeepLink = "codex://plugins/?marketplacePath=" + url.QueryEscape(marketplacePath)

	enabled, configured, err := readPluginEnabled(filepath.Join(codexHome, "config.toml"), result.PluginID)
	if err != nil {
		return result, err
	}
	if configured {
		result.Enabled = &enabled
	}

	installedBuildInfoPath := filepath.Join(result.InstalledPackage, "BUILDINFO.json")
	if platform.IsFile(installedBuildInfoPath) {
		var installed pluginBuildInfo
		if err := readPluginJSON(installedBuildInfoPath, &installed); err != nil {
			return result, fmt.Errorf("read installed plugin BUILDINFO.json: %w", err)
		}
		result.Installed = true
		result.InstalledBuildInfo = &installed
	} else if platform.Exists(result.InstalledPackage) {
		result.Installed = true
		result.DriftReasons = append(result.DriftReasons, "installed plugin cache is missing BUILDINFO.json")
	} else {
		result.DriftReasons = append(result.DriftReasons, "installed plugin cache is missing")
	}

	if result.InstalledBuildInfo != nil {
		result.DriftReasons = append(result.DriftReasons, comparePluginBuildInfo(result.SourceBuildInfo, *result.InstalledBuildInfo)...)
	}
	result.Drift = len(result.DriftReasons) > 0
	if path, findErr := findCodexCLI(); findErr == nil {
		result.CLIPath = path
	}
	return result, nil
}

func syncPlatformPlugin(repo string, manifest Manifest) (PlatformPluginSync, error) {
	result, err := inspectPlatformPlugin(repo, manifest)
	if err != nil {
		return result, err
	}
	if !result.Drift {
		return result, nil
	}
	if result.Enabled != nil && !*result.Enabled {
		result.ManualRequired = true
		return result, fmt.Errorf("plugin %s is disabled; refresh it manually to preserve enabled state: %s", result.PluginID, result.DeepLink)
	}
	cliPath, err := findCodexCLI()
	if err != nil {
		result.ManualRequired = true
		return result, fmt.Errorf("%w; run %q or open %s", err, result.ManualCommand, result.DeepLink)
	}
	result.CLIPath = cliPath

	if result.Enabled != nil {
		args := []string{"plugin", "remove", result.PluginID, "--json"}
		result.Commands = append(result.Commands, formatCommand(cliPath, args))
		if _, err := runCodexPluginCommand(cliPath, args...); err != nil {
			result.ManualRequired = true
			return result, err
		}
	}

	args := []string{"plugin", "add", result.PluginID, "--json"}
	result.Commands = append(result.Commands, formatCommand(cliPath, args))
	if _, err := runCodexPluginCommand(cliPath, args...); err != nil {
		result.ManualRequired = true
		return result, err
	}

	after, err := inspectPlatformPlugin(repo, manifest)
	if err != nil {
		return result, err
	}
	after.CLIPath = cliPath
	after.Commands = result.Commands
	after.Refreshed = true
	if after.Drift {
		return after, fmt.Errorf("plugin cache still drifts after refresh: %s", strings.Join(after.DriftReasons, "; "))
	}
	return after, nil
}

func configurePlatformRepository(repo string, result *PlatformPluginSync) ([]string, error) {
	if _, err := gitx.Run(repo, "config", "core.hooksPath", ".githooks"); err != nil {
		return nil, err
	}

	warnings := []string{}
	link := filepath.Join(repo, "plugins", "AiCoding")
	result.RepositoryLink = link
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return warnings, err
	}
	if info, err := os.Lstat(link); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			result.RepositoryLinkStatus = "unmanaged-existing"
			warnings = append(warnings, "repository plugin convenience link exists and was left unchanged: "+link)
			return warnings, nil
		}
		target, readErr := os.Readlink(link)
		if readErr == nil && samePath(target, result.SourcePackage) {
			result.RepositoryLinkStatus = "exists"
			return warnings, nil
		}
		if err := os.Remove(link); err != nil {
			return warnings, err
		}
	} else if !os.IsNotExist(err) {
		return warnings, err
	}
	if err := os.Symlink(result.SourcePackage, link); err != nil {
		result.RepositoryLinkStatus = "unavailable"
		warnings = append(warnings, "repository plugin convenience link could not be created: "+err.Error())
		return warnings, nil
	}
	result.RepositoryLinkStatus = "created"
	return warnings, nil
}

func comparePluginBuildInfo(source, installed pluginBuildInfo) []string {
	type field struct {
		name      string
		source    string
		installed string
	}
	fields := []field{
		{"pluginName", source.PluginName, installed.PluginName},
		{"pluginVersion", source.PluginVersion, installed.PluginVersion},
		{"sourceCommit", source.SourceCommit, installed.SourceCommit},
		{"packManifestHash", source.PackManifestHash, installed.PackManifestHash},
		{"pluginManifestHash", source.PluginManifestHash, installed.PluginManifestHash},
		{"skillsDigest", source.SkillsDigest, installed.SkillsDigest},
		{"hooksDigest", source.HooksDigest, installed.HooksDigest},
	}
	reasons := []string{}
	for _, item := range fields {
		if item.source != item.installed {
			reasons = append(reasons, item.name+" mismatch")
		}
	}
	if installed.DirtySource {
		reasons = append(reasons, "installed BUILDINFO.json reports dirtySource=true")
	}
	return reasons
}

func resolveCodexHome() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return filepath.Clean(value), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}

func findCodexCLI() (string, error) {
	if value := strings.TrimSpace(os.Getenv("AICODING_CODEX_CLI")); value != "" {
		return value, nil
	}
	localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if localAppData != "" {
		root := filepath.Join(localAppData, "OpenAI", "Codex", "bin")
		var newest string
		var newestMod int64
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !strings.EqualFold(entry.Name(), "codex.exe") {
				return nil
			}
			info, err := entry.Info()
			if err == nil && info.ModTime().UnixNano() > newestMod {
				newest = path
				newestMod = info.ModTime().UnixNano()
			}
			return nil
		})
		if newest != "" {
			return newest, nil
		}
	}
	if value, err := codexLookPath("codex"); err == nil {
		return value, nil
	}
	return "", errors.New("codex CLI with plugin commands was not found")
}

func runCodexPluginCommand(path string, args ...string) ([]byte, error) {
	output, err := codexPluginCommand(path, args...)
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return output, fmt.Errorf("%s failed: %s", formatCommand(path, args), detail)
	}
	return output, nil
}

func readPluginEnabled(path, pluginID string) (bool, bool, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read Codex config: %w", err)
	}
	defer file.Close()

	doubleSection := `[plugins."` + pluginID + `"]`
	singleSection := `[plugins.'` + pluginID + `']`
	inSection := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inSection = line == doubleSection || line == singleSection
			continue
		}
		if !inSection || !strings.HasPrefix(line, "enabled") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(parts[1])) {
		case "true":
			return true, true, nil
		case "false":
			return false, true, nil
		default:
			return false, false, fmt.Errorf("invalid enabled value for plugin %s", pluginID)
		}
	}
	if err := scanner.Err(); err != nil {
		return false, false, fmt.Errorf("read Codex config: %w", err)
	}
	return false, false, nil
}

func readPluginJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(data, target)
}

func formatCommand(path string, args []string) string {
	return strings.TrimSpace(strings.Join(append([]string{path}, args...), " "))
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
}
