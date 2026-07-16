package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/report/tokenusage"
)

// runCodexUsage is intentionally thin: token parsing and normalization live in
// report/tokenusage so Plan reports and other commands can reuse the same API.
func runCodexUsage(args []string, start time.Time) (report.Result, error) {
	if len(args) == 0 || args[0] != "usage" {
		return report.Result{}, usageErrorf("codex 需要 usage 子命令")
	}
	if len(args) == 1 {
		return report.Result{}, usageErrorf("codex usage 需要 parse 或 run")
	}
	switch args[1] {
	case "parse":
		return runCodexUsageParse(args[2:], start)
	case "run":
		return runCodexUsageCommand(args[2:], start)
	default:
		return report.Result{}, usageErrorf("不支持的 codex usage 子命令：%s", args[1])
	}
}

func runCodexUsageParse(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("codex usage parse")
	file := fs.String("file", "-", "Codex JSONL file, or - for stdin")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}

	var r io.Reader = os.Stdin
	var closeFn func() error
	if *file != "-" {
		f, err := os.Open(*file)
		if err != nil {
			return report.Result{}, err
		}
		r, closeFn = f, f.Close
	}
	if closeFn != nil {
		defer closeFn()
	}
	usage, updates, err := tokenusage.ParseJSONL(r)
	if err != nil {
		return report.Result{}, err
	}
	if usage.Empty() {
		return report.Result{}, fmt.Errorf("未找到 Codex Token 使用量事件")
	}
	return tokenUsageResult("codex usage parse", usage, updates, start), nil
}

func runCodexUsageCommand(args []string, start time.Time) (report.Result, error) {
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || sep == len(args)-1 {
		return report.Result{}, usageErrorf("用法：aicoding codex usage run -- codex exec --json PROMPT")
	}
	fs := newFlagSet("codex usage run")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[:sep]); err != nil {
		return report.Result{}, err
	}
	command := args[sep+1:]
	cmd := exec.Command(command[0], command[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return report.Result{}, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return report.Result{}, err
	}

	collector := tokenusage.NewCollector()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var parseErr error
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if _, err := collector.ConsumeJSONLine(line); err != nil && parseErr == nil {
			parseErr = err
		}
		// stdout is reserved for the final AiCoding report, so preserve the
		// child JSONL event stream on stderr for diagnostics.
		fmt.Fprintln(os.Stderr, string(line))
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if scanErr != nil {
		return report.Result{}, scanErr
	}
	if waitErr != nil {
		return report.Result{}, waitErr
	}
	if parseErr != nil {
		return report.Result{}, parseErr
	}
	usage := collector.Snapshot()
	if usage.Empty() {
		return report.Result{}, fmt.Errorf("命令完成但未产生 Token 使用量事件：%s", strings.Join(command, " "))
	}
	return tokenUsageResult("codex usage run", usage, collector.SeenUpdates(), start), nil
}

func tokenUsageResult(command string, usage tokenusage.Usage, updates int64, start time.Time) report.Result {
	elapsed := report.Elapsed(start)
	details := map[string]interface{}{
		"token_usage":         usage,
		"token_usage_updates": updates,
	}
	data := standardReport(command, "codex", elapsed, map[string]interface{}{
		"input_tokens":             usage.InputTokens,
		"cached_input_tokens":      usage.CachedInputTokens,
		"cache_write_input_tokens": usage.CacheWriteInputTokens,
		"output_tokens":            usage.OutputTokens,
		"total_tokens":             usage.TotalTokens,
		"context_tokens":           usage.ContextTokens,
		"context_window":           usage.ContextWindow,
		"updates":                  updates,
	}, nil, nil, details)
	return report.Result{
		SchemaVersion: 1,
		Command:       command,
		OK:            true,
		Message:       "已采集 Codex Token 使用量",
		Data:          data,
		ElapsedMS:     elapsed,
	}
}

func writeCodexUsageText(w io.Writer, res report.Result) {
	data, ok := res.Data.(report.StandardReport)
	if !ok {
		return
	}
	details, ok := data.Details.(map[string]interface{})
	if !ok {
		return
	}
	usage, ok := details["token_usage"].(tokenusage.Usage)
	if !ok {
		return
	}
	tokenusage.WriteText(w, usage)
}
