package tokenusage

import (
	"encoding/json"
	"sort"
)

var usageContainerKeys = []string{
	"usage",
	"tokenUsage",
	"token_usage",
	"total",
	"last",
	"lastTokenUsage",
	"last_token_usage",
}

// Extract accepts schema evolution by recursively locating a usage-shaped map.
// It supports official camelCase app-server fields and snake_case exec fields.
func Extract(value any) (Usage, bool) {
	root, ok := value.(map[string]any)
	if !ok {
		return Usage{}, false
	}

	candidate, contextTokens, contextWindow, ok := findUsageCandidate(root)
	if !ok {
		return Usage{}, false
	}
	u := Usage{
		Source:                sourceOf(root),
		ThreadID:              firstNonEmpty(deepStringAt(root, "threadId", "thread_id"), stringAt(candidate, "threadId", "thread_id")),
		TurnID:                firstNonEmpty(deepStringAt(root, "turnId", "turn_id"), stringAt(candidate, "turnId", "turn_id")),
		InputTokens:           intAt(candidate, "inputTokens", "input_tokens"),
		CachedInputTokens:     intAt(candidate, "cachedInputTokens", "cached_input_tokens"),
		CacheWriteInputTokens: intAt(candidate, "cacheWriteInputTokens", "cache_write_input_tokens"),
		OutputTokens:          intAt(candidate, "outputTokens", "output_tokens"),
		ReasoningOutputTokens: intAt(candidate, "reasoningOutputTokens", "reasoning_output_tokens", "reasoningTokens", "reasoning_tokens"),
		TotalTokens:           intAt(candidate, "totalTokens", "total_tokens"),
		ContextTokens:         contextTokens,
		ContextWindow:         contextWindow,
	}
	u.Normalize()
	return u, !u.Empty()
}

func findUsageCandidate(root map[string]any) (map[string]any, int64, int64, bool) {
	if params, ok := mapAt(root, "params"); ok {
		if tokenUsage, ok := mapAt(params, "tokenUsage", "token_usage"); ok {
			contextWindow := intAt(tokenUsage, "modelContextWindow", "model_context_window", "contextWindow", "context_window")
			last, _ := mapAt(tokenUsage, "last", "lastTokenUsage", "last_token_usage")
			contextTokens := intAt(last, "totalTokens", "total_tokens")
			if total, ok := mapAt(tokenUsage, "total"); ok && hasTokenFields(total) {
				return total, contextTokens, contextWindow, true
			}
			if hasTokenFields(tokenUsage) {
				return tokenUsage, contextTokens, contextWindow, true
			}
			if hasTokenFields(last) {
				return last, contextTokens, contextWindow, true
			}
		}
	}

	if usage, ok := mapAt(root, "usage"); ok && hasTokenFields(usage) {
		return usage, 0, deepIntAt(root, "modelContextWindow", "model_context_window", "contextWindow", "context_window"), true
	}
	candidate, ok := findUsageMap(root)
	return candidate, 0, deepIntAt(root, "modelContextWindow", "model_context_window", "contextWindow", "context_window"), ok
}

func findUsageMap(m map[string]any) (map[string]any, bool) {
	if hasTokenFields(m) {
		return m, true
	}
	for _, key := range usageContainerKeys {
		if child, ok := m[key].(map[string]any); ok {
			if found, ok := findUsageMap(child); ok {
				return found, true
			}
		}
	}
	for _, key := range sortedKeys(m) {
		value := m[key]
		switch child := value.(type) {
		case map[string]any:
			if found, ok := findUsageMap(child); ok {
				return found, true
			}
		case []any:
			for _, item := range child {
				if itemMap, ok := item.(map[string]any); ok {
					if found, ok := findUsageMap(itemMap); ok {
						return found, true
					}
				}
			}
		}
	}
	return nil, false
}

func hasTokenFields(m map[string]any) bool {
	if m == nil {
		return false
	}
	for _, key := range []string{"inputTokens", "input_tokens", "outputTokens", "output_tokens", "totalTokens", "total_tokens"} {
		if _, exists := m[key]; exists {
			return true
		}
	}
	return false
}

func sourceOf(m map[string]any) string {
	if method := stringAt(m, "method"); method != "" {
		return method
	}
	if typ := stringAt(m, "type"); typ != "" {
		return typ
	}
	return "codex"
}

func intAt(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		switch n := value.(type) {
		case float64:
			return int64(n)
		case json.Number:
			v, _ := n.Int64()
			return v
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return 0
}

func stringAt(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok {
			return value
		}
	}
	return ""
}

func mapAt(m map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		if value, ok := m[key].(map[string]any); ok {
			return value, true
		}
	}
	return nil, false
}

func deepStringAt(m map[string]any, keys ...string) string {
	if value := stringAt(m, keys...); value != "" {
		return value
	}
	for _, key := range sortedKeys(m) {
		if child, ok := m[key].(map[string]any); ok {
			if value := deepStringAt(child, keys...); value != "" {
				return value
			}
		}
	}
	return ""
}

func deepIntAt(m map[string]any, keys ...string) int64 {
	if value := intAt(m, keys...); value != 0 {
		return value
	}
	for _, key := range sortedKeys(m) {
		if child, ok := m[key].(map[string]any); ok {
			if value := deepIntAt(child, keys...); value != 0 {
				return value
			}
		}
	}
	return 0
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
