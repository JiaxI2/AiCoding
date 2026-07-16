package tokenusage

import (
	"strings"
	"testing"
)

func TestExtractExecCompleted(t *testing.T) {
	line := `{"type":"turn.completed","usage":{"input_tokens":24763,"cached_input_tokens":24448,"output_tokens":122,"reasoning_output_tokens":7}}`
	c := NewCollector()
	updated, err := c.ConsumeJSONLine([]byte(line))
	if err != nil || !updated {
		t.Fatalf("updated=%v err=%v", updated, err)
	}
	got := c.Snapshot()
	if got.TotalTokens != 24885 {
		t.Fatalf("total=%d", got.TotalTokens)
	}
	if got.CachedInputTokens != 24448 {
		t.Fatalf("cached=%d", got.CachedInputTokens)
	}
	if got.ReasoningOutputTokens != 7 {
		t.Fatalf("reasoning=%d", got.ReasoningOutputTokens)
	}
}

func TestExtractAppServerNotificationUsesCumulativeTotalAndLastContext(t *testing.T) {
	line := `{"method":"thread/tokenUsage/updated","params":{"threadId":"thr-1","turnId":"turn-1","tokenUsage":{"total":{"inputTokens":180,"cachedInputTokens":120,"cacheWriteInputTokens":3,"outputTokens":40,"reasoningOutputTokens":8,"totalTokens":220},"last":{"inputTokens":80,"cachedInputTokens":40,"cacheWriteInputTokens":1,"outputTokens":20,"reasoningOutputTokens":4,"totalTokens":100},"modelContextWindow":200}}}`
	c := NewCollector()
	updated, err := c.ConsumeJSONLine([]byte(line))
	if err != nil || !updated {
		t.Fatalf("updated=%v err=%v", updated, err)
	}
	got := c.Snapshot()
	if got.ThreadID != "thr-1" || got.TurnID != "turn-1" {
		t.Fatalf("ids=%q/%q", got.ThreadID, got.TurnID)
	}
	if got.TotalTokens != 220 || got.InputTokens != 180 {
		t.Fatalf("expected cumulative total usage, got %#v", got)
	}
	if got.CacheWriteInputTokens != 3 {
		t.Fatalf("cache write=%d", got.CacheWriteInputTokens)
	}
	if got.ContextTokens != 100 || got.ContextRemaining != 100 || got.ContextUsedPercent != 50 {
		t.Fatalf("unexpected context metrics: %#v", got)
	}
}

func TestLatestSnapshotDoesNotDoubleCount(t *testing.T) {
	stream := strings.NewReader(
		`{"method":"thread/tokenUsage/updated","params":{"usage":{"inputTokens":10,"outputTokens":2}}}` + "\n" +
			`{"method":"thread/tokenUsage/updated","params":{"usage":{"inputTokens":20,"outputTokens":4}}}` + "\n")
	got, updates, err := ParseJSONL(stream)
	if err != nil {
		t.Fatal(err)
	}
	if updates != 2 {
		t.Fatalf("updates=%d", updates)
	}
	if got.TotalTokens != 24 {
		t.Fatalf("want latest cumulative 24, got %d", got.TotalTokens)
	}
}

func TestContextUsageIsClamped(t *testing.T) {
	u := Usage{TotalTokens: 300, ContextTokens: 300, ContextWindow: 200}
	u.Normalize()
	if u.ContextRemaining != 0 || u.ContextUsedPercent != 100 {
		t.Fatalf("unexpected clamped context metrics: %#v", u)
	}
}

func TestIgnoreUnrelatedEvent(t *testing.T) {
	c := NewCollector()
	updated, err := c.ConsumeJSONLine([]byte(`{"type":"item.completed","item":{"type":"agent_message"}}`))
	if err != nil || updated {
		t.Fatalf("updated=%v err=%v", updated, err)
	}
}
