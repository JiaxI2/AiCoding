package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/report/tokenusage"
)

func TestRunCodexUsageParseReturnsStandardReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "codex.jsonl")
	mustWrite(t, path, `{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":60,"output_tokens":25,"reasoning_output_tokens":5}}`+"\n")

	res, err := runCodexUsage([]string{"usage", "parse", "--file", path, "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "codex usage parse" {
		t.Fatalf("codex usage parse failed: res=%#v err=%v", res, err)
	}
	standard, ok := res.Data.(report.StandardReport)
	if !ok || standard.Status != "PASS" || standard.Profile != "codex" {
		t.Fatalf("expected standard Codex report, got %#v", res.Data)
	}
	details, ok := standard.Details.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected details: %#v", standard.Details)
	}
	usage, ok := details["token_usage"].(tokenusage.Usage)
	if !ok || usage.TotalTokens != 125 {
		t.Fatalf("unexpected token usage: %#v", details["token_usage"])
	}
}

func TestRunCodexUsageParseRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.jsonl")
	mustWrite(t, path, "not-json\n")
	res, err := runCodexUsage([]string{"usage", "parse", "--file", path}, time.Now())
	if err == nil || res.OK {
		t.Fatalf("invalid JSON must fail: res=%#v err=%v", res, err)
	}
}

func TestRunCodexUsageRejectsUnsupportedRoute(t *testing.T) {
	for _, args := range [][]string{{}, {"status"}, {"usage"}, {"usage", "unknown"}} {
		res, err := runCodexUsage(args, time.Now())
		if err == nil || res.OK {
			t.Fatalf("unsupported args must fail: args=%#v res=%#v err=%v", args, res, err)
		}
	}
}

func TestWriteCodexUsageText(t *testing.T) {
	res := tokenUsageResult("codex usage parse", tokenusage.Usage{InputTokens: 10, OutputTokens: 2, TotalTokens: 12}, 1, time.Now())
	var out bytes.Buffer
	writeCodexUsageText(&out, res)
	if !strings.Contains(out.String(), "total:       12") {
		t.Fatalf("unexpected text output: %q", out.String())
	}
}

func TestJSONRequestedStopsAtChildCommandSeparator(t *testing.T) {
	if jsonRequested([]string{"usage", "run", "--", "codex", "exec", "--json", "prompt"}) {
		t.Fatal("child --json must not select JSON output for the AiCoding report")
	}
	if !jsonRequested([]string{"usage", "run", "--json", "--", "codex", "exec", "--json", "prompt"}) {
		t.Fatal("parent --json must select JSON output")
	}
}
