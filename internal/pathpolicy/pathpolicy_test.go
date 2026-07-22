package pathpolicy

import (
	"reflect"
	"testing"
)

func TestCompileSortsDeduplicatesAndMatchesFrozenDialect(t *testing.T) {
	compiled, err := Compile([]string{"z/**", "internal/cli/**", "z/**", "docs/*.md"})
	if err != nil {
		t.Fatal(err)
	}
	values := make([]string, 0, len(compiled))
	for _, pattern := range compiled {
		values = append(values, pattern.Value)
	}
	if want := []string{"docs/*.md", "internal/cli/**", "z/**"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("compiled values = %#v, want %#v", values, want)
	}

	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"internal/cli/**", "internal/cli/x/y.go", true},
		{"internal/cli/**", "internal/clix/y.go", false},
		{"docs/*.md", "docs/README.md", true},
		{"docs/*.md", "docs/architecture/README.md", false},
		{"internal/**/*.go", "internal/plan/plan.go", true},
		{"internal/?.go", "internal/x.go", true},
	}
	for _, test := range tests {
		patterns, err := Compile([]string{test.pattern})
		if err != nil {
			t.Fatalf("Compile(%q): %v", test.pattern, err)
		}
		got, err := Match(patterns[0], test.path)
		if err != nil || got != test.want {
			t.Fatalf("Match(%q, %q) = %v, %v; want %v", test.pattern, test.path, got, err, test.want)
		}
	}
}

func TestValidateFailsClosed(t *testing.T) {
	for _, patterns := range [][]string{{"../**"}, {"/internal/**"}, {"docs/[ab].md"}, {""}} {
		if err := Validate(patterns); err == nil {
			t.Fatalf("Validate(%#v) unexpectedly passed", patterns)
		}
	}
	compiled, err := Compile([]string{"internal/**"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Match(compiled[0], "../outside.go"); err == nil {
		t.Fatal("Match accepted traversal")
	}
}

func BenchmarkCompileAndMatch(b *testing.B) {
	patterns := []string{"docs/**", "internal/**/*.go", "config/**", ".github/**"}
	for index := 0; index < b.N; index++ {
		compiled, err := Compile(patterns)
		if err != nil {
			b.Fatal(err)
		}
		for _, pattern := range compiled {
			if _, err := Match(pattern, "internal/cli/x/y.go"); err != nil {
				b.Fatal(err)
			}
		}
	}
}
