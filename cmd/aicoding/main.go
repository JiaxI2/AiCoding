package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const version = "fast-path-v1"

type result struct {
	SchemaVersion int         `json:"schemaVersion"`
	Command       string      `json:"command"`
	OK            bool        `json:"ok"`
	Message       string      `json:"message,omitempty"`
	RepoRoot      string      `json:"repoRoot,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	Errors        []string    `json:"errors,omitempty"`
	ElapsedMS     int64       `json:"elapsedMs"`
}

type kitRegistry struct {
	SchemaVersion int           `json:"schemaVersion"`
	Name          string        `json:"name"`
	DefaultMode   string        `json:"defaultMode"`
	Kits          []registryKit `json:"kits"`
}

type registryKit struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	Order    int    `json:"order"`
	Manifest string `json:"manifest"`
}

type kitManifest struct {
	SchemaVersion int                               `json:"schemaVersion"`
	ID            string                            `json:"id"`
	Name          string                            `json:"name"`
	Version       string                            `json:"version"`
	Kind          []string                          `json:"kind"`
	Mode          string                            `json:"mode"`
	Description   string                            `json:"description"`
	Paths         map[string]string                 `json:"paths"`
	Commands      map[string]commandDef             `json:"commands"`
	Skills        map[string]json.RawMessage        `json:"skills"`
	Hooks         map[string]json.RawMessage        `json:"hooks"`
	State         map[string]string                 `json:"state"`
	Trust         map[string]interface{}            `json:"trust"`
	Profiles      map[string]map[string]interface{} `json:"profiles"`
}

type commandDef struct {
	Type           string   `json:"type"`
	Path           string   `json:"path"`
	Executable     string   `json:"executable"`
	Args           []string `json:"args"`
	Steps          []string `json:"steps"`
	RequiredPaths  []string `json:"requiredPaths"`
	SupportsJSON   *bool    `json:"supportsJson"`
	SupportsDryRun bool     `json:"supportsDryRun"`
	Reason         string   `json:"reason"`
}

type kitView struct {
	ID       string   `json:"id"`
	Enabled  bool     `json:"enabled"`
	Order    int      `json:"order"`
	Name     string   `json:"name,omitempty"`
	Version  string   `json:"version,omitempty"`
	Kind     []string `json:"kind,omitempty"`
	Mode     string   `json:"mode,omitempty"`
	Manifest string   `json:"manifest"`
}

type kitSmokeResult struct {
	ID       string   `json:"id"`
	OK       bool     `json:"ok"`
	Status   string   `json:"status"`
	Manifest string   `json:"manifest"`
	Errors   []string `json:"errors"`
}

func main() {
	start := time.Now()
	if len(os.Args) < 2 {
		printUsageAndExit(2)
	}
	cmd := os.Args[1]
	var res result
	var err error
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println(version)
		return
	case "hook":
		res, err = runHook(os.Args[2:], start)
	case "kit":
		res, err = runKit(os.Args[2:], start)
	case "doctor":
		res, err = runDoctor(os.Args[2:], start)
	case "governance":
		res, err = runGovernance(os.Args[2:], start)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsageAndExit(2)
	}
	if err != nil {
		if res.SchemaVersion == 0 {
			res = result{SchemaVersion: 1, Command: cmd, OK: false, Message: err.Error(), ElapsedMS: elapsed(start)}
		} else if res.Message == "" {
			res.Message = err.Error()
		}
	}
	if jsonRequested(os.Args[2:]) {
		writeJSON(res)
	} else {
		writeText(res)
	}
	if err != nil || !res.OK {
		os.Exit(1)
	}
}

func printUsageAndExit(code int) {
	fmt.Fprintf(os.Stderr, `AiCoding fast path CLI %s

Usage:
  aicoding hook pre-commit [--repo-root PATH] [--json]
  aicoding hook commit-msg --file COMMIT_MSG [--repo-root PATH] [--json]
  aicoding governance lint [--repo-root PATH] [--json]
  aicoding kit list [--repo-root PATH] [--json]
  aicoding kit verify --all --profile Smoke [--repo-root PATH] [--json]
  aicoding kit doctor [--repo-root PATH] [--json]
  aicoding doctor perf [--repo-root PATH] [--json]

This v1 CLI intentionally accelerates hot-path governance, staged DocSync and Smoke checks.
Full/Release gates remain in PowerShell/Python and CI.
`, version)
	os.Exit(code)
}

func jsonRequested(args []string) bool {
	for _, a := range args {
		if a == "--json" || a == "-json" || a == "-Json" {
			return true
		}
	}
	return false
}

func elapsed(start time.Time) int64 { return time.Since(start).Milliseconds() }

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeText(res result) {
	status := "OK"
	if !res.OK {
		status = "FAIL"
	}
	if res.Message != "" {
		fmt.Printf("[%s] %s (%d ms)\n", status, res.Message, res.ElapsedMS)
	} else {
		fmt.Printf("[%s] %s (%d ms)\n", status, res.Command, res.ElapsedMS)
	}
	for _, e := range res.Errors {
		fmt.Printf("  - %s\n", e)
	}
	switch data := res.Data.(type) {
	case []kitView:
		for _, k := range data {
			fmt.Printf("  %02d %-38s %-8t %-10s %s\n", k.Order, k.ID, k.Enabled, k.Version, k.Manifest)
		}
	case []kitSmokeResult:
		for _, k := range data {
			label := "OK"
			if !k.OK {
				label = "FAIL"
			}
			fmt.Printf("  [%s] %-38s %s\n", label, k.ID, k.Status)
			for _, e := range k.Errors {
				fmt.Printf("      - %s\n", e)
			}
		}
	}
}

func runHook(args []string, start time.Time) (result, error) {
	if len(args) < 1 {
		return result{}, errors.New("hook requires subcommand: pre-commit or commit-msg")
	}
	sub := args[0]
	switch sub {
	case "pre-commit":
		fs := flag.NewFlagSet("hook pre-commit", flag.ContinueOnError)
		repoArg := fs.String("repo-root", "", "repository root")
		_ = fs.Bool("json", false, "json output")
		_ = fs.Parse(args[1:])
		repo, err := resolveRepoRoot(*repoArg)
		if err != nil {
			return failResult("hook pre-commit", start, "cannot resolve repo root", nil, err.Error()), err
		}
		errs := append(lintGovernance(repo, "pre-commit", ""), lintDocSyncStaged(repo)...)
		return result{SchemaVersion: 1, Command: "hook pre-commit", OK: len(errs) == 0, Message: "pre-commit fast gate", RepoRoot: repo, Errors: errs, ElapsedMS: elapsed(start)}, boolErr(errs)
	case "commit-msg":
		fs := flag.NewFlagSet("hook commit-msg", flag.ContinueOnError)
		repoArg := fs.String("repo-root", "", "repository root")
		fileArg := fs.String("file", "", "commit message file")
		_ = fs.Bool("json", false, "json output")
		_ = fs.Parse(args[1:])
		if *fileArg == "" && fs.NArg() > 0 {
			*fileArg = fs.Arg(0)
		}
		repo, err := resolveRepoRoot(*repoArg)
		if err != nil {
			return failResult("hook commit-msg", start, "cannot resolve repo root", nil, err.Error()), err
		}
		errs := lintGovernance(repo, "commit-msg", *fileArg)
		return result{SchemaVersion: 1, Command: "hook commit-msg", OK: len(errs) == 0, Message: "commit-msg fast gate", RepoRoot: repo, Errors: errs, ElapsedMS: elapsed(start)}, boolErr(errs)
	default:
		return result{}, fmt.Errorf("unsupported hook: %s", sub)
	}
}

func runGovernance(args []string, start time.Time) (result, error) {
	if len(args) < 1 || args[0] != "lint" {
		return result{}, errors.New("governance requires subcommand: lint")
	}
	fs := flag.NewFlagSet("governance lint", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	mode := fs.String("mode", "all", "all or pre-commit")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := resolveRepoRoot(*repoArg)
	if err != nil {
		return failResult("governance lint", start, "cannot resolve repo root", nil, err.Error()), err
	}
	errs := lintGovernance(repo, *mode, "")
	return result{SchemaVersion: 1, Command: "governance lint", OK: len(errs) == 0, Message: "governance fast lint", RepoRoot: repo, Errors: errs, ElapsedMS: elapsed(start)}, boolErr(errs)
}

func runKit(args []string, start time.Time) (result, error) {
	if len(args) < 1 {
		return result{}, errors.New("kit requires subcommand")
	}
	sub := args[0]
	fs := flag.NewFlagSet("kit "+sub, flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	kitArg := fs.String("kit", "", "kit id")
	allArg := fs.Bool("all", false, "all enabled kits")
	profile := fs.String("profile", "Smoke", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := resolveRepoRoot(*repoArg)
	if err != nil {
		return failResult("kit "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	entries, err := loadRegistry(repo)
	if err != nil {
		return failResult("kit "+sub, start, "cannot load registry", nil, err.Error()), err
	}
	withManifests := loadKitViews(repo, entries)
	switch sub {
	case "list":
		return result{SchemaVersion: 1, Command: "kit list", OK: true, Message: "kit registry", RepoRoot: repo, Data: withManifests, ElapsedMS: elapsed(start)}, nil
	case "doctor":
		errs := doctorKits(repo, entries)
		return result{SchemaVersion: 1, Command: "kit doctor", OK: len(errs) == 0, Message: "kit registry doctor", RepoRoot: repo, Data: withManifests, Errors: errs, ElapsedMS: elapsed(start)}, boolErr(errs)
	case "verify", "test":
		if !strings.EqualFold(*profile, "Smoke") {
			return failResult("kit "+sub, start, "fast CLI only handles Smoke profile", nil, "use scripts/aicoding-kit.ps1 for Full/Release"), errors.New("non-Smoke profile is not handled by fast CLI")
		}
		selected, err := selectKits(entries, *kitArg, *allArg)
		if err != nil {
			return failResult("kit "+sub, start, "kit selection failed", nil, err.Error()), err
		}
		results := smokeKits(repo, selected)
		errs := []string{}
		for _, r := range results {
			if !r.OK {
				for _, e := range r.Errors {
					errs = append(errs, r.ID+": "+e)
				}
			}
		}
		return result{SchemaVersion: 1, Command: "kit " + sub, OK: len(errs) == 0, Message: "kit smoke " + sub, RepoRoot: repo, Data: results, Errors: errs, ElapsedMS: elapsed(start)}, boolErr(errs)
	default:
		return result{}, fmt.Errorf("unsupported kit subcommand: %s", sub)
	}
}

func runDoctor(args []string, start time.Time) (result, error) {
	if len(args) < 1 || args[0] != "perf" {
		return result{}, errors.New("doctor requires subcommand: perf")
	}
	fs := flag.NewFlagSet("doctor perf", flag.ContinueOnError)
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	_ = fs.Parse(args[1:])
	repo, err := resolveRepoRoot(*repoArg)
	if err != nil {
		return failResult("doctor perf", start, "cannot resolve repo root", nil, err.Error()), err
	}
	checks := []map[string]interface{}{}
	measure := func(name string, fn func() error) {
		t0 := time.Now()
		err := fn()
		item := map[string]interface{}{"name": name, "elapsedMs": time.Since(t0).Milliseconds(), "ok": err == nil}
		if err != nil {
			item["error"] = err.Error()
		}
		checks = append(checks, item)
	}
	measure("git rev-parse", func() error { _, e := runGit(repo, "rev-parse", "--show-toplevel"); return e })
	measure("git diff cached names", func() error { _, e := stagedFiles(repo); return e })
	measure("load kit registry", func() error { _, e := loadRegistry(repo); return e })
	measure("governance lint", func() error { return boolErr(lintGovernance(repo, "pre-commit", "")) })
	measure("staged docsync lint", func() error { return boolErr(lintDocSyncStaged(repo)) })
	return result{SchemaVersion: 1, Command: "doctor perf", OK: true, Message: "performance probes", RepoRoot: repo, Data: checks, ElapsedMS: elapsed(start)}, nil
}

func failResult(cmd string, start time.Time, msg string, data interface{}, errs ...string) result {
	return result{SchemaVersion: 1, Command: cmd, OK: false, Message: msg, Data: data, Errors: errs, ElapsedMS: elapsed(start)}
}

func boolErr(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func resolveRepoRoot(repoArg string) (string, error) {
	if repoArg != "" {
		p, err := filepath.Abs(repoArg)
		if err != nil {
			return "", err
		}
		return p, nil
	}
	out, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func runGit(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if repo != "" {
		cmd.Dir = repo
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func readText(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }
func isFile(path string) bool { st, err := os.Stat(path); return err == nil && !st.IsDir() }
func isDir(path string) bool  { st, err := os.Stat(path); return err == nil && st.IsDir() }

func repoPath(repo, rel string) string { return filepath.Join(repo, filepath.FromSlash(rel)) }

func lintGovernance(repo, mode, commitMsgPath string) []string {
	errs := []string{}
	fail := func(msg string) { errs = append(errs, msg) }
	requiredFiles := []string{"README.md", "README_EN.md", "CHANGELOG.md", ".github/RELEASE_TEMPLATE.md", ".github/repository-governance.toml", ".githooks/pre-commit", ".githooks/commit-msg", "scripts/verify-release-notes.ps1"}
	for _, f := range requiredFiles {
		if !isFile(repoPath(repo, f)) {
			fail("required governance file missing: " + f)
		}
	}
	scanFiles := []string{"README.md", "README_CN.md", "README_EN.md", "CHANGELOG.md", ".github/repository-governance.toml"}
	placeholder := regexp.MustCompile(`\{\{[^}]+\}\}|UNRESOLVED_PLACEHOLDER|TODO_PLACEHOLDER`)
	for _, f := range scanFiles {
		p := repoPath(repo, f)
		if !isFile(p) {
			continue
		}
		content, err := readText(p)
		if err != nil {
			fail("cannot read " + f + ": " + err.Error())
			continue
		}
		if placeholder.MatchString(content) {
			fail("unresolved placeholder found in " + f)
		}
	}
	readme, _ := readText(repoPath(repo, "README.md"))
	readmeEN, _ := readText(repoPath(repo, "README_EN.md"))
	gov, _ := readText(repoPath(repo, ".github/repository-governance.toml"))
	changelog, _ := readText(repoPath(repo, "CHANGELOG.md"))
	readmeHead := strings.Join(firstLines(readme, 16), "\n")
	if isFile(repoPath(repo, "README_CN.md")) {
		if !strings.Contains(readmeHead, "README_CN.md") {
			fail("README.md must include top-of-file README_CN.md link")
		}
		if !strings.Contains(readmeHead, "README_EN.md") {
			fail("README.md must include top-of-file README_EN.md link")
		}
		if strings.Contains(readmeHead, "README.md#english") {
			fail("README.md must not use in-page English anchor")
		}
		mustContain(gov, `primary_language = "zh-CN"`, ".github/repository-governance.toml must set README primary_language to zh-CN", fail)
		mustContain(gov, `secondary_language_surface = "top-file-language-switch-and-github-about"`, ".github/repository-governance.toml must define secondary language surface", fail)
		mustContain(gov, `english_language_file = "README_EN.md"`, ".github/repository-governance.toml must define README_EN.md", fail)
		mustContain(gov, `quick_environment_preview = true`, ".github/repository-governance.toml must require quick environment preview", fail)
		if !strings.Contains(gov, "[github_about]") || !strings.Contains(gov, "require_bilingual = true") {
			fail(".github/repository-governance.toml must require bilingual GitHub About metadata")
		}
	}
	reGitGov := regexp.MustCompile(`Git Governance Standard|Git 治理标准`)
	reCommit := regexp.MustCompile(`feat.+fix.+docs.+style.+refactor.+perf.+test.+chore|feat.+fix.+docs.+build.+ci.+chore`)
	reBranch := regexp.MustCompile(`main.+master.+develop.+feature.+test.+release.+hotfix|main.+develop.+feature.+test.+release.+hotfix`)
	reRelease := regexp.MustCompile(`Release.+type|Release.+typed|按类型汇总|主类型`)
	if !reGitGov.MatchString(readme) {
		fail("README.md must document Git governance standard")
	}
	if !reCommit.MatchString(readme) {
		fail("README.md must document commit type taxonomy")
	}
	if !reBranch.MatchString(readme) {
		fail("README.md must document branch naming and environment mapping")
	}
	if !reRelease.MatchString(readme) {
		fail("README.md must document typed release notes")
	}
	if !reGitGov.MatchString(readmeEN) {
		fail("README_EN.md must document Git governance standard")
	}
	if !reCommit.MatchString(readmeEN) {
		fail("README_EN.md must document commit type taxonomy")
	}
	if !regexp.MustCompile(`notes_template\s*=\s*"\.github/RELEASE_TEMPLATE\.md"`).MatchString(gov) {
		fail("repository-governance.toml must declare release notes template")
	}
	if !regexp.MustCompile(`notes_validator\s*=\s*"scripts/verify-release-notes\.ps1"`).MatchString(gov) {
		fail("repository-governance.toml must declare release notes validator")
	}
	if !strings.Contains(gov, "required_bilingual_sections") {
		fail("repository-governance.toml must require bilingual release notes sections")
	}
	if strings.Contains(gov, `mode = "unreleased"`) && !strings.Contains(changelog, "[Unreleased]") {
		fail("CHANGELOG.md must contain [Unreleased] when changelog.mode is unreleased")
	}
	if !regexp.MustCompile(`\*\*(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]*\))?\*\*`).MatchString(changelog) {
		fail("CHANGELOG.md must include typed entries such as **docs** or **chore**")
	}
	if mode == "all" || mode == "pre-commit" {
		staged, err := stagedFiles(repo)
		if err != nil {
			fail(err.Error())
		} else if len(staged) > 0 && !contains(staged, "CHANGELOG.md") && os.Getenv("AICODING_SKIP_CHANGELOG") != "1" {
			fail("CHANGELOG.md must be staged for normal commits; set AICODING_SKIP_CHANGELOG=1 only for approved exclusion")
		}
	}
	if mode == "commit-msg" {
		if commitMsgPath == "" {
			fail("commit message path is required")
		} else {
			p := commitMsgPath
			if !filepath.IsAbs(p) {
				p = filepath.Join(repo, p)
			}
			content, err := readText(p)
			if err != nil {
				fail("cannot read commit message: " + err.Error())
			} else {
				subject := firstCommitSubject(content)
				if subject == "" {
					fail("commit message subject is empty")
				}
				if !regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]+\))?: `).MatchString(subject) {
					fail("commit subject must start with allowed type and optional scope. Got: " + subject)
				}
				if !regexp.MustCompile(`: .{8,}$`).MatchString(subject) {
					fail("commit subject summary must be at least 8 characters. Got: " + subject)
				}
			}
		}
	}
	return errs
}

func lintDocSyncStaged(repo string) []string {
	staged, err := stagedFiles(repo)
	if err != nil {
		return []string{err.Error()}
	}
	if len(staged) == 0 {
		return nil
	}
	docChanged := false
	riskChanged := false
	for _, f := range staged {
		if isDocPath(f) {
			docChanged = true
		}
		if isDocSyncRiskPath(f) {
			riskChanged = true
		}
	}
	if riskChanged && !docChanged && os.Getenv("AICODING_SKIP_DOCSYNC") != "1" {
		return []string{"documentation sync fast gate: source/script/config/hook/skill changes require staged docs or AICODING_SKIP_DOCSYNC=1; CI still runs full DocSync Plus"}
	}
	return nil
}

func stagedFiles(repo string) ([]string, error) {
	out, err := runGit(repo, "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\\", "/"))
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func isDocPath(f string) bool {
	f = strings.ReplaceAll(f, "\\", "/")
	if strings.HasSuffix(f, ".md") {
		return true
	}
	return f == "README.md" || f == "README_CN.md" || f == "README_EN.md" || f == "CHANGELOG.md" || strings.HasPrefix(f, "docs/") || strings.HasPrefix(f, "config/") && strings.HasSuffix(f, ".md")
}

func isDocSyncRiskPath(f string) bool {
	f = strings.ReplaceAll(f, "\\", "/")
	if strings.HasPrefix(f, ".git/") || strings.Contains(f, "/__pycache__/") || strings.Contains(f, "/.pytest_cache/") {
		return false
	}
	if strings.HasPrefix(f, "scripts/") || strings.HasPrefix(f, "src/") || strings.HasPrefix(f, "config/") || strings.HasPrefix(f, ".githooks/") || strings.HasPrefix(f, ".github/workflows/") || strings.HasPrefix(f, ".agents/") || strings.HasPrefix(f, "CodingKit/") || strings.HasPrefix(f, "skills/") || strings.HasPrefix(f, "codex-skills/") {
		return true
	}
	for _, ext := range []string{".c", ".h", ".cpp", ".hpp", ".py", ".ps1", ".sh"} {
		if strings.HasSuffix(f, ext) {
			return true
		}
	}
	return false
}

func mustContain(content, needle, msg string, fail func(string)) {
	if !strings.Contains(content, needle) {
		fail(msg)
	}
}

func firstLines(s string, n int) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) > n {
		return lines[:n]
	}
	return lines
}

func firstCommitSubject(s string) string {
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

func contains(list []string, target string) bool {
	for _, x := range list {
		if x == target {
			return true
		}
	}
	return false
}

func loadRegistry(repo string) ([]registryKit, error) {
	p := repoPath(repo, "config/kit-registry.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var reg kitRegistry
	if err := json.Unmarshal(b, &reg); err != nil {
		return nil, err
	}
	sort.SliceStable(reg.Kits, func(i, j int) bool { return reg.Kits[i].Order < reg.Kits[j].Order })
	return reg.Kits, nil
}

func loadManifest(repo, rel string) (kitManifest, error) {
	b, err := os.ReadFile(repoPath(repo, rel))
	if err != nil {
		return kitManifest{}, err
	}
	var m kitManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return kitManifest{}, err
	}
	return m, nil
}

func loadKitViews(repo string, entries []registryKit) []kitView {
	views := []kitView{}
	for _, e := range entries {
		v := kitView{ID: e.ID, Enabled: e.Enabled, Order: e.Order, Manifest: e.Manifest}
		if m, err := loadManifest(repo, e.Manifest); err == nil {
			v.Name = m.Name
			v.Version = m.Version
			v.Kind = m.Kind
			v.Mode = m.Mode
		}
		views = append(views, v)
	}
	return views
}

func doctorKits(repo string, entries []registryKit) []string {
	errs := []string{}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.ID == "" {
			errs = append(errs, "registry kit id is empty")
		}
		if seen[e.ID] {
			errs = append(errs, "duplicate kit id: "+e.ID)
		}
		seen[e.ID] = true
		if e.Manifest == "" {
			errs = append(errs, e.ID+": manifest is empty")
			continue
		}
		m, err := loadManifest(repo, e.Manifest)
		if err != nil {
			errs = append(errs, e.ID+": cannot load manifest: "+err.Error())
			continue
		}
		if m.ID != e.ID {
			errs = append(errs, e.ID+": manifest id mismatch: "+m.ID)
		}
		if m.Mode != "script-adapter" && m.Mode != "declarative" {
			errs = append(errs, e.ID+": invalid mode: "+m.Mode)
		}
		if len(m.Kind) == 0 {
			errs = append(errs, e.ID+": empty kind")
		}
		if len(m.Commands) == 0 {
			errs = append(errs, e.ID+": empty commands")
		}
	}
	return errs
}

func selectKits(entries []registryKit, kit string, all bool) ([]registryKit, error) {
	if all && kit != "" {
		return nil, errors.New("use either --all or --kit, not both")
	}
	if !all && kit == "" {
		return nil, errors.New("kit verify/test requires --all or --kit")
	}
	selected := []registryKit{}
	for _, e := range entries {
		if all && e.Enabled {
			selected = append(selected, e)
		}
		if kit != "" && e.ID == kit {
			selected = append(selected, e)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no kit matched")
	}
	return selected, nil
}

func smokeKits(repo string, entries []registryKit) []kitSmokeResult {
	results := []kitSmokeResult{}
	for _, e := range entries {
		errs := []string{}
		if !isFile(repoPath(repo, e.Manifest)) {
			errs = append(errs, "manifest missing")
		}
		m, err := loadManifest(repo, e.Manifest)
		if err != nil {
			errs = append(errs, "manifest parse failed: "+err.Error())
		} else {
			if m.ID != e.ID {
				errs = append(errs, "manifest id mismatch: "+m.ID)
			}
			if m.Mode != "script-adapter" && m.Mode != "declarative" {
				errs = append(errs, "invalid mode: "+m.Mode)
			}
			if len(m.Kind) == 0 {
				errs = append(errs, "empty kind")
			}
			for action, c := range m.Commands {
				switch c.Type {
				case "powershell-script":
					if c.Path == "" {
						errs = append(errs, action+": powershell-script path is empty")
					} else if !isFile(repoPath(repo, c.Path)) {
						errs = append(errs, action+": missing command script: "+c.Path)
					}
				case "builtin-check":
					for _, rel := range c.RequiredPaths {
						if !exists(repoPath(repo, rel)) {
							errs = append(errs, action+": missing required path: "+rel)
						}
					}
				case "composed", "external-command", "builtin-package", "unsupported":
					// Smoke mode validates manifest shape only; external tools stay out of the hot path.
				default:
					errs = append(errs, action+": unsupported command type in manifest: "+c.Type)
				}
			}
		}
		status := "smoke"
		if len(errs) > 0 {
			status = "failed"
		}
		results = append(results, kitSmokeResult{ID: e.ID, OK: len(errs) == 0, Status: status, Manifest: e.Manifest, Errors: errs})
	}
	return results
}
