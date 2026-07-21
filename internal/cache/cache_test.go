package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

func TestStatusReportsRegisteredScopesAndTotals(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, filepath.Join(repo, ".aicoding", "cache", "fast-path", "state.json"), "{}")
	writeFixture(t, filepath.Join(repo, "test-results", "aicoding-global-test-one", "summary.json"), `{"conclusion":"PASS"}`)
	writeFixture(t, filepath.Join(repo, ".aicoding", "state", "work", "job", "attempts.jsonl"), "{}\n")

	status, err := Status(repo)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Scopes) != 4 {
		t.Fatalf("scope count = %d, want 4: %#v", len(status.Scopes), status.Scopes)
	}
	wantScopes := []Scope{ScopeFastPath, ScopeTestResults, ScopeValidationReports, ScopeWorkState}
	for index, want := range wantScopes {
		if status.Scopes[index].Scope != want || status.Scopes[index].Path == "" || status.Scopes[index].Policy == "" {
			t.Fatalf("scope[%d] = %#v, want populated %q", index, status.Scopes[index], want)
		}
	}
	if status.TotalEntryCount != 3 {
		t.Fatalf("total entry count = %d, want 3", status.TotalEntryCount)
	}
	if status.TotalSizeBytes <= 0 {
		t.Fatalf("total size = %d, want positive", status.TotalSizeBytes)
	}
}

func TestCleanTestResultsDryRunMatchesRemovalAndRetainsFailure(t *testing.T) {
	repo := t.TempDir()
	base := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	for index := 0; index < 8; index++ {
		conclusion := "PASS"
		if index == 6 {
			conclusion = "FAIL"
		}
		writeTestResultFixture(t, repo, index, conclusion, base.Add(time.Duration(index)*time.Minute))
	}
	before := testResultNames(t, repo)

	dryRun, err := Clean(repo, CleanOptions{Scope: ScopeTestResults, Keep: 5, DryRun: true})
	if err != nil {
		t.Fatalf("dry-run Clean: %v", err)
	}
	if dryRun.PlannedCount != 3 || dryRun.RemovedCount != 0 || len(dryRun.Scopes) != 1 {
		t.Fatalf("unexpected dry-run result: %#v", dryRun)
	}
	if after := testResultNames(t, repo); !reflect.DeepEqual(after, before) {
		t.Fatalf("dry-run changed files: before=%v after=%v", before, after)
	}
	planned := cleanEntryPaths(dryRun.Scopes[0].Planned)
	if containsSuffix(planned, "-06") {
		t.Fatalf("FAIL result was planned for deletion: %v", planned)
	}

	removed, err := Clean(repo, CleanOptions{Scope: ScopeTestResults, Keep: 5})
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if removed.RemovedCount != 3 || removed.PlannedCount != 3 {
		t.Fatalf("unexpected removal result: %#v", removed)
	}
	if actual := cleanEntryPaths(removed.Scopes[0].Removed); !reflect.DeepEqual(actual, planned) {
		t.Fatalf("dry-run and removal differ: dry=%v actual=%v", planned, actual)
	}
	remaining := testResultNames(t, repo)
	if len(remaining) != 5 || !containsSuffix(remaining, "-07") || !containsSuffix(remaining, "-06") {
		t.Fatalf("retained results = %v, want latest and FAIL among five", remaining)
	}
}

func TestCleanFastPathRemovesOnlyRegisteredRoot(t *testing.T) {
	repo := t.TempDir()
	cacheRoot := filepath.Join(repo, ".aicoding", "cache", "fast-path")
	writeFixture(t, filepath.Join(cacheRoot, "state.json"), "{}")
	unrelated := filepath.Join(repo, ".aicoding", "cache", "other", "state.json")
	writeFixture(t, unrelated, "keep")

	result, err := Clean(repo, CleanOptions{Scope: ScopeFastPath})
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedCount != 1 || directoryExists(cacheRoot) {
		t.Fatalf("fast-path clean result = %#v, root exists=%t", result, directoryExists(cacheRoot))
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unregistered cache path was changed: %v", err)
	}
}

func TestCleanTestResultsRetainsOldFailureOutsideKeepWindow(t *testing.T) {
	repo := t.TempDir()
	base := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	for index := 0; index < 8; index++ {
		conclusion := "PASS"
		if index == 0 {
			conclusion = "FAIL"
		}
		writeTestResultFixture(t, repo, index, conclusion, base.Add(time.Duration(index)*time.Minute))
	}

	result, err := Clean(repo, CleanOptions{Scope: ScopeTestResults, Keep: 5})
	if err != nil {
		t.Fatal(err)
	}
	remaining := testResultNames(t, repo)
	if result.RemovedCount != 2 || len(remaining) != 6 || !containsSuffix(remaining, "-00") {
		t.Fatalf("old FAIL was not retained: result=%#v remaining=%v", result, remaining)
	}
}

func TestCleanValidationReportsRetainsReceiptAndAliasReferences(t *testing.T) {
	repo := newGitRepo(t)
	writeFixture(t, filepath.Join(repo, "tracked.txt"), "tracked\n")
	mustGit(t, repo, "add", "tracked.txt")
	mustGit(t, repo, "commit", "-m", "initial")
	store, err := validationevidence.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(validationevidence.TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := store.Fingerprint(subject, validationevidence.FingerprintSpec{
		Profile: "smoke", ValidationPlanDigest: testDigest("plan"),
		EngineSemanticDigest: testDigest("engine"), OptionsDigest: testDigest("options"),
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := store.Put(validationevidence.Receipt{
		ValidationIdentity: fingerprint.Identity, Fingerprint: fingerprint,
		Conclusion: "PASS", ResultsDigest: testDigest("results"), Reusable: true,
		Scope: validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}, validationevidence.ReportBundle{
		ResultsJSON: []byte(`{"results":[]}`), SummaryJSON: []byte(`{"conclusion":"PASS"}`), ReportMarkdown: []byte("# PASS\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.BindCommit("HEAD", receipt); err != nil {
		t.Fatal(err)
	}
	reportRoot := filepath.Join(repo, ".git", "aicoding", "validation", "reports")
	referencedDir := filepath.Join(reportRoot, strings.TrimPrefix(receipt.ValidationIdentity, "sha256:"))
	orphanDir := filepath.Join(reportRoot, strings.Repeat("f", 64))
	writeFixture(t, filepath.Join(orphanDir, "report.md"), "orphan\n")

	result, err := Clean(repo, CleanOptions{Scope: ScopeValidationReports})
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedCount != 1 || !directoryExists(referencedDir) || directoryExists(orphanDir) {
		t.Fatalf("Receipt retention failed: result=%#v referenced=%t orphan=%t", result, directoryExists(referencedDir), directoryExists(orphanDir))
	}

	receiptPath := filepath.Join(repo, ".git", "aicoding", "validation", "receipts", "smoke", strings.TrimPrefix(receipt.ValidationIdentity, "sha256:")+".json")
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(orphanDir, "report.md"), "orphan-again\n")
	result, err = Clean(repo, CleanOptions{Scope: ScopeValidationReports})
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedCount != 1 || !directoryExists(referencedDir) || directoryExists(orphanDir) {
		t.Fatalf("alias retention failed: result=%#v referenced=%t orphan=%t", result, directoryExists(referencedDir), directoryExists(orphanDir))
	}
}

func TestWorkStateIsAuditOnly(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, filepath.Join(repo, ".aicoding", "state", "work", "job", "attempts.jsonl"), "{}\n")
	if _, err := Clean(repo, CleanOptions{Scope: ScopeWorkState}); err == nil || !strings.Contains(err.Error(), "audit-only") {
		t.Fatalf("Clean work-state error = %v, want audit-only refusal", err)
	}
}

func TestBloatWarningsUsesTestResultThreshold(t *testing.T) {
	repo := t.TempDir()
	for index := 0; index < 21; index++ {
		if err := os.MkdirAll(filepath.Join(repo, "test-results", fmt.Sprintf("aicoding-global-test-%02d", index)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	status, err := Status(repo)
	if err != nil {
		t.Fatal(err)
	}
	warnings := BloatWarnings(status)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "cache clean --scope test-results") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func writeTestResultFixture(t *testing.T, repo string, index int, conclusion string, timestamp time.Time) {
	t.Helper()
	path := filepath.Join(repo, "test-results", fmt.Sprintf("aicoding-global-test-%02d", index), "summary.json")
	writeFixture(t, path, fmt.Sprintf("{\"conclusion\":%q}\n", conclusion))
	if err := os.Chtimes(path, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
}

func testResultNames(t *testing.T, repo string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repo, "test-results"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "aicoding-global-test-") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

func cleanEntryPaths(entries []CleanEntry) []string {
	paths := make([]string, len(entries))
	for index := range entries {
		paths[index] = entries[index].Path
	}
	sort.Strings(paths)
	return paths
}

func containsSuffix(values []string, suffix string) bool {
	for _, value := range values {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "test@example.com")
	mustGit(t, repo, "config", "user.name", "Test User")
	return repo
}

func mustGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func testDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("sha256:%x", digest)
}
