package runner

import (
	"context"
	"reflect"
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

func TestPlanAddRemoveOnly(t *testing.T) {
	p := NewPlan(
		Task{ID: "a", Run: func(context.Context) TaskResult { return TaskResult{OK: true} }},
		Task{ID: "b", Run: func(context.Context) TaskResult { return TaskResult{OK: true} }},
	)
	p.Add(Task{ID: "b", Group: "replacement", Run: func(context.Context) TaskResult { return TaskResult{OK: true} }})
	p.Add(Task{ID: "c", Run: func(context.Context) TaskResult { return TaskResult{OK: true} }})
	p.Remove("a")

	only := p.Only("c", "missing")
	tasks := only.Tasks()
	if len(tasks) != 1 || tasks[0].ID != "c" {
		t.Fatalf("Only returned %#v", tasks)
	}

	results := p.Run(context.Background(), Options{MaxParallel: 2})
	got := []string{results[0].ID, results[1].ID}
	if !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("plan order = %v", got)
	}
	if results[0].Group != "replacement" {
		t.Fatalf("Add did not replace same id task: %#v", results[0])
	}
}
