// Package adrreview is a single-responsibility primitive: enumerate the ADRs
// under docs/decisions/ and report which ones declare a primitive review
// (`PrimitiveReview: required`) but are missing the mandatory `## §12 Checklist
// 自评` section demanded by docs/architecture/PRIMITIVE_CONSTITUTION.md. It is an
// existence check only — judging the review's quality stays with humans. It reads
// exactly one directory (top-level *.md, no recursion, no repo scan), so its cost
// is bounded and independent of repository size. See ADR 0006.
package adrreview

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dir is the single directory this primitive reads, relative to the repo root.
const Dir = "docs/decisions"

// reviewField is the opt-in header that marks an ADR as introducing a new
// primitive/domain; historical ADRs without it are ignored.
const reviewField = "PrimitiveReview:"

// checklistAnchor marks the mandatory self-review section.
const checklistAnchor = "## §12 Checklist 自评"

// Item is one scanned ADR file's review state.
type Item struct {
	File         string `json:"file"`
	Review       string `json:"review"`       // required | n/a | "" (undeclared)
	HasChecklist bool   `json:"hasChecklist"` // §12 self-review section present
	OK           bool   `json:"ok"`           // false only for required-but-missing
}

// Report is the standardized, composable result.
type Report struct {
	Dir   string   `json:"dir"`
	Items []Item   `json:"items"`
	Gaps  []string `json:"gaps,omitempty"` // one message per violating ADR
}

// Check scans docs/decisions/*.md (top level only; plan-artifact subfolders are
// not ADRs) and reports every ADR that declares `PrimitiveReview: required` but
// lacks the §12 checklist section. Deterministic: files sorted by name, one read
// per file. A missing directory yields an empty report, not an error.
func Check(repo string) (Report, error) {
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		item, err := scanFile(filepath.Join(dir, name))
		if err != nil {
			return report, err
		}
		item.File = name
		item.OK = !(item.Review == "required" && !item.HasChecklist)
		report.Items = append(report.Items, item)
		if !item.OK {
			report.Gaps = append(report.Gaps,
				name+": declares PrimitiveReview: required but is missing the \"## §12 Checklist 自评\" section required by docs/architecture/PRIMITIVE_CONSTITUTION.md")
		}
	}
	return report, nil
}

// scanFile reads one ADR in a single pass, capturing the review header and
// whether the checklist anchor appears. It stops early once both are known.
func scanFile(path string) (Item, error) {
	file, err := os.Open(path)
	if err != nil {
		return Item{}, err
	}
	defer file.Close()

	item := Item{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if item.Review == "" && strings.HasPrefix(line, reviewField) {
			item.Review = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, reviewField)))
		}
		if !item.HasChecklist && strings.HasPrefix(line, checklistAnchor) {
			item.HasChecklist = true
		}
		if item.Review != "" && item.HasChecklist {
			break
		}
	}
	return item, scanner.Err()
}
