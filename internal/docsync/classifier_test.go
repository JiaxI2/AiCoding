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
