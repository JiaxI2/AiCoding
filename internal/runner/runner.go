package runner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Task struct {
	ID       string
	Group    string
	Timeout  time.Duration
	Critical bool
	Run      func(context.Context) TaskResult
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

type Plan struct {
	tasks []Task
}

func NewPlan(tasks ...Task) Plan {
	p := Plan{}
	for _, task := range tasks {
		p.Add(task)
	}
	return p
}

func (p *Plan) Add(task Task) {
	p.Remove(task.ID)
	p.tasks = append(p.tasks, task)
}

func (p *Plan) Remove(ids ...string) {
	if len(ids) == 0 {
		return
	}
	remove := map[string]bool{}
	for _, id := range ids {
		remove[id] = true
	}
	kept := p.tasks[:0]
	for _, task := range p.tasks {
		if !remove[task.ID] {
			kept = append(kept, task)
		}
	}
	p.tasks = kept
}

func (p Plan) Only(ids ...string) Plan {
	if len(ids) == 0 {
		return p
	}
	keep := map[string]bool{}
	for _, id := range ids {
		keep[id] = true
	}
	out := Plan{}
	for _, task := range p.tasks {
		if keep[task.ID] {
			out.tasks = append(out.tasks, task)
		}
	}
	return out
}

func (p Plan) Tasks() []Task {
	out := make([]Task, len(p.tasks))
	copy(out, p.tasks)
	return out
}

func (p Plan) Run(ctx context.Context, opts Options) []TaskResult {
	return Run(ctx, p.tasks, opts)
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
