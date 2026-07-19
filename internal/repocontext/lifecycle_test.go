package repocontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sampleRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, repo, "go.mod", "module github.com/example/widget\n\ngo 1.22\n")
	writeFile(t, repo, "cmd/widget/main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, repo, "internal/core/core.go", "package core\n")
	writeFile(t, repo, "internal/core/util.go", "package core\n")
	writeFile(t, repo, "docs/README.md", "# docs\n")
	return repo
}

func TestScanIsDeterministic(t *testing.T) {
	repo := sampleRepo(t)
	_, first, err := Scan(repo)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := Scan(repo)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() || !strings.HasPrefix(first.Digest(), "sha256:") {
		t.Fatalf("scan digest unstable: %q != %q", first.Digest(), second.Digest())
	}
}

func TestScanExtractsFactsAndSkipsOwnedRoot(t *testing.T) {
	repo := sampleRepo(t)
	// Pre-seed a stale artifact under the owned root; it must not affect facts.
	writeFile(t, repo, ownedRoot+"/domains/stale.md", "# stale\n")
	facts, _, err := Scan(repo)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Repo != "widget" {
		t.Fatalf("repo name = %q, want widget", facts.Repo)
	}
	domainPaths := map[string]int{}
	for _, d := range facts.Domains {
		domainPaths[d.Path] = d.Files
	}
	if _, ok := domainPaths[".aicoding"]; ok {
		t.Fatalf("owned root leaked into domains: %#v", facts.Domains)
	}
	if domainPaths["internal"] != 2 {
		t.Fatalf("internal domain files = %d, want 2", domainPaths["internal"])
	}
	if !containsToolchain(facts.Toolchains, "Go modules") {
		t.Fatalf("expected Go modules toolchain: %#v", facts.Toolchains)
	}
}

func TestInstallGeneratesOwnedArtifactsAndManifest(t *testing.T) {
	repo := sampleRepo(t)
	report := Install(repo, false)
	if !report.OK || report.Status != "ok" || !report.Installed {
		t.Fatalf("install failed: %#v", report)
	}
	index := filepath.Join(repo, filepath.FromSlash(ownedRoot+"/index.md"))
	if _, err := os.Stat(index); err != nil {
		t.Fatalf("index not generated: %v", err)
	}
	manifest, err := loadManifest(repo)
	if err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
	if manifest.FactsDigest != report.FactsDigest || len(manifest.Files) == 0 {
		t.Fatalf("manifest mismatch: %#v", manifest)
	}
	// Every domain file plus the index should be recorded.
	for _, f := range manifest.Files {
		if !strings.HasPrefix(f.Path, ownedRoot+"/") {
			t.Fatalf("manifest file outside owned root: %s", f.Path)
		}
	}
}

func TestStatusReportsFreshThenDriftAfterCodeChange(t *testing.T) {
	repo := sampleRepo(t)
	if r := Install(repo, false); !r.OK {
		t.Fatalf("install failed: %#v", r)
	}
	fresh := Status(repo)
	if !fresh.OK || fresh.Status != "fresh" || !fresh.Fresh {
		t.Fatalf("expected fresh status: %#v", fresh)
	}
	// Add a new source file: facts change, generated context becomes stale.
	writeFile(t, repo, "internal/added/added.go", "package added\n")
	drift := Status(repo)
	if drift.Status != "drift" || drift.Fresh {
		t.Fatalf("expected drift status: %#v", drift)
	}
	// Update reconverges to fresh.
	if r := Update(repo, false); !r.OK {
		t.Fatalf("update failed: %#v", r)
	}
	if again := Status(repo); again.Status != "fresh" {
		t.Fatalf("expected fresh after update: %#v", again)
	}
}

func TestDoctorDetectsTampering(t *testing.T) {
	repo := sampleRepo(t)
	if r := Install(repo, false); !r.OK {
		t.Fatalf("install failed: %#v", r)
	}
	if ok := Doctor(repo); !ok.OK {
		t.Fatalf("doctor should pass on clean install: %#v", ok)
	}
	index := filepath.Join(repo, filepath.FromSlash(ownedRoot+"/index.md"))
	if err := os.WriteFile(index, []byte("hand edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tampered := Doctor(repo)
	if tampered.OK || len(tampered.Errors) == 0 {
		t.Fatalf("doctor should flag tampering: %#v", tampered)
	}
}

func TestUninstallRemovesOnlyOwnedArtifacts(t *testing.T) {
	repo := sampleRepo(t)
	// A user-authored file inside the owned root must survive uninstall because
	// it is not recorded in the manifest.
	if r := Install(repo, false); !r.OK {
		t.Fatalf("install failed: %#v", r)
	}
	userFile := ownedRoot + "/domains/user-note.md"
	writeFile(t, repo, userFile, "# my note\n")

	report := Uninstall(repo, false)
	if !report.OK || report.Installed {
		t.Fatalf("uninstall failed: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(ownedRoot+"/index.md"))); !os.IsNotExist(err) {
		t.Fatalf("owned index survived uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(userFile))); err != nil {
		t.Fatalf("user-authored file was deleted by uninstall: %v", err)
	}
	if _, err := os.Stat(manifestPath(repo)); !os.IsNotExist(err) {
		t.Fatalf("manifest survived uninstall: %v", err)
	}
}

func TestPlanDoesNotWrite(t *testing.T) {
	repo := sampleRepo(t)
	plan := Install(repo, true)
	if !plan.OK || plan.Status != "planned" || len(plan.Planned) == 0 {
		t.Fatalf("plan failed: %#v", plan)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(ownedRoot))); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote owned root: %v", err)
	}
}

func containsToolchain(toolchains []string, want string) bool {
	for _, tc := range toolchains {
		if tc == want {
			return true
		}
	}
	return false
}
