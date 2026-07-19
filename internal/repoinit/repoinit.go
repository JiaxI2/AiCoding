// Package repoinit sets up the local AI-coding environment in a repository,
// parallel to how `git init` prepares .git/. It composes existing primitives
// (git via internal/gitx, paths via internal/platform) — it holds no business
// logic of its own beyond orchestrating an idempotent setup. See ADR 0005
// (docs/decisions/0005-repo-init.md) and docs/architecture/GRAPH_FIRST.md.
package repoinit

import (
	"os"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

// configPrefix is the git-config namespace that holds AI-coding local markers.
// These live in .git/config: local, per-clone, never committed — the same place
// git keeps its own settings — so later commands read setup state instantly via
// `git config --get aicoding.*` without scanning the working tree.
const configPrefix = "aicoding."

// SchemaVersion of the local marker set; bumped only on a marker-shape change.
const SchemaVersion = "1"

// Marker is one aicoding.* git-config key/value that init ensures.
var markers = []struct{ Key, Value string }{
	{configPrefix + "initialized", "true"},
	{configPrefix + "home", ".aicoding"},
	{configPrefix + "schemaVersion", SchemaVersion},
}

// Report is the standardized, composable result of an init run.
type Report struct {
	Repo             string            `json:"repo"`
	GitInitialized   bool              `json:"gitInitialized"`   // true when init created .git this run
	GitAlreadyRepo   bool              `json:"gitAlreadyRepo"`   // true when .git already existed
	HooksPath        string            `json:"hooksPath"`        // core.hooksPath after wiring
	ConfigMarkers    map[string]string `json:"configMarkers"`    // aicoding.* keys written to .git/config
	AicodingHomePath string            `json:"aicodingHomePath"` // .aicoding local state root
	Actions          []string          `json:"actions"`          // human-readable, ordered
	OK               bool              `json:"ok"`
	Errors           []string          `json:"errors,omitempty"`
}

// Init is idempotent: it ensures a git repo, wires the repo hooks, writes the
// AI-coding markers into git's own config, and ensures the .aicoding home exists.
// Re-running changes nothing. It never writes loose files under .git/ (fragile);
// the local key/value store is git config, which git owns.
func Init(repo string) Report {
	report := Report{Repo: repo, ConfigMarkers: map[string]string{}, AicodingHomePath: ".aicoding"}

	// 1) Ensure a git repository. `git init` is itself idempotent, but report which.
	if platform.Exists(platform.RepoPath(repo, ".git")) {
		report.GitAlreadyRepo = true
		report.Actions = append(report.Actions, "git repository already present")
	} else {
		if _, err := gitx.Run(repo, "init"); err != nil {
			report.Errors = append(report.Errors, "git init: "+err.Error())
			return report
		}
		report.GitInitialized = true
		report.Actions = append(report.Actions, "ran git init")
	}

	// 2) Activate the repository hooks via git's own mechanism.
	if _, err := gitx.Run(repo, "config", "core.hooksPath", ".githooks"); err != nil {
		report.Errors = append(report.Errors, "wire core.hooksPath: "+err.Error())
		return report
	}
	report.HooksPath = ".githooks"
	report.Actions = append(report.Actions, "wired core.hooksPath = .githooks")

	// 3) Write AI-coding markers into .git/config (local, per-clone, uncommitted).
	for _, m := range markers {
		if _, err := gitx.Run(repo, "config", m.Key, m.Value); err != nil {
			report.Errors = append(report.Errors, "set "+m.Key+": "+err.Error())
			return report
		}
		report.ConfigMarkers[m.Key] = m.Value
	}
	report.Actions = append(report.Actions, "wrote aicoding.* markers to .git/config")

	// 4) Ensure the .aicoding local state home exists (its versioned subtrees are
	//    created on demand and gitignored; init only guarantees the root).
	home := platform.RepoPath(repo, ".aicoding")
	if err := os.MkdirAll(home, 0o755); err != nil {
		report.Errors = append(report.Errors, "ensure .aicoding home: "+err.Error())
		return report
	}
	report.Actions = append(report.Actions, "ensured .aicoding home")

	report.OK = true
	return report
}

// Status reads the AI-coding markers from git config so later commands can judge
// setup state quickly (e.g. gate on aicoding.initialized) without scanning the
// working tree. It is read only.
func Status(repo string) (map[string]string, bool) {
	found := map[string]string{}
	initialized := false
	for _, m := range markers {
		out, err := gitx.Run(repo, "config", "--get", m.Key)
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
