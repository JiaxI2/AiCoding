// Package repoinit sets up the local AI-coding environment in a repository,
// parallel to how `git init` prepares .git/. It composes existing primitives
// (git via internal/gitx, paths via internal/platform) — it holds no business
// logic of its own beyond orchestrating an idempotent setup. See ADR 0005
// (docs/decisions/0005-repo-init.md) and docs/architecture/GRAPH_FIRST.md.
package repoinit

import (
	"fmt"
	"os"
	"path/filepath"

	provisiontemplates "github.com/JiaxI2/AiCoding/config/templates/provision"
	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

// configPrefix is the git-config namespace that holds AI-coding local markers.
// These live in .git/config: local, per-clone, never committed — the same place
// git keeps its own settings — so later commands read setup state instantly via
// `git config --get aicoding.*` without scanning the working tree.
const configPrefix = "aicoding."

// SchemaVersion of the local marker set; bumped only on a marker-shape change.
const SchemaVersion = "2"

// Marker is one aicoding.* git-config key/value that init ensures.
var markers = []struct{ Key, Value string }{
	{configPrefix + "initialized", "true"},
	{configPrefix + "home", ".aicoding"},
	{configPrefix + "schemaVersion", SchemaVersion},
	{configPrefix + "docsSkeleton", "1"},
}

var transportConfig = []struct{ Key, Value string }{
	{"fetch.parallel", "0"},
	{"submodule.fetchJobs", "4"},
	{"core.fscache", "true"},
}

var docsSkeleton = []string{
	"docs/README.md",
	"docs/architecture/README.md",
	"docs/decisions/README.md",
	"docs/spec/README.md",
	"docs/todolist/README.md",
}

// Report is the standardized, composable result of an init run.
type Report struct {
	Repo             string            `json:"repo"`
	GitInitialized   bool              `json:"gitInitialized"`   // true when init created .git this run
	GitAlreadyRepo   bool              `json:"gitAlreadyRepo"`   // true when .git already existed
	HooksPath        string            `json:"hooksPath"`        // core.hooksPath after wiring
	ConfigMarkers    map[string]string `json:"configMarkers"`    // aicoding.* keys written to .git/config
	TransportConfig  map[string]string `json:"transportConfig"`  // local Git transfer settings written to .git/config
	AicodingHomePath string            `json:"aicodingHomePath"` // .aicoding local state root
	DocsSkeleton     []string          `json:"docsSkeleton,omitempty"`
	Actions          []string          `json:"actions"` // human-readable, ordered
	OK               bool              `json:"ok"`
	Errors           []string          `json:"errors,omitempty"`
}

// Init is idempotent: it ensures a git repo, wires the repo hooks, writes the
// AI-coding markers into git's own config, ensures the .aicoding home exists,
// and places the minimal SDD documentation skeleton without overwriting files.
// Re-running changes nothing. It never writes loose files under .git/ (fragile);
// the local key/value store is git config, which git owns.
func Init(repo string) Report {
	report := Report{
		Repo: repo, ConfigMarkers: map[string]string{}, TransportConfig: map[string]string{}, AicodingHomePath: ".aicoding",
		DocsSkeleton: append([]string(nil), docsSkeleton...),
	}

	// 1) Ensure a git repository. `git init` is itself idempotent, but report which.
	if platform.Exists(platform.RepoPath(repo, ".git")) {
		report.GitAlreadyRepo = true
		report.Actions = append(report.Actions, "kept git repository")
	} else {
		if _, err := gitx.Run(repo, "init"); err != nil {
			report.Errors = append(report.Errors, "git init: "+err.Error())
			return report
		}
		report.GitInitialized = true
		report.Actions = append(report.Actions, "created git repository")
	}

	// 2) Activate the repository hooks via git's own mechanism.
	hooksState, err := ensureGitConfig(repo, "core.hooksPath", ".githooks")
	if err != nil {
		report.Errors = append(report.Errors, "wire core.hooksPath: "+err.Error())
		return report
	}
	report.HooksPath = ".githooks"
	report.Actions = append(report.Actions, hooksState+" git config core.hooksPath = .githooks")

	// Keep normal pull/fetch on Git's local fast path. Skills submodules are
	// synchronized explicitly by lifecycle install/update instead of every pull.
	for _, setting := range transportConfig {
		state, err := ensureGitConfig(repo, setting.Key, setting.Value)
		if err != nil {
			report.Errors = append(report.Errors, "set "+setting.Key+": "+err.Error())
			return report
		}
		report.TransportConfig[setting.Key] = setting.Value
		report.Actions = append(report.Actions, fmt.Sprintf("%s git config %s = %s", state, setting.Key, setting.Value))
	}

	// 3) Ensure the .aicoding local state home exists (its versioned subtrees are
	//    created on demand and gitignored; init only guarantees the root).
	home := platform.RepoPath(repo, ".aicoding")
	homeState, err := ensureDirectory(home)
	if err != nil {
		report.Errors = append(report.Errors, "ensure .aicoding home: "+err.Error())
		return report
	}
	report.Actions = append(report.Actions, homeState+" .aicoding home")

	// 4) Place the tracked documentation convention. Existing paths are owned by
	//    the repository and are never read, compared, or overwritten.
	for _, relative := range docsSkeleton {
		state, err := ensureSkeletonFile(repo, relative)
		if err != nil {
			report.Errors = append(report.Errors, "ensure "+relative+": "+err.Error())
			return report
		}
		report.Actions = append(report.Actions, state+" "+relative)
	}

	// 5) Publish local markers only after all filesystem steps succeed.
	for _, marker := range markers {
		state, err := ensureGitConfig(repo, marker.Key, marker.Value)
		if err != nil {
			report.Errors = append(report.Errors, "set "+marker.Key+": "+err.Error())
			return report
		}
		report.ConfigMarkers[marker.Key] = marker.Value
		report.Actions = append(report.Actions, fmt.Sprintf("%s git config %s = %s", state, marker.Key, marker.Value))
	}

	report.OK = true
	return report
}

func ensureDirectory(path string) (string, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("path exists and is not a directory")
		}
		return "kept", nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return "created", nil
}

func ensureSkeletonFile(repo, relative string) (string, error) {
	target := platform.RepoPath(repo, relative)
	if _, err := os.Lstat(target); err == nil {
		return "kept", nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	content, err := provisiontemplates.Files.ReadFile(filepath.ToSlash(relative) + ".tmpl")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return "kept", nil
	}
	if err != nil {
		return "", err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		_ = os.Remove(target)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	return "created", nil
}

func ensureGitConfig(repo, key, value string) (string, error) {
	current, readErr := gitx.Run(repo, "config", "--local", "--get", key)
	current = trimLine(current)
	if readErr == nil && current == value {
		return "kept", nil
	}
	if _, err := gitx.Run(repo, "config", "--local", key, value); err != nil {
		return "", err
	}
	if readErr == nil && current != "" {
		return "updated", nil
	}
	return "created", nil
}

// Status reads the AI-coding markers from git config so later commands can judge
// setup state quickly (e.g. gate on aicoding.initialized) without scanning the
// working tree. It is read only.
func Status(repo string) (map[string]string, bool) {
	found := map[string]string{}
	initialized := false
	for _, m := range markers {
		out, err := gitx.Run(repo, "config", "--local", "--get", m.Key)
		value := trimLine(out)
		if err != nil || value == "" {
			continue
		}
		found[m.Key] = value
		if m.Key == configPrefix+"initialized" && value == "true" {
			initialized = true
		}
	}
	return found, initialized
}

func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
