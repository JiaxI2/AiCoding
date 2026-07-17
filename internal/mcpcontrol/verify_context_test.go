package mcpcontrol

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestRunPythonStepHonorsCanceledContext(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := runPythonStep(ctx, t.TempDir(), executable, []string{"-test.run=TestDoesNotRun"})
	if result.OK || len(result.Errors) == 0 {
		t.Fatalf("canceled command unexpectedly passed: %#v", result)
	}
	if !strings.Contains(strings.ToLower(strings.Join(result.Errors, " ")), "canceled") {
		t.Fatalf("canceled command did not report context cancellation: %#v", result)
	}
}
