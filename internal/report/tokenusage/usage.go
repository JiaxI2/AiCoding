package tokenusage

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// Usage is the normalized token-usage child report embedded into AiCoding reports.
// CachedInputTokens and CacheWriteInputTokens are subsets of InputTokens and are
// not added again to TotalTokens.
type Usage struct {
	Source                string    `json:"source,omitempty"`
	ThreadID              string    `json:"thread_id,omitempty"`
	TurnID                string    `json:"turn_id,omitempty"`
	InputTokens           int64     `json:"input_tokens"`
	CachedInputTokens     int64     `json:"cached_input_tokens"`
	CacheWriteInputTokens int64     `json:"cache_write_input_tokens"`
	OutputTokens          int64     `json:"output_tokens"`
	ReasoningOutputTokens int64     `json:"reasoning_output_tokens"`
	TotalTokens           int64     `json:"total_tokens"`
	ContextTokens         int64     `json:"context_tokens,omitempty"`
	ContextWindow         int64     `json:"context_window,omitempty"`
	ContextRemaining      int64     `json:"context_remaining,omitempty"`
	ContextUsedPercent    float64   `json:"context_used_percent,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (u Usage) Empty() bool {
	return u.InputTokens == 0 && u.CachedInputTokens == 0 && u.CacheWriteInputTokens == 0 &&
		u.OutputTokens == 0 && u.ReasoningOutputTokens == 0 && u.TotalTokens == 0
}

func (u *Usage) Normalize() {
	if u.TotalTokens == 0 {
		u.TotalTokens = u.InputTokens + u.OutputTokens
	}
	if u.ContextTokens == 0 && u.ContextWindow > 0 {
		u.ContextTokens = u.TotalTokens
	}
	if u.ContextWindow > 0 {
		used := min(max(u.ContextTokens, 0), u.ContextWindow)
		u.ContextRemaining = u.ContextWindow - used
		u.ContextUsedPercent = float64(used) * 100 / float64(u.ContextWindow)
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = time.Now().UTC()
	}
}

// Collector consumes both Codex app-server JSON-RPC notifications and
// `codex exec --json` JSONL events. Updates replace the latest cumulative
// snapshot, so repeated events are idempotent and do not double count.
type Collector struct {
	mu     sync.RWMutex
	latest Usage
	seen   int64
}

func NewCollector() *Collector { return &Collector{} }

func (c *Collector) ConsumeJSONLine(line []byte) (bool, error) {
	var value any
	if err := json.Unmarshal(line, &value); err != nil {
		return false, fmt.Errorf("decode codex event: %w", err)
	}
	return c.consume(value), nil
}

func (c *Collector) consume(value any) bool {
	usage, ok := Extract(value)
	if !ok {
		return false
	}
	c.mu.Lock()
	c.latest = usage
	c.seen++
	c.mu.Unlock()
	return true
}

func (c *Collector) Snapshot() Usage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latest
}

func (c *Collector) SeenUpdates() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.seen
}

// ParseJSONL is a small standalone API for files, pipes and tests.
func ParseJSONL(r io.Reader) (Usage, int64, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	collector := NewCollector()
	for {
		var value any
		if err := dec.Decode(&value); err != nil {
			if err == io.EOF {
				break
			}
			return Usage{}, collector.SeenUpdates(), fmt.Errorf("decode codex JSONL: %w", err)
		}
		collector.consume(value)
	}
	return collector.Snapshot(), collector.SeenUpdates(), nil
}
