package repohealth

import (
	"path/filepath"
	"strings"
)

type processSnapshot struct {
	PID       uint32
	ParentPID uint32
	Name      string
}

type OrphanProcess struct {
	PID       uint32 `json:"pid"`
	ParentPID uint32 `json:"parentPid"`
	Name      string `json:"name"`
}

type OrphanProcessReport struct {
	Supported      bool            `json:"supported"`
	CandidateCount int             `json:"candidateCount"`
	Orphans        []OrphanProcess `json:"orphans"`
	KillAttempted  bool            `json:"killAttempted"`
}

func InspectOrphanProcesses() (OrphanProcessReport, error) {
	processes, supported, err := systemProcessSnapshot()
	if err != nil {
		return OrphanProcessReport{Supported: supported, Orphans: []OrphanProcess{}}, err
	}
	return detectOrphanProcesses(processes, supported), nil
}

func detectOrphanProcesses(processes []processSnapshot, supported bool) OrphanProcessReport {
	report := OrphanProcessReport{Supported: supported, Orphans: []OrphanProcess{}, KillAttempted: false}
	alive := make(map[uint32]bool, len(processes))
	for _, process := range processes {
		alive[process.PID] = true
	}
	for _, process := range processes {
		if !isOrphanCandidate(process.Name) {
			continue
		}
		report.CandidateCount++
		if process.ParentPID == 0 || process.ParentPID == process.PID || alive[process.ParentPID] {
			continue
		}
		report.Orphans = append(report.Orphans, OrphanProcess(process))
	}
	return report
}

func isOrphanCandidate(name string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	base = strings.TrimSuffix(base, ".exe")
	return base == "aicoding" || base == "pwsh" || base == "powershell"
}
