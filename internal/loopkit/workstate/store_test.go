package workstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/loopkit/transition"
)

func TestRecordAppendsAndLoadsSession(t *testing.T) {
	repo := t.TempDir()
	when := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	attempt := transition.Attempt{
		Number: 1, SubjectTreeOID: "tree-1", StartedAt: when, EndedAt: when.Add(time.Second),
	}
	decision := transition.Decision{State: transition.Continue, Attempt: 2, Reason: "pending"}
	session, err := Record(repo, "work-1", "sha256:spec", "spec.json", attempt, decision)
	if err != nil {
		t.Fatal(err)
	}
	if !session.Exists || session.Snapshot.Attempts != 1 || len(session.History) != 1 {
		t.Fatalf("unexpected session: %#v", session)
	}
	loaded, err := Load(repo, "work-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Snapshot.SpecDigest != "sha256:spec" || loaded.History[0].SubjectTreeOID != "tree-1" {
		t.Fatalf("unexpected loaded session: %#v", loaded)
	}
}

func TestRecordRejectsSpecDriftAndAttemptGap(t *testing.T) {
	repo := t.TempDir()
	when := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	first := transition.Attempt{Number: 1, SubjectTreeOID: "tree-1", StartedAt: when, EndedAt: when.Add(time.Second)}
	if _, err := Record(repo, "work-1", "sha256:a", "spec.json", first, transition.Decision{}); err != nil {
		t.Fatal(err)
	}
	second := transition.Attempt{Number: 3, SubjectTreeOID: "tree-2", StartedAt: when, EndedAt: when.Add(time.Second)}
	if _, err := Record(repo, "work-1", "sha256:a", "spec.json", second, transition.Decision{}); err == nil {
		t.Fatal("expected attempt gap rejection")
	}
	second.Number = 2
	if _, err := Record(repo, "work-1", "sha256:b", "spec.json", second, transition.Decision{}); err == nil {
		t.Fatal("expected spec drift rejection")
	}
}

func TestRecordReplacesStateProjectionAfterSecondAppend(t *testing.T) {
	repo := t.TempDir()
	when := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	first := transition.Attempt{Number: 1, SubjectTreeOID: "tree-1", StartedAt: when, EndedAt: when.Add(time.Second)}
	if _, err := Record(repo, "work-1", "sha256:a", "spec.json", first, transition.Decision{State: transition.Continue}); err != nil {
		t.Fatal(err)
	}
	second := transition.Attempt{Number: 2, SubjectTreeOID: "tree-2", StartedAt: when.Add(2 * time.Second), EndedAt: when.Add(3 * time.Second)}
	session, err := Record(repo, "work-1", "sha256:a", "spec.json", second, transition.Decision{State: transition.StopSatisfied})
	if err != nil {
		t.Fatal(err)
	}
	if session.Snapshot.Attempts != 2 || session.Snapshot.LastSubjectTreeOID != "tree-2" || len(session.History) != 2 {
		t.Fatalf("unexpected second projection: %#v", session)
	}
}

func TestLoadFailsClosedOnTruncatedAttempt(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, ".aicoding", "state", "work", "work-1")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "attempts.jsonl"), []byte("{\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(repo, "work-1"); err == nil {
		t.Fatal("expected corrupt attempt log to fail closed")
	}
}
