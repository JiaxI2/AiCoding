package repohealth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestScanPwshBudgetClassifiesInvocationPoints(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "Taskfile.yml"), "tasks:\n  smoke:\n    cmds:\n      - bin/aicoding.exe kit verify --all --profile Smoke --json\n  full:\n    cmds:\n      - bin/aicoding.exe full --json\n")
	mustWrite(t, filepath.Join(repo, ".githooks", "pre-commit"), "bin/aicoding.exe hook pre-commit --json\n")
	mustWrite(t, filepath.Join(repo, "docs", "POWERSHELL_BOUNDARY.md"), "pwsh -File tools/specialty/verify-release-governance-overlay.ps1\n")
	writePwshBudgetConfig(t, repo, []pwshBudgetBaseline{pwshTestBaseline(nil, "a")})

	budget, errs := ScanPwshBudget(repo)
	if len(errs) != 0 {
		t.Fatalf("ScanPwshBudget errs = %v", errs)
	}
	if budget.Counts["hot-path"] != 0 || budget.Counts["slow-path"] != 0 || budget.Counts["fallback"] != 0 {
		t.Fatalf("expected no PowerShell hot/slow/fallback routes after Go routing, got %#v", budget.Counts)
	}
	if budget.Counts["documentation-only"] == 0 {
		t.Fatalf("expected documentation-only PowerShell inventory, got %#v", budget.Counts)
	}
	if !budget.Ratchet.OK || budget.Ratchet.BaselineRemainingScripts != 0 {
		t.Fatalf("empty ratchet baseline = %#v", budget.Ratchet)
	}
}

func TestPwshBudgetRatchetRejectsNewScriptsAndAllowsStrictDecrease(t *testing.T) {
	repo := t.TempDir()
	one := "tools/specialty/one.ps1"
	newScript := "tools/specialty/new.ps1"
	mustWrite(t, filepath.Join(repo, filepath.FromSlash(one)), "# RETIRE-AFTER: next release\nWrite-Output one\n")
	first := pwshTestBaseline([]string{one}, "a")
	writePwshBudgetConfig(t, repo, []pwshBudgetBaseline{first})

	budget, errs := ScanPwshBudget(repo)
	if len(errs) != 0 || !budget.Ratchet.OK {
		t.Fatalf("matching ratchet failed: budget=%#v errs=%v", budget.Ratchet, errs)
	}

	mustWrite(t, filepath.Join(repo, filepath.FromSlash(newScript)), "Write-Output new\n")
	budget, errs = ScanPwshBudget(repo)
	joined := strings.Join(errs, "\n")
	if budget.Ratchet.OK || !strings.Contains(joined, "exceeds ratchet baseline: "+newScript) ||
		!strings.Contains(joined, "missing # RETIRE-AFTER: "+newScript) {
		t.Fatalf("unmarked new script was not rejected with its path: budget=%#v errs=%v", budget.Ratchet, errs)
	}

	mustWrite(t, filepath.Join(repo, filepath.FromSlash(newScript)), "# RETIRE-AFTER: next release\nWrite-Output new\n")
	_, errs = ScanPwshBudget(repo)
	joined = strings.Join(errs, "\n")
	if !strings.Contains(joined, "exceeds ratchet baseline: "+newScript) || strings.Contains(joined, "missing # RETIRE-AFTER") {
		t.Fatalf("marked new script did not fail only the count/path ratchet: %v", errs)
	}

	if err := os.Remove(filepath.Join(repo, filepath.FromSlash(newScript))); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo, filepath.FromSlash(one))); err != nil {
		t.Fatal(err)
	}
	second := pwshTestBaseline(nil, "b")
	writePwshBudgetConfig(t, repo, []pwshBudgetBaseline{first, second})
	budget, errs = ScanPwshBudget(repo)
	if len(errs) != 0 || !budget.Ratchet.OK || budget.Ratchet.BaselineRemainingScripts != 0 {
		t.Fatalf("strict decrease did not converge: budget=%#v errs=%v", budget.Ratchet, errs)
	}
}

func TestPwshBudgetRatchetRejectsBaselineIncreaseOrReplacement(t *testing.T) {
	repo := t.TempDir()
	one := "tools/specialty/one.ps1"
	two := "tools/specialty/two.ps1"
	for _, path := range []string{one, two} {
		mustWrite(t, filepath.Join(repo, filepath.FromSlash(path)), "# RETIRE-AFTER: next release\n")
	}
	first := pwshTestBaseline([]string{one}, "a")
	increased := pwshTestBaseline([]string{one, two}, "b")
	writePwshBudgetConfig(t, repo, []pwshBudgetBaseline{first, increased})

	_, errs := ScanPwshBudget(repo)
	joined := strings.Join(errs, "\n")
	if !strings.Contains(joined, "is not a strict subset; added path: "+two) ||
		!strings.Contains(joined, "must lower remainingScripts below 1") {
		t.Fatalf("baseline increase was not rejected: %v", errs)
	}

	replacement := pwshTestBaseline([]string{two}, "c")
	writePwshBudgetConfig(t, repo, []pwshBudgetBaseline{first, replacement})
	_, errs = ScanPwshBudget(repo)
	joined = strings.Join(errs, "\n")
	if !strings.Contains(joined, "is not a strict subset; added path: "+two) ||
		!strings.Contains(joined, "must lower remainingScripts below 1") {
		t.Fatalf("same-count replacement was not rejected: %v", errs)
	}
}

func TestPwshBudgetRatchetFailsClosedOnMissingOrCorruptConfig(t *testing.T) {
	repo := t.TempDir()
	if _, errs := ScanPwshBudget(repo); len(errs) == 0 || !strings.Contains(strings.Join(errs, "\n"), "config is missing or unreadable") {
		t.Fatalf("missing config did not fail closed: %v", errs)
	}
	mustWrite(t, filepath.Join(repo, pwshBudgetConfigPath), "{corrupt\n")
	if _, errs := ScanPwshBudget(repo); len(errs) == 0 || !strings.Contains(strings.Join(errs, "\n"), "config is invalid") {
		t.Fatalf("corrupt config did not fail closed: %v", errs)
	}
}

func pwshTestBaseline(scripts []string, commitChar string) pwshBudgetBaseline {
	scripts = append([]string(nil), scripts...)
	sort.Strings(scripts)
	return pwshBudgetBaseline{
		RemainingScripts: len(scripts),
		Unspecified:      0,
		Scripts:          scripts,
		ObservedCommit:   strings.Repeat(commitChar, 40),
		Evidence:         "docs/operations/evidence/pwsh-budget-" + commitChar + ".json",
	}
}

func writePwshBudgetConfig(t *testing.T, repo string, baselines []pwshBudgetBaseline) {
	t.Helper()
	for _, baseline := range baselines {
		evidence := map[string]any{
			"schemaVersion": 1,
			"command":       "doctor pwsh",
			"ok":            true,
			"data": map[string]any{"retirement": map[string]any{
				"scope": pwshBudgetScope, "remainingScripts": baseline.RemainingScripts, "unspecified": baseline.Unspecified,
			}},
		}
		raw, err := json.Marshal(evidence)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(repo, filepath.FromSlash(baseline.Evidence)), string(raw)+"\n")
	}
	config := pwshBudgetConfig{SchemaVersion: 1, Scope: pwshBudgetScope, Baselines: baselines}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, pwshBudgetConfigPath), string(raw)+"\n")
}
