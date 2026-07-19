package repocontext

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkScan measures the single-walk cost of the scan primitive so its
// performance is independently observable and regressions are caught. Per the
// architecture constitution every Primitive must be independently benchmarkable.
func BenchmarkScan(b *testing.B) {
	repo := benchRepo(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Scan(repo); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReconcileNoOp measures the cost of a converged update: reconcile must
// write nothing when content is unchanged, so this reflects the steady-state
// (post-commit sync on an already-fresh tree) fast path.
func BenchmarkReconcileNoOp(b *testing.B) {
	repo := benchRepo(b)
	if r := Install(repo, false); !r.OK {
		b.Fatalf("install failed: %#v", r)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if r := Update(repo, false); !r.OK {
			b.Fatalf("update failed: %#v", r)
		}
	}
}

func benchRepo(b *testing.B) string {
	b.Helper()
	repo := b.TempDir()
	writeBench(b, repo, "go.mod", "module github.com/example/bench\n")
	for _, dir := range []string{"cmd", "internal", "pkg", "docs", "config"} {
		for i := 0; i < 20; i++ {
			writeBench(b, repo, dir+"/f"+string(rune('a'+i%26))+".go", "package x\n")
		}
	}
	return repo
}

func writeBench(b *testing.B, repo, rel, content string) {
	b.Helper()
	full := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}
}
