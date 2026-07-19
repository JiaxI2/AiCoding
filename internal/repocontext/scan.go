package repocontext

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

// languageByExt maps a source file extension to a human language label. Only
// extensions listed here are counted, keeping the scan deterministic and focused
// on source rather than generated or binary noise.
var languageByExt = map[string]string{
	".go":   "Go",
	".c":    "C",
	".h":    "C Header",
	".cpp":  "C++",
	".hpp":  "C++ Header",
	".py":   "Python",
	".ps1":  "PowerShell",
	".psm1": "PowerShell Module",
	".sh":   "Shell",
	".js":   "JavaScript",
	".ts":   "TypeScript",
	".md":   "Markdown",
	".json": "JSON",
	".yaml": "YAML",
	".yml":  "YAML",
	".toml": "TOML",
}

// toolchainMarkers maps a repo-root file to the toolchain it signals.
var toolchainMarkers = map[string]string{
	"go.mod":           "Go modules",
	"Taskfile.yml":     "Taskfile",
	".clang-format":    "clang-format",
	"package.json":     "Node",
	"pyproject.toml":   "Python (pyproject)",
	"requirements.txt": "Python (requirements)",
	".gitmodules":      "Git submodules",
}

// skipDirs are directory names never descended into during a scan: version
// control, generated output, dependency caches, and the domain's own owned root.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".venv":        true,
	"venv":         true,
	"bin":          true,
	"dist":         true,
	"test-results": true,
	".aicoding":    true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
}

// Scan walks the repository and produces normalized Facts plus a stable snapshot.
// The walk is deterministic: entries are sorted, skip directories are pruned, and
// no absolute path or timestamp enters the returned value.
func Scan(repo string) (Facts, registryobject.Snapshot, error) {
	langFiles := map[string]int{}
	domainFiles := map[string]int{}
	domainLangFiles := map[string]map[string]int{}

	err := filepath.WalkDir(repo, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == repo {
			return nil
		}
		rel, relErr := filepath.Rel(repo, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		top := topLevel(rel)
		if entry.IsDir() {
			if skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		lang, ok := languageByExt[ext]
		if !ok {
			return nil
		}
		langFiles[ext]++
		if top != "" {
			domainFiles[top]++
			if domainLangFiles[top] == nil {
				domainLangFiles[top] = map[string]int{}
			}
			domainLangFiles[top][lang]++
		}
		return nil
	})
	if err != nil {
		return Facts{}, registryobject.Snapshot{}, err
	}

	facts := Facts{
		Repo:       repoName(repo),
		Languages:  sortedLanguages(langFiles),
		Toolchains: detectToolchains(repo),
		Domains:    sortedDomains(domainFiles, domainLangFiles),
	}
	snapshot, err := registryobject.NewSnapshot("repo-context-facts", facts)
	if err != nil {
		return Facts{}, registryobject.Snapshot{}, err
	}
	return facts, snapshot, nil
}

func topLevel(rel string) string {
	if idx := strings.IndexByte(rel, '/'); idx >= 0 {
		return rel[:idx]
	}
	return ""
}

func repoName(repo string) string {
	if module, ok := goModuleName(repo); ok {
		return module
	}
	return filepath.Base(repo)
}

func goModuleName(repo string) (string, bool) {
	content, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if module != "" {
				return path.Base(module), true
			}
		}
	}
	return "", false
}

func detectToolchains(repo string) []string {
	found := []string{}
	for marker, label := range toolchainMarkers {
		if _, err := os.Stat(filepath.Join(repo, marker)); err == nil {
			found = append(found, label)
		}
	}
	sort.Strings(found)
	return found
}

func sortedLanguages(langFiles map[string]int) []LanguageStat {
	stats := make([]LanguageStat, 0, len(langFiles))
	for ext, count := range langFiles {
		stats = append(stats, LanguageStat{
			Language:  languageByExt[ext],
			Extension: ext,
			Files:     count,
		})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Files == stats[j].Files {
			return stats[i].Extension < stats[j].Extension
		}
		return stats[i].Files > stats[j].Files
	})
	return stats
}

func sortedDomains(domainFiles map[string]int, domainLangFiles map[string]map[string]int) []Domain {
	domains := make([]Domain, 0, len(domainFiles))
	for dir, count := range domainFiles {
		domains = append(domains, Domain{
			Path:            dir,
			Files:           count,
			PrimaryLanguage: primaryLanguage(domainLangFiles[dir]),
		})
	}
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Path < domains[j].Path
	})
	return domains
}

func primaryLanguage(langFiles map[string]int) string {
	primary := ""
	best := 0
	for lang, count := range langFiles {
		if count > best || (count == best && lang < primary) {
			best = count
			primary = lang
		}
	}
	return primary
}
