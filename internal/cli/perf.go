package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/report"
)

const latencyProbeRuns = 3

var runLatencyDoctor = func(_ string, start time.Time) (report.Result, error) {
	err := fmt.Errorf("command latency catalog is not initialized")
	return report.Fail("doctor perf", start, err.Error(), nil, err.Error()), err
}

func init() {
	runLatencyDoctor = runCatalogLatencyDoctor
}

type latencyProbeResult struct {
	Command     string       `json:"command"`
	Probe       []string     `json:"probe,omitempty"`
	Class       LatencyClass `json:"class"`
	BudgetMS    int64        `json:"budgetMs"`
	SamplesMS   []float64    `json:"samplesMs"`
	MedianMS    float64      `json:"medianMs"`
	BudgetRatio float64      `json:"budgetRatio"`
	Status      string       `json:"status"`
	OK          bool         `json:"ok"`
	Errors      []string     `json:"errors,omitempty"`
}

func runCatalogLatencyDoctor(repo string, start time.Time) (report.Result, error) {
	checks := []latencyProbeResult{}
	warnings := []string{}
	errorsFound := []string{}
	for _, descriptor := range Catalog().Commands {
		if descriptor.LatencyClass == LatencyWork {
			continue
		}
		check := measureCatalogLatency(repo, descriptor)
		checks = append(checks, check)
		switch check.Status {
		case "warn":
			warnings = append(warnings, fmt.Sprintf("%s median %.3fms exceeds 1.5x %s budget (%dms)", check.Command, check.MedianMS, check.Class, check.BudgetMS))
		case "fail":
			errorsFound = append(errorsFound, fmt.Sprintf("%s median %.3fms exceeds 3x %s budget (%dms)", check.Command, check.MedianMS, check.Class, check.BudgetMS))
		}
		for _, issue := range check.Errors {
			errorsFound = append(errorsFound, check.Command+": "+issue)
		}
	}
	result := report.Result{SchemaVersion: 1, Command: "doctor perf", OK: len(errorsFound) == 0, Message: "typed command latency budget", RepoRoot: repo, Data: checks, Warnings: warnings, Errors: errorsFound, ElapsedMS: report.Elapsed(start)}
	return result, report.BoolErr(errorsFound)
}

func measureCatalogLatency(repo string, descriptor CommandDescriptor) latencyProbeResult {
	samples := make([]float64, 0, latencyProbeRuns)
	errorsFound := []string{}
	for range latencyProbeRuns {
		started := time.Now()
		err := runCatalogLatencyProbe(repo, descriptor)
		samples = append(samples, milliseconds(time.Since(started)))
		if err != nil {
			errorsFound = appendUnique(errorsFound, err.Error())
		}
	}
	result := classifyLatency(descriptor, samples)
	result.Errors = errorsFound
	if len(errorsFound) > 0 {
		result.Status = "fail"
		result.OK = false
	}
	return result
}

func runCatalogLatencyProbe(repo string, descriptor CommandDescriptor) error {
	route, ok := commands.lookup(descriptor.Name)
	if !ok {
		return fmt.Errorf("command route is unavailable: %s", descriptor.Name)
	}
	switch route.direct {
	case directHelp:
		_ = renderCatalogHelp()
		return nil
	case directVersion:
		_ = productVersion()
		return nil
	}
	if route.handler == nil {
		return fmt.Errorf("command route has no handler: %s", descriptor.Name)
	}
	args := append([]string(nil), descriptor.LatencyProbe...)
	args = append(args, "--repo-root", repo)
	result, err := route.handler(args, time.Now())
	if err != nil {
		return err
	}
	if !result.OK {
		if len(result.Errors) > 0 {
			return fmt.Errorf("%s", strings.Join(result.Errors, "; "))
		}
		return fmt.Errorf("probe returned not ok")
	}
	return nil
}

func classifyLatency(descriptor CommandDescriptor, samples []float64) latencyProbeResult {
	ordered := append([]float64(nil), samples...)
	sort.Float64s(ordered)
	median := 0.0
	if len(ordered) > 0 {
		median = ordered[len(ordered)/2]
	}
	budget := descriptor.LatencyClass.BudgetMS()
	ratio := 0.0
	if budget > 0 {
		ratio = median / float64(budget)
	}
	status := "pass"
	if ratio > 3 {
		status = "fail"
	} else if ratio > 1.5 {
		status = "warn"
	}
	return latencyProbeResult{
		Command:     descriptor.Name,
		Probe:       append([]string(nil), descriptor.LatencyProbe...),
		Class:       descriptor.LatencyClass,
		BudgetMS:    budget,
		SamplesMS:   append([]float64(nil), samples...),
		MedianMS:    median,
		BudgetRatio: ratio,
		Status:      status,
		OK:          status != "fail",
	}
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}

func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}
