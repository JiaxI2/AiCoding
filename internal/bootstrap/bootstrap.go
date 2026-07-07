package bootstrap

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

type Options struct {
	Build bool `json:"build"`
}

type CheckItem struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Path   string `json:"path,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type Status struct {
	RepoRoot       string      `json:"repoRoot"`
	GoMod          bool        `json:"goMod"`
	GitDir         bool        `json:"gitDir"`
	GitFound       bool        `json:"gitFound"`
	GitVersion     string      `json:"gitVersion,omitempty"`
	GoFound        bool        `json:"goFound"`
	GoVersion      string      `json:"goVersion,omitempty"`
	BinDir         string      `json:"binDir"`
	BinDirExists   bool        `json:"binDirExists"`
	BinaryPath     string      `json:"binaryPath"`
	BinaryExists   bool        `json:"binaryExists"`
	BuildAttempted bool        `json:"buildAttempted"`
	BuildOK        bool        `json:"buildOk"`
	Checks         []CheckItem `json:"checks"`
}

func Check(repo string) (Status, []string) {
	status := baseStatus(repo)
	errs := []string{}
	add := func(name string, ok bool, path string, detail string) {
		item := CheckItem{Name: name, OK: ok, Path: path}
		if !ok {
			item.Detail = detail
			errs = append(errs, name+": "+detail)
		}
		status.Checks = append(status.Checks, item)
	}

	status.GoMod = platform.IsFile(platform.RepoPath(repo, "go.mod"))
	status.GitDir = platform.Exists(platform.RepoPath(repo, ".git"))
	status.BinDirExists = platform.IsDir(platform.RepoPath(repo, "bin"))
	status.BinaryExists = platform.IsFile(platform.RepoPath(repo, status.BinaryPath))

	status.GitVersion, status.GitFound = toolVersion("git", "--version")
	status.GoVersion, status.GoFound = toolVersion("go", "version")

	add("repo-root", platform.IsDir(repo), ".", "repo root not found")
	add("go.mod", status.GoMod, "go.mod", "go.mod is missing")
	add("git", status.GitFound && status.GitDir, ".git", "git is unavailable or .git is missing")
	add("go", status.GoFound, "", "go executable is unavailable")
	add("bin-dir", true, status.BinDir, "bin directory is created by bootstrap when missing")
	return status, errs
}

func Bootstrap(repo string, opts Options) (Status, []string) {
	status, errs := Check(repo)
	if err := os.MkdirAll(platform.RepoPath(repo, "bin"), 0o755); err != nil {
		errs = append(errs, "bin-dir: "+err.Error())
		status.BinDirExists = false
		return status, errs
	}
	status.BinDirExists = true
	if !opts.Build {
		return status, errs
	}
	status.BuildAttempted = true
	if len(errs) != 0 {
		return status, errs
	}
	cmd := exec.Command("go", "build", "-o", filepath.FromSlash(status.BinaryPath), "./cmd/aicoding")
	cmd.Dir = repo
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errs = append(errs, "build: "+strings.TrimSpace(stderr.String()))
		return status, errs
	}
	status.BuildOK = true
	status.BinaryExists = platform.IsFile(platform.RepoPath(repo, status.BinaryPath))
	return status, errs
}

func baseStatus(repo string) Status {
	return Status{
		RepoRoot:   repo,
		BinDir:     "bin",
		BinaryPath: "bin/aicoding.exe",
	}
}

func toolVersion(name string, args ...string) (string, bool) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
