package pwshregex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDoubleQuotedCaptureReplacementFails(t *testing.T) {
	issues := LintText("bad.ps1", `$x = $text -creplace '([a-z]+)(\d+)', "$1-$2"`)
	if len(BlockingMessages(issues)) != 1 {
		t.Fatalf("expected one blocking issue, got %#v", issues)
	}
}

func TestSingleQuotedCaptureReplacementPasses(t *testing.T) {
	issues := LintText("good.ps1", `$x = $text -creplace '([a-z]+)(\d+)', '$1-$2'`)
	if len(BlockingMessages(issues)) != 0 {
		t.Fatalf("expected no blocking issue, got %#v", issues)
	}
}

func TestLinePipelineReplaceFails(t *testing.T) {
	issues := LintText("bad.ps1", `Get-Content file.ps1 | ForEach-Object { $_ -replace 'source', 'target' } | Set-Content file.ps1`)
	if len(BlockingMessages(issues)) != 1 {
		t.Fatalf("expected one blocking issue, got %#v", issues)
	}
}

func TestRawBulkReplacePasses(t *testing.T) {
	issues := LintText("good.ps1", `$c = Get-Content -LiteralPath file.ps1 -Raw
$c = $c -creplace 'source', 'target'
Set-Content -LiteralPath file.ps1 -Value $c -NoNewline`)
	if len(BlockingMessages(issues)) != 0 {
		t.Fatalf("expected no blocking issue, got %#v", issues)
	}
}

func TestDynamicCallbackWarnsWithoutRequires(t *testing.T) {
	issues := LintText("callback.ps1", `$x = $source -creplace '(?:^|_)(\w)', { $_.Groups[1].Value.ToUpperInvariant() }`)
	if len(issues) != 1 || issues[0].Severity != "warning" {
		t.Fatalf("expected one warning, got %#v", issues)
	}
	if len(BlockingMessages(issues)) != 0 {
		t.Fatalf("warning must not block by default, got %#v", issues)
	}
}

func TestLintPathSkipsBadFixtureDirectories(t *testing.T) {
	repo := t.TempDir()
	badDir := filepath.Join(repo, "tests", "cases", "bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badFile := filepath.Join(badDir, "Regex-DoubleQuotedCaptureReplacement.ps1")
	if err := os.WriteFile(badFile, []byte(`$x = $text -creplace '([a-z]+)', "$1"`), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := LintPath(repo, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(BlockingMessages(issues)) != 0 {
		t.Fatalf("bad fixtures should be skipped during directory lint, got %#v", issues)
	}
}

func TestBadFixturePathDetection(t *testing.T) {
	if !isBadFixturePath("dist/kit/tests/cases/bad/Regex-LinePipelineReplace.ps1") {
		t.Fatal("expected bad fixture path to be skipped")
	}
	if isBadFixturePath("dist/kit/tests/cases/good/Regex-SafeBulkReplace.ps1") {
		t.Fatal("good fixture path must not be skipped")
	}
	if isBadFixturePath("tools/specialty/BadButRealScript.ps1") {
		t.Fatal("ordinary scripts must not be skipped")
	}
}
