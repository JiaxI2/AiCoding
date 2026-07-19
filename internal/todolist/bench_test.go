package todolist

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkList measures the cost of the primitive over a realistic todo set.
// It reads only docs/todolist/, so its cost is independent of repository size —
// this benchmark makes that bound observable (constitution #3/#10).
func BenchmarkList(b *testing.B) {
	repo := b.TempDir()
	for i := 0; i < 30; i++ {
		rel := fmt.Sprintf("%s/%04d-item.md", Dir, i)
		full := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			b.Fatal(err)
		}
		content := fmt.Sprintf("# TODO %04d: item\n\nStatus: Planned\nVerify: go test ./x\n\nbody\n", i)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := List(repo); err != nil {
			b.Fatal(err)
		}
	}
}
