package repohealth

import "testing"

func TestOrphanProcessInspectionReportsWithoutKilling(t *testing.T) {
	report := detectOrphanProcesses([]processSnapshot{
		{PID: 10, ParentPID: 1, Name: "parent.exe"},
		{PID: 11, ParentPID: 10, Name: "aicoding.exe"},
		{PID: 12, ParentPID: 99, Name: "pwsh.exe"},
		{PID: 13, ParentPID: 98, Name: "unrelated.exe"},
	}, true)
	if !report.Supported || report.CandidateCount != 2 || len(report.Orphans) != 1 {
		t.Fatalf("unexpected orphan report: %#v", report)
	}
	if report.Orphans[0].PID != 12 || report.KillAttempted {
		t.Fatalf("orphan inspection attempted mutation: %#v", report)
	}
	t.Logf("candidates=%d orphans=%d killAttempted=%t", report.CandidateCount, len(report.Orphans), report.KillAttempted)
}
