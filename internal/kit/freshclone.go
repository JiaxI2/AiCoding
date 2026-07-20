package kit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type FreshCloneReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	Profile       string           `json:"profile"`
	OK            bool             `json:"ok"`
	SourceRoot    string           `json:"sourceRoot"`
	TempRoot      string           `json:"tempRoot"`
	CloneRoot     string           `json:"cloneRoot"`
	KeptTemp      bool             `json:"keptTemp"`
	Steps         []FreshCloneStep `json:"steps"`
	Errors        []string         `json:"errors,omitempty"`
}

type FreshCloneStep struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

func FreshClone(repo, profile string, keepTemp bool) FreshCloneReport {
	profile = strings.Title(strings.ToLower(strings.TrimSpace(profile)))
	if profile == "" {
		profile = "Smoke"
	}
	tempRoot := filepath.Join(os.TempDir(), "aicoding-fresh-clone-"+time.Now().UTC().Format("20060102-150405")+"-"+randomSuffix())
	cloneRoot := filepath.Join(tempRoot, "AiCoding")
	report := FreshCloneReport{SchemaVersion: 1, Profile: profile, OK: true, SourceRoot: repo, TempRoot: tempRoot, CloneRoot: cloneRoot}
	add := func(name string, ok bool, message string, output string) {
		report.Steps = append(report.Steps, FreshCloneStep{Name: name, OK: ok, Message: message, Output: trimOutput(output)})
		if !ok {
			report.OK = false
			report.Errors = append(report.Errors, name+": "+message)
		}
	}
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		add("temp", false, err.Error(), "")
		return report
	}
	defer func() {
		if report.OK && !keepTemp {
			_ = os.RemoveAll(tempRoot)
		} else {
			report.KeptTemp = true
		}
	}()
	if out, err := runFresh("", "git", "clone", "--recurse-submodules", repo, cloneRoot); err != nil {
		add("git.clone", false, err.Error(), out)
		return report
	} else {
		add("git.clone", true, "cloned local repository", out)
	}
	if out, err := verifyFreshCloneSubmodules(cloneRoot); err != nil {
		add("git.submodule", false, err.Error(), out)
		return report
	} else {
		add("git.submodule", true, "submodules verified", out)
	}
	if out, err := overlayWorkingTree(repo, cloneRoot); err != nil {
		add("worktree.overlay", false, err.Error(), out)
		return report
	} else {
		add("worktree.overlay", true, "current worktree changes overlaid", out)
	}
	bin := filepath.Join(cloneRoot, "bin", "aicoding.exe")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		add("go.build.mkdir", false, err.Error(), "")
		return report
	}
	if out, err := runFresh(cloneRoot, "go", "build", "-o", bin, "./cmd/aicoding"); err != nil {
		add("go.build", false, err.Error(), out)
		return report
	} else {
		add("go.build", true, "built Go CLI", out)
	}
	checks, err := freshCloneChecks(bin, profile)
	if err != nil {
		add("profile", false, err.Error(), "")
		return report
	}
	for _, check := range checks {
		out, err := runFresh(cloneRoot, check[0], check[1:]...)
		name := "check." + filepath.Base(check[0]) + " " + strings.Join(check[1:], " ")
		if err != nil {
			add(name, false, err.Error(), out)
			return report
		}
		add(name, true, "passed", out)
	}
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
func randomSuffix() string { return fmt.Sprintf("%d", time.Now().UnixNano()%1000000) }

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
