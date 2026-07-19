// Package todolist is a single-responsibility primitive: enumerate the pending
// implementation items recorded under docs/todolist/ and report their status.
// It reads only that one directory — never the repository tree — so its cost is
// bounded and independent of repo size. See docs/architecture/PRIMITIVE_CONSTITUTION.md
// and ADR 0004 (docs/decisions/0004-todolist-primitive.md).
package todolist

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dir is the single directory this primitive reads, relative to the repo root.
const Dir = "docs/todolist"

// Valid statuses. A todo starts Planned and turns Done ("green") once its Verify
// command passes.
const (
	StatusPlanned    = "Planned"
	StatusInProgress = "In-Progress"
	StatusDone       = "Done"
)

// Item is one tracked todo. Fields come only from the file header; the body is
// not parsed, keeping the primitive's output minimal and deterministic.
type Item struct {
	File   string `json:"file"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Verify string `json:"verify,omitempty"`
}

// Summary is the aggregate view returned alongside the items.
type Summary struct {
	Total      int `json:"total"`
	Planned    int `json:"planned"`
	InProgress int `json:"inProgress"`
	Done       int `json:"done"`
	Unknown    int `json:"unknown"`
}

// Report is the standardized, composable output of the primitive.
type Report struct {
	Dir     string  `json:"dir"`
	Items   []Item  `json:"items"`
	Summary Summary `json:"summary"`
}

// List reads docs/todolist/*.md (excluding README.md), parses each file's header,
// and returns the items sorted by filename with an aggregate summary. Missing
// directory is not an error — it yields an empty report. Deterministic: identical
// files produce an identical report.
func List(repo string) (Report, error) {
	dir := filepath.Join(repo, filepath.FromSlash(Dir))
	report := Report{Dir: Dir, Items: []Item{}}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || strings.EqualFold(name, "README.md") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		item, err := parseItem(filepath.Join(dir, name))
		if err != nil {
			return report, err
		}
		item.File = name
		report.Items = append(report.Items, item)
		countStatus(&report.Summary, item.Status)
	}
	report.Summary.Total = len(report.Items)
	return report, nil
}

// parseItem reads only the leading header lines: the first "# " title, and the
// "Status:" / "Verify:" fields. It stops at the first blank line after content so
// it never reads the whole body.
func parseItem(path string) (Item, error) {
	file, err := os.Open(path)
	if err != nil {
		return Item{}, err
	}
	defer file.Close()

	item := Item{Status: "Unknown"}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case item.Title == "" && strings.HasPrefix(line, "# "):
			item.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "Status:"):
			item.Status = normalizeStatus(strings.TrimSpace(strings.TrimPrefix(line, "Status:")))
		case strings.HasPrefix(line, "Verify:"):
			item.Verify = strings.TrimSpace(strings.TrimPrefix(line, "Verify:"))
		}
		if item.Title != "" && item.Status != "Unknown" && item.Verify != "" {
			break
		}
	}
	return item, scanner.Err()
}

func normalizeStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "planned":
		return StatusPlanned
	case "in-progress", "in progress", "wip":
		return StatusInProgress
	case "done", "green", "complete", "completed":
		return StatusDone
	default:
		return "Unknown"
	}
}

func countStatus(summary *Summary, status string) {
	switch status {
	case StatusPlanned:
		summary.Planned++
	case StatusInProgress:
		summary.InProgress++
	case StatusDone:
		summary.Done++
	default:
		summary.Unknown++
	}
}
