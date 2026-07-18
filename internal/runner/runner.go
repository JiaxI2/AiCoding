package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID         string
	Action     string
	Group      string
	Parameters map[string]string
	Timeout    time.Duration
	Critical   bool
	Run        func(context.Context) TaskResult
}

type TaskDescriptor struct {
	ID         string            `json:"id"`
	Action     string            `json:"action"`
	Group      string            `json:"group,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Timeout    string            `json:"timeout,omitempty"`
	Critical   bool              `json:"critical,omitempty"`
}

type ExecutionPlanSnapshot struct {
	Object string           `json:"object"`
	Tasks  []TaskDescriptor `json:"tasks"`
}

type TaskResult struct {
	ID        string      `json:"id"`
	Group     string      `json:"group,omitempty"`
	OK        bool        `json:"ok"`
	Skipped   bool        `json:"skipped,omitempty"`
	Errors    []string    `json:"errors,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	ElapsedMS int64       `json:"elapsedMs"`
}

type Options struct {
	MaxParallel int
	FailFast    bool
}

type ExecutionPlan struct {
	tasks []Task
}

func NewExecutionPlan(tasks ...Task) (ExecutionPlan, error) {
	p := ExecutionPlan{tasks: cloneTasks(tasks)}
	if err := p.Validate(); err != nil {
		return ExecutionPlan{}, err
	}
	return p, nil
}

func (p ExecutionPlan) Without(ids ...string) ExecutionPlan {
	if len(ids) == 0 {
		return ExecutionPlan{tasks: cloneTasks(p.tasks)}
	}
	remove := map[string]bool{}
	for _, id := range ids {
		remove[id] = true
	}
	kept := make([]Task, 0, len(p.tasks))
	for _, task := range p.tasks {
		if !remove[task.ID] {
			kept = append(kept, cloneTask(task))
		}
	}
	return ExecutionPlan{tasks: kept}
}

func (p ExecutionPlan) Only(ids ...string) ExecutionPlan {
	if len(ids) == 0 {
		return ExecutionPlan{tasks: cloneTasks(p.tasks)}
	}
	keep := map[string]bool{}
	for _, id := range ids {
		keep[id] = true
	}
	out := ExecutionPlan{}
	for _, task := range p.tasks {
		if keep[task.ID] {
			out.tasks = append(out.tasks, cloneTask(task))
		}
	}
	return out
}

func (p ExecutionPlan) Tasks() []Task {
	return cloneTasks(p.tasks)
}

func (p ExecutionPlan) Validate() error {
	seen := make(map[string]struct{}, len(p.tasks))
	for index, task := range p.tasks {
		if strings.TrimSpace(task.ID) == "" {
			return fmt.Errorf("task %d id is required", index)
		}
		if strings.TrimSpace(task.Action) == "" {
			return fmt.Errorf("task %q action is required", task.ID)
		}
		if task.Timeout < 0 {
			return fmt.Errorf("task %q timeout must not be negative", task.ID)
		}
		if _, exists := seen[task.ID]; exists {
			return fmt.Errorf("duplicate task id: %s", task.ID)
		}
		seen[task.ID] = struct{}{}
	}
	return nil
}

func (p ExecutionPlan) Snapshot() ExecutionPlanSnapshot {
	tasks := make([]TaskDescriptor, 0, len(p.tasks))
	for _, task := range p.tasks {
		descriptor := TaskDescriptor{
			ID:         task.ID,
			Action:     task.Action,
			Group:      task.Group,
			Parameters: cloneParameters(task.Parameters),
			Critical:   task.Critical,
		}
		if task.Timeout > 0 {
			descriptor.Timeout = task.Timeout.String()
		}
		tasks = append(tasks, descriptor)
	}
	return ExecutionPlanSnapshot{Object: "execution-plan", Tasks: tasks}
}

func (p ExecutionPlan) Digest() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(p.Snapshot())
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum), nil
}

func (p ExecutionPlan) Run(ctx context.Context, opts Options) []TaskResult {
	return Run(ctx, p.tasks, opts)
}

func cloneTasks(tasks []Task) []Task {
	out := make([]Task, len(tasks))
	for index, task := range tasks {
		out[index] = cloneTask(task)
	}
	return out
}

func cloneTask(task Task) Task {
	task.Parameters = cloneParameters(task.Parameters)
	return task
}

func cloneParameters(parameters map[string]string) map[string]string {
	if len(parameters) == 0 {
		return nil
	}
	out := make(map[string]string, len(parameters))
	for key, value := range parameters {
		out[key] = value
	}
	return out
}

func Run(ctx context.Context, tasks []Task, opts Options) []TaskResult {
	maxParallel := opts.MaxParallel
	if maxParallel <= 0 {
		maxParallel = runtime.NumCPU()
		if maxParallel > 8 {
			maxParallel = 8
		}
		if maxParallel < 1 {
			maxParallel = 1
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]TaskResult, len(tasks))
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	var cancelOnce sync.Once

	for i, task := range tasks {
		i := i
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = skippedResult(task, ctx.Err())
				return
			}

			res := runOne(ctx, task)
			results[i] = res
			if opts.FailFast && task.Critical && !res.OK {
				cancelOnce.Do(cancel)
			}
		}()
	}
	wg.Wait()
	return results
}

func runOne(parent context.Context, task Task) (res TaskResult) {
	start := time.Now()
	ctx := parent
	cancel := func() {}
	if task.Timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, task.Timeout)
	}
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			res = TaskResult{ID: task.ID, Group: task.Group, OK: false, Errors: []string{fmt.Sprintf("panic: %v", r)}}
		}
		if res.ID == "" {
			res.ID = task.ID
		}
		if res.Group == "" {
			res.Group = task.Group
		}
		if res.ElapsedMS == 0 {
			res.ElapsedMS = time.Since(start).Milliseconds()
		}
	}()

	if task.Run == nil {
		return TaskResult{ID: task.ID, Group: task.Group, OK: false, Errors: []string{"task run function is nil"}}
	}
	if err := ctx.Err(); err != nil {
		return skippedResult(task, err)
	}
	res = task.Run(ctx)
	if err := ctx.Err(); err != nil && res.OK {
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
	}
	return res
}

func skippedResult(task Task, err error) TaskResult {
	msg := "task skipped"
	if err != nil {
		msg = err.Error()
	}
	return TaskResult{ID: task.ID, Group: task.Group, OK: true, Skipped: true, Warnings: []string{msg}}
}
