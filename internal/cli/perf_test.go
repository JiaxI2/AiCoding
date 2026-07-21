package cli

import "testing"

func TestLatencyBudgetClassifiesInjectedSlowCommand(t *testing.T) {
	descriptor := CommandDescriptor{Name: "injected-slow", LatencyClass: LatencyFast}
	warn := classifyLatency(descriptor, []float64{599, 601, 603})
	if warn.Status != "warn" || !warn.OK {
		t.Fatalf("650ms-class fast probe did not warn: %#v", warn)
	}
	fail := classifyLatency(descriptor, []float64{1201, 1202, 1203})
	if fail.Status != "fail" || fail.OK {
		t.Fatalf("3x fast probe did not fail: %#v", fail)
	}
	t.Logf("injected fast command median=%.0fms budget=%dms status=%s", warn.MedianMS, warn.BudgetMS, warn.Status)
	t.Logf("injected fast command median=%.0fms budget=%dms status=%s", fail.MedianMS, fail.BudgetMS, fail.Status)
}
