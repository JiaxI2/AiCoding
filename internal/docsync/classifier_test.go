package docsync

import "testing"

func TestDocPathClassifiers(t *testing.T) {
	if !IsDocPath("docs/FAST.md") || !IsDocPath("README.md") {
		t.Fatalf("doc path classifier rejected known docs")
	}
	if !IsDocSyncRiskPath("tools/specialty/test.ps1") || !IsDocSyncRiskPath(".github/workflows/aicoding-ci.yml") {
		t.Fatalf("risk path classifier rejected known risk paths")
	}
	if IsDocSyncRiskPath("docs/FAST.md") {
		t.Fatalf("doc path should not be treated as risk source path")
	}
}

func TestCommandControlSurfacesRequireDocumentationReview(t *testing.T) {
	for _, path := range []string{
		"cmd/aicoding/main.go",
		"internal/cli/cli.go",
		"internal/testengine/engine.go",
		"Taskfile.yml",
		".github/workflows/aicoding-ci.yml",
	} {
		if !IsDocSyncRiskPath(path) {
			t.Fatalf("command control surface must require documentation review: %s", path)
		}
	}
}
