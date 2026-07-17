package testengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Load(outDir string) (Report, error) {
	var testReport Report
	raw, err := os.ReadFile(filepath.Join(outDir, "results.json"))
	if err != nil {
		return testReport, fmt.Errorf("read results.json: %w", err)
	}
	if err := json.Unmarshal(raw, &testReport); err != nil {
		return testReport, fmt.Errorf("parse results.json: %w", err)
	}
	if testReport.Summary.Profile == "" {
		summaryRaw, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
		if err != nil {
			return testReport, fmt.Errorf("read summary.json: %w", err)
		}
		if err := json.Unmarshal(summaryRaw, &testReport.Summary); err != nil {
			return testReport, fmt.Errorf("parse summary.json: %w", err)
		}
	}
	return testReport, nil
}

func LatestDir(repo string) (string, error) {
	pattern := filepath.Join(repo, "test-results", "aicoding-global-test-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	type candidate struct {
		path    string
		modTime time.Time
	}
	candidates := []candidate{}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(match, "summary.json")); err != nil {
			continue
		}
		candidates = append(candidates, candidate{path: match, modTime: info.ModTime()})
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no test-results/aicoding-global-test-* report found")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].path > candidates[j].path
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	return candidates[0].path, nil
}
