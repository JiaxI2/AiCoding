package adrreview

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func write(t testing.TB, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFlagsRequiredWithoutChecklistOnly(t *testing.T) {
	repo := t.TempDir()
	// Compliant: declares required and has the §12 section.
	write(t, repo, Dir+"/0001-good.md", "# ADR 0001: good\n\nPrimitiveReview: required\n\n## Status\n\n## §12 Checklist 自评\n\n- ok\n")
	// Violating: declares required, section missing.
	write(t, repo, Dir+"/0002-bad.md", "# ADR 0002: bad\n\nPrimitiveReview: required\n\n## Status\n")
	// Historical: no header — ignored, never a gap.
	write(t, repo, Dir+"/0003-legacy.md", "# ADR 0003: legacy\n\n## Status\n")
	// Opt-out: explicitly n/a — ignored.
	write(t, repo, Dir+"/0004-na.md", "# ADR 0004: na\n\nPrimitiveReview: n/a\n\n## Status\n")
	// Plan-artifact subfolders are not ADRs and must not be scanned.
	write(t, repo, Dir+"/some-topic/IMPLEMENTATION_PLAN.md", "PrimitiveReview: required\n")

	report, err := Check(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Items) != 4 {
		t.Fatalf("items = %d, want 4 (subfolder excluded): %#v", len(report.Items), report.Items)
	}
	if len(report.Gaps) != 1 || report.Gaps[0][:12] != "0002-bad.md:" {
		t.Fatalf("expected exactly the one gap for 0002-bad.md: %#v", report.Gaps)
	}
	// Deterministic ordering by filename.
	if report.Items[0].File != "0001-good.md" || !report.Items[0].OK || !report.Items[0].HasChecklist {
		t.Fatalf("unexpected first item: %#v", report.Items[0])
	}
}

func TestCheckMissingDirIsEmptyNotError(t *testing.T) {
	report, err := Check(t.TempDir())
	if err != nil || len(report.Items) != 0 || len(report.Gaps) != 0 {
		t.Fatalf("missing dir should be an empty report: %#v err=%v", report, err)
	}
}

// BenchmarkCheck makes the primitive's bounded cost observable: it reads only
// docs/decisions/*.md, independent of repository size (constitution #3/#10).
func BenchmarkCheck(b *testing.B) {
	repo := b.TempDir()
	for i := 0; i < 20; i++ {
		write(b, repo, fmt.Sprintf("%s/%04d-adr.md", Dir, i),
			"# ADR: x\n\nPrimitiveReview: required\n\n## §12 Checklist 自评\n\n- ok\n")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Check(repo); err != nil {
			b.Fatal(err)
		}
	}
}
