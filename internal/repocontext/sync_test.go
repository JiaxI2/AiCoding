package repocontext

import (
	"os"
	"path/filepath"
	"testing"
)

func readOwned(t *testing.T, repo, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(content)
}

func TestSyncNoOpWhenNotInstalled(t *testing.T) {
	repo := sampleRepo(t)
	report := Sync(repo, []string{"internal/core/core.go"}, false)
	if !report.OK || report.Status != "not-installed" || report.Installed {
		t.Fatalf("expected quiet no-op: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(ownedRoot))); !os.IsNotExist(err) {
		t.Fatalf("sync created owned root without install: %v", err)
	}
}

func TestSyncWritesOnlyChangedDomainAndReconvergesFresh(t *testing.T) {
	repo := sampleRepo(t)
	if r := Install(repo, false); !r.OK {
		t.Fatalf("install failed: %#v", r)
	}
	// cmd/ domain is unaffected by a change under internal/; capture its bytes.
	cmdBefore := readOwned(t, repo, ownedRoot+"/domains/cmd.md")

	// Add a source file under internal/, then sync with that changed path.
	writeFile(t, repo, "internal/added/added.go", "package added\n")
	report := Sync(repo, []string{"internal/added/added.go"}, false)
	if !report.OK || report.Status != "ok" || !report.Fresh {
		t.Fatalf("sync failed: %#v", report)
	}

	// The unaffected domain file must be byte-identical (never rewritten).
	if got := readOwned(t, repo, ownedRoot+"/domains/cmd.md"); got != cmdBefore {
		t.Fatalf("unaffected domain context was rewritten by sync")
	}
	// The affected domain file must now reflect the new file count.
	internalDoc := readOwned(t, repo, ownedRoot+"/domains/internal.md")
	if internalDoc == "" {
		t.Fatal("internal domain doc missing after sync")
	}
	// Only the affected domain (plus the index which tracks per-domain counts)
	// should have been written.
	wroteInternal := false
	for _, w := range report.Files {
		if w == ownedRoot+"/domains/internal.md" {
			wroteInternal = true
		}
		if w == ownedRoot+"/domains/cmd.md" {
			t.Fatalf("sync rewrote the unaffected cmd domain: %v", report.Files)
		}
	}
	if !wroteInternal {
		t.Fatalf("sync did not rewrite the affected internal domain: %v", report.Files)
	}
	// Status must now report fresh.
	if s := Status(repo); s.Status != "fresh" {
		t.Fatalf("expected fresh after sync: %#v", s)
	}
}

func TestSyncRemovesDomainWhenAllFilesGone(t *testing.T) {
	repo := sampleRepo(t)
	if r := Install(repo, false); !r.OK {
		t.Fatalf("install failed: %#v", r)
	}
	docsDoc := filepath.Join(repo, filepath.FromSlash(ownedRoot+"/domains/docs.md"))
	if _, err := os.Stat(docsDoc); err != nil {
		t.Fatalf("expected docs domain doc: %v", err)
	}
	// Remove the only file under docs/, then sync.
	if err := os.Remove(filepath.Join(repo, "docs", "README.md")); err != nil {
		t.Fatal(err)
	}
	report := Sync(repo, []string{"docs/README.md"}, false)
	if !report.OK {
		t.Fatalf("sync failed: %#v", report)
	}
	if _, err := os.Stat(docsDoc); !os.IsNotExist(err) {
		t.Fatalf("sync did not remove context for the emptied domain: %v", err)
	}
}

func TestAffectedDomainsIgnoresRootAndOwnedPaths(t *testing.T) {
	got := affectedDomains([]string{
		"internal/core/core.go",
		"internal/cli/cli.go",
		"README.md",             // root file → ignored
		ownedRoot + "/index.md", // owned artifact → ignored
		".git/COMMIT_EDITMSG",   // skip dir → ignored
		"docs\\guide.md",        // backslash normalized
	})
	want := map[string]bool{"internal": true, "docs": true}
	if len(got) != len(want) {
		t.Fatalf("affected = %v, want keys %v", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Fatalf("unexpected affected domain %q in %v", d, got)
		}
	}
}
