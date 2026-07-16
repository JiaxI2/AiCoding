package tokenusage

import (
	"fmt"
	"io"
)

func WriteText(w io.Writer, u Usage) {
	fmt.Fprintln(w, "Codex token usage")
	fmt.Fprintf(w, "  input:       %d\n", u.InputTokens)
	fmt.Fprintf(w, "  cached:      %d\n", u.CachedInputTokens)
	fmt.Fprintf(w, "  cache write: %d\n", u.CacheWriteInputTokens)
	fmt.Fprintf(w, "  output:      %d\n", u.OutputTokens)
	fmt.Fprintf(w, "  reasoning:   %d\n", u.ReasoningOutputTokens)
	fmt.Fprintf(w, "  total:       %d\n", u.TotalTokens)
	if u.ContextWindow > 0 {
		fmt.Fprintf(w, "  context:     %.2f%% (%d/%d)\n", u.ContextUsedPercent, u.ContextTokens, u.ContextWindow)
	}
}
