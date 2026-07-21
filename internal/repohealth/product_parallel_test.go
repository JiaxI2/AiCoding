package repohealth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/runner"
)

func TestProductChecksParallelMatchSerialJSONWithoutElapsed(t *testing.T) {
	tasks := []struct {
		id       string
		category string
		delay    time.Duration
	}{
		{id: "first", category: "REPOSITORY", delay: 4 * time.Millisecond},
		{id: "second", category: "GOVERNANCE", delay: time.Millisecond},
		{id: "third", category: "SKILL", delay: 2 * time.Millisecond},
	}
	build := func() []runner.Task {
		out := make([]runner.Task, 0, len(tasks))
		for _, spec := range tasks {
			spec := spec
			out = append(out, productCheck(spec.id, spec.category, func() (interface{}, []string, []string) {
				time.Sleep(spec.delay)
				return map[string]interface{}{"id": spec.id}, nil, nil
			}))
		}
		return out
	}

	serial := normalizeProductCheckDurations(executeProductChecks(context.Background(), build(), 1))
	parallel := normalizeProductCheckDurations(executeProductChecks(context.Background(), build(), 4))
	serialJSON, err := json.Marshal(serial)
	if err != nil {
		t.Fatal(err)
	}
	parallelJSON, err := json.Marshal(parallel)
	if err != nil {
		t.Fatal(err)
	}
	if string(serialJSON) != string(parallelJSON) {
		t.Fatalf("serial/parallel product JSON differs:\nserial=%s\nparallel=%s", serialJSON, parallelJSON)
	}
	t.Logf("serial=%s", serialJSON)
	t.Logf("parallel=%s", parallelJSON)
}

func normalizeProductCheckDurations(checks []report.Check) []report.Check {
	for index := range checks {
		checks[index].DurationMS = 0
	}
	return checks
}
