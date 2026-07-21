package kit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
)

type FreshCloneReport struct {
	SchemaVersion    int              `json:"schemaVersion"`
	Profile          string           `json:"profile"`
	OK               bool             `json:"ok"`
	SourceMode       string           `json:"sourceMode"`
	SourceRoot       string           `json:"sourceRoot"`
	SourceTreeOID    string           `json:"sourceTreeOID"`
	TempRoot         string           `json:"tempRoot"`
	CloneRoot        string           `json:"cloneRoot"`
	MaterializedRoot string           `json:"materializedRoot,omitempty"`
	ManifestPath     string           `json:"manifestPath,omitempty"`
	SourceManifest   *SourceManifest  `json:"sourceManifest,omitempty"`
	KeptTemp         bool             `json:"keptTemp"`
	Steps            []FreshCloneStep `json:"steps"`
	Errors           []string         `json:"errors,omitempty"`
}

type FreshCloneStep struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

func FreshClone(repo, profile string, keepTemp bool) (report FreshCloneReport) {
	profile = normalizeKitProfile(profile)
	report = FreshCloneReport{SchemaVersion: 1, Profile: profile, OK: true, SourceMode: "cloned", SourceRoot: repo}
	add := func(name string, started time.Time, ok bool, message string, output string) {
		report.Steps = append(report.Steps, FreshCloneStep{
			Name: name, OK: ok, Message: message, Output: trimOutput(output), ElapsedMS: time.Since(started).Milliseconds(),
		})
		if !ok {
			report.OK = false
			report.Errors = append(report.Errors, name+": "+message)
		}
	}
	stepStarted := time.Now()
	tempRoot, err := platform.CreateTempDir(repo, "fresh-clone")
	report.TempRoot = tempRoot
	report.CloneRoot = filepath.Join(tempRoot, "AiCoding")
	if err != nil {
		add("temp", stepStarted, false, err.Error(), "")
		return report
	}
	add("temp", stepStarted, true, "created and registered temporary directory", "")
	cloneRoot := report.CloneRoot
	defer func() {
		started := time.Now()
		switch {
		case report.OK && !keepTemp:
			if err := platform.ReleaseTempDir(repo, tempRoot, "fresh-clone"); err != nil {
				report.KeptTemp = true
				add("temp.release", started, false, err.Error(), "")
				return
			}
			add("temp.release", started, true, "released and recorded temporary directory", "")
		case keepTemp:
			report.KeptTemp = true
			if err := platform.RecordTempOutcome(repo, tempRoot, "fresh-clone", "investigating"); err != nil {
				add("temp.ledger", started, false, err.Error(), "")
				return
			}
			add("temp.ledger", started, true, "kept as investigating by explicit request", "")
		default:
			report.KeptTemp = true
			if err := platform.RecordTempOutcome(repo, tempRoot, "fresh-clone", "failed"); err != nil {
				add("temp.ledger", started, false, err.Error(), "")
				return
			}
			add("temp.ledger", started, true, "failed evidence retained and registered", "")
		}
	}()
	stepStarted = time.Now()
	report.SourceTreeOID, err = gitx.TreeOID(repo, "HEAD")
	if err != nil {
		add("git.source-tree", stepStarted, false, err.Error(), "")
		return report
	}
	add("git.source-tree", stepStarted, true, "captured source HEAD tree", "")
	stepStarted = time.Now()
	if out, err := runFresh("", "git", "clone", "--recurse-submodules", repo, cloneRoot); err != nil {
		add("git.clone", stepStarted, false, err.Error(), out)
		return report
	} else {
		add("git.clone", stepStarted, true, "cloned local repository", out)
	}
	stepStarted = time.Now()
	if out, err := verifyFreshCloneSubmodules(cloneRoot); err != nil {
		add("git.submodule", stepStarted, false, err.Error(), out)
		return report
	} else {
		add("git.submodule", stepStarted, true, "submodules verified", out)
	}
	stepStarted = time.Now()
	if out, err := overlayWorkingTree(repo, cloneRoot); err != nil {
		add("worktree.overlay", stepStarted, false, err.Error(), out)
		return report
	} else {
		add("worktree.overlay", stepStarted, true, "current worktree changes overlaid", out)
	}
	bin := filepath.Join(cloneRoot, "bin", "aicoding.exe")
	stepStarted = time.Now()
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		add("go.build.mkdir", stepStarted, false, err.Error(), "")
		return report
	}
	stepStarted = time.Now()
	if out, err := runFresh(cloneRoot, "go", "build", "-o", bin, "./cmd/aicoding"); err != nil {
		add("go.build", stepStarted, false, err.Error(), out)
		return report
	} else {
		add("go.build", stepStarted, true, "built Go CLI", out)
	}
	stepStarted = time.Now()
	checks, err := freshCloneChecks(bin, profile)
	if err != nil {
		add("profile", stepStarted, false, err.Error(), "")
		return report
	}
	for _, check := range checks {
		stepStarted = time.Now()
		out, err := runFresh(cloneRoot, check[0], check[1:]...)
		name := "check." + filepath.Base(check[0]) + " " + strings.Join(check[1:], " ")
		if err != nil {
			add(name, stepStarted, false, err.Error(), out)
			return report
		}
		add(name, stepStarted, true, "passed", out)
	}
	stepStarted = time.Now()
	if err := recordFreshCloneBaseline(repo, report.SourceTreeOID); err != nil {
		add("transport.baseline", stepStarted, false, err.Error(), "")
		return report
	}
	add("transport.baseline", stepStarted, true, "recorded successful true-clone tree", "")
	return report
}

func CheckFreshCloneContract(repo string) error {
	for _, path := range []string{".gitmodules", "CodingKit/agents/skills"} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("fresh-clone prerequisite %s: %w", path, err)
		}
		if path == "CodingKit/agents/skills" && !info.IsDir() {
			return fmt.Errorf("fresh-clone prerequisite is not a directory: %s", path)
		}
	}
	entries, err := os.ReadDir(filepath.Join(repo, "CodingKit", "agents", "skills"))
	if err != nil || len(entries) == 0 {
		return errorsf("fresh-clone skills submodule is empty")
	}
	for _, profile := range []string{"Smoke", "Full", "Release"} {
		checks, err := freshCloneChecks("aicoding", profile)
		if err != nil || len(checks) == 0 {
			return fmt.Errorf("fresh-clone %s checks are undefined", profile)
		}
	}
	return nil
}

func verifyFreshCloneSubmodules(cloneRoot string) (string, error) {
	args := freshCloneSubmoduleArgs()
	out, err := runFresh(cloneRoot, args[0], args[1:]...)
	if err != nil {
		return out, err
	}
	lines := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines++
		if strings.ContainsRune("-+U", rune(line[0])) {
			return out, fmt.Errorf("submodule is not at the recursively cloned commit: %s", strings.TrimSpace(line))
		}
	}
	if lines == 0 {
		return out, errorsf("recursive clone reported no submodules")
	}
	return out, nil
}

func freshCloneSubmoduleArgs() []string {
	return []string{"git", "submodule", "status", "--recursive"}
}

func runFresh(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func freshCloneChecks(bin, profile string) ([][]string, error) {
	switch profile {
	case "Smoke":
		return [][]string{{bin, "version"}}, nil
	case "Full":
		return [][]string{{"go", "test", "./..."}}, nil
	case "Release":
		return [][]string{{bin, "release", "verify", "--json"}}, nil
	default:
		return nil, fmt.Errorf("unsupported profile: %s", profile)
	}
}

func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 4000 {
		return s[:4000] + "\n...<truncated>"
	}
	return s
}
func overlayWorkingTree(repo, cloneRoot string) (string, error) {
	changed := []string{}
	for _, args := range [][]string{
		{"diff", "--name-only", "--diff-filter=ACMRT"},
		{"diff", "--cached", "--name-only", "--diff-filter=ACMRT"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		out, err := runFresh(repo, "git", args...)
		if err != nil {
			return out, err
		}
		for _, line := range strings.Split(out, "\n") {
			if rel := strings.TrimSpace(line); rel != "" {
				changed = append(changed, filepath.ToSlash(rel))
			}
		}
	}
	removed := []string{}
	for _, args := range [][]string{{"diff", "--name-only", "--diff-filter=D"}, {"diff", "--cached", "--name-only", "--diff-filter=D"}} {
		out, err := runFresh(repo, "git", args...)
		if err != nil {
			return out, err
		}
		for _, line := range strings.Split(out, "\n") {
			if rel := strings.TrimSpace(line); rel != "" {
				removed = append(removed, filepath.ToSlash(rel))
			}
		}
	}
	seen := map[string]bool{}
	copied := 0
	for _, rel := range changed {
		if seen[rel] || strings.HasPrefix(rel, ".git/") {
			continue
		}
		seen[rel] = true
		src := filepath.Join(repo, filepath.FromSlash(rel))
		info, err := os.Stat(src)
		if err != nil || info.IsDir() {
			continue
		}
		dst := filepath.Join(cloneRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		if err := copyFile(src, dst); err != nil {
			return "", err
		}
		copied++
	}
	removedCount := 0
	for _, rel := range removed {
		if rel == "" || strings.HasPrefix(rel, ".git/") {
			continue
		}
		if err := os.Remove(filepath.Join(cloneRoot, filepath.FromSlash(rel))); err == nil {
			removedCount++
		}
	}
	return fmt.Sprintf("copied=%d removed=%d", copied, removedCount), nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
