package todolist

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListMissingDirIsEmptyNotError(t *testing.T) {
	report, err := List(t.TempDir())
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if report.Summary.Total != 0 || len(report.Items) != 0 {
		t.Fatalf("expected empty report: %#v", report)
	}
}

func TestListParsesHeaderAndSummarizesDeterministically(t *testing.T) {
	repo := t.TempDir()
	write(t, repo, Dir+"/README.md", "# 说明\n\n忽略我\n")
	write(t, repo, Dir+"/0002-b.md", "# TODO 0002: B\n\nStatus: Done\nVerify: go test ./x\n\n正文\n")
	write(t, repo, Dir+"/0001-a.md", "# TODO 0001: A\n\nStatus: Planned\nVerify: go test ./y\n\n正文\n")

	report, err := List(repo)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Total != 2 || report.Summary.Planned != 1 || report.Summary.Done != 1 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
	// README.md is excluded; items are sorted by filename.
	if report.Items[0].File != "0001-a.md" || report.Items[1].File != "0002-b.md" {
		t.Fatalf("items not sorted or README leaked: %#v", report.Items)
	}
	if report.Items[0].Title != "TODO 0001: A" || report.Items[0].Status != StatusPlanned || report.Items[0].Verify != "go test ./y" {
		t.Fatalf("header not parsed: %#v", report.Items[0])
	}

	// Determinism: a second call yields an identical report.
	again, _ := List(repo)
	if report.Summary != again.Summary || len(report.Items) != len(again.Items) {
		t.Fatal("List is not deterministic")
	}
}

func TestNormalizeStatusAliases(t *testing.T) {
	cases := map[string]string{
		"Planned": StatusPlanned, "wip": StatusInProgress, "in progress": StatusInProgress,
		"green": StatusDone, "done": StatusDone, "": "Unknown", "nonsense": "Unknown",
	}
	for in, want := range cases {
		if got := normalizeStatus(in); got != want {
			t.Fatalf("normalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}
