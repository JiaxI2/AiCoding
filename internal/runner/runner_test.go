package runner

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRunPreservesTaskOrder(t *testing.T) {
	tasks := []Task{
		{ID: "slow", Run: func(context.Context) TaskResult {
			time.Sleep(20 * time.Millisecond)
			return TaskResult{OK: true, Data: "slow"}
		}},
		{ID: "fast", Run: func(context.Context) TaskResult {
			return TaskResult{OK: true, Data: "fast"}
		}},
	}

	results := Run(context.Background(), tasks, Options{MaxParallel: 2})
	got := []string{results[0].ID, results[1].ID}
	if !reflect.DeepEqual(got, []string{"slow", "fast"}) {
		t.Fatalf("result order = %v", got)
	}
}

func TestRunReportsTimeout(t *testing.T) {
	tasks := []Task{
		{ID: "timeout", Timeout: time.Millisecond, Run: func(ctx context.Context) TaskResult {
			<-ctx.Done()
			return TaskResult{OK: true}
		}},
	}

	results := Run(context.Background(), tasks, Options{MaxParallel: 1})
	if len(results) != 1 || results[0].OK {
		t.Fatalf("expected timeout failure, got %#v", results)
	}
}

func TestExecutionPlanSelectionDoesNotMutateSource(t *testing.T) {
	parameters := map[string]string{"scope": "staged"}
	p, err := NewExecutionPlan(
		Task{ID: "a", Action: "checks.a", Parameters: parameters, Run: okTask},
		Task{ID: "b", Action: "checks.b", Group: "replacement", Run: okTask},
		Task{ID: "c", Action: "checks.c", Run: okTask},
	)
	if err != nil {
		t.Fatalf("NewExecutionPlan: %v", err)
	}
	parameters["scope"] = "all"

	only := p.Without("a").Only("c", "missing")
	tasks := only.Tasks()
	if len(tasks) != 1 || tasks[0].ID != "c" {
		t.Fatalf("Only returned %#v", tasks)
	}
	if got := p.Tasks(); len(got) != 3 || got[0].Parameters["scope"] != "staged" {
		t.Fatalf("source plan was mutated: %#v", got)
	}

	results := p.Run(context.Background(), Options{MaxParallel: 2})
	got := []string{results[0].ID, results[1].ID, results[2].ID}
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("plan order = %v", got)
	}
	if results[1].Group != "replacement" {
		t.Fatalf("task group was lost: %#v", results[1])
	}
}

func TestExecutionPlanSnapshotAndDigestAreDeterministic(t *testing.T) {
	first, err := NewExecutionPlan(Task{
		ID: "lint", Action: "governance.lint", Group: "hook",
		Parameters: map[string]string{"scope": "staged", "mode": "strict"},
		Timeout:    time.Second, Critical: true, Run: okTask,
	})
	if err != nil {
		t.Fatalf("NewExecutionPlan: %v", err)
	}
	second, err := NewExecutionPlan(Task{
		ID: "lint", Action: "governance.lint", Group: "hook",
		Parameters: map[string]string{"mode": "strict", "scope": "staged"},
		Timeout:    time.Second, Critical: true, Run: func(context.Context) TaskResult { return TaskResult{OK: false} },
	})
	if err != nil {
		t.Fatalf("NewExecutionPlan: %v", err)
	}

	firstDigest, err := first.Digest()
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	secondDigest, err := second.Digest()
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}
	if firstDigest != secondDigest || !strings.HasPrefix(firstDigest, "sha256:") {
		t.Fatalf("digests differ: %q != %q", firstDigest, secondDigest)
	}

	data, err := json.Marshal(first.Snapshot())
	if err != nil {
		t.Fatalf("Marshal snapshot: %v", err)
	}
	if strings.Contains(string(data), "Run") || !strings.Contains(string(data), `"action":"governance.lint"`) {
		t.Fatalf("snapshot is not a stable descriptor: %s", data)
	}

	changed, _ := NewExecutionPlan(Task{ID: "lint", Action: "governance.lint", Parameters: map[string]string{"scope": "all"}, Run: okTask})
	changedDigest, _ := changed.Digest()
	if firstDigest == changedDigest {
		t.Fatal("semantic plan change did not change digest")
	}
}

func TestExecutionPlanRejectsInvalidTasks(t *testing.T) {
	for _, tasks := range [][]Task{
		{{ID: "missing-action", Run: okTask}},
		{{ID: "duplicate", Action: "checks.a", Run: okTask}, {ID: "duplicate", Action: "checks.b", Run: okTask}},
	} {
		if _, err := NewExecutionPlan(tasks...); err == nil {
			t.Fatalf("invalid tasks accepted: %#v", tasks)
		}
	}
}

func TestExecutionPlanCanDescribeUnboundWork(t *testing.T) {
	plan, err := NewExecutionPlan(Task{ID: "describe", Action: "checks.describe"})
	if err != nil {
		t.Fatalf("descriptor-only plan was rejected: %v", err)
	}
	if _, err := plan.Digest(); err != nil {
		t.Fatalf("descriptor-only plan has no digest: %v", err)
	}
	results := plan.Run(context.Background(), Options{MaxParallel: 1})
	if len(results) != 1 || results[0].OK || !strings.Contains(strings.Join(results[0].Errors, " "), "run function is nil") {
		t.Fatalf("unbound execution did not fail explicitly: %#v", results)
	}
}

func okTask(context.Context) TaskResult {
	return TaskResult{OK: true}
}
