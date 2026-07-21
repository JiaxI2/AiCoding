//go:build windows

package repohealth

import (
	"fmt"
	"syscall"
	"unsafe"
)

const toolhelpSnapshotProcess = 0x00000002

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	createToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	process32FirstW          = kernel32.NewProc("Process32FirstW")
	process32NextW           = kernel32.NewProc("Process32NextW")
)

type processEntry32 struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [260]uint16
}

func systemProcessSnapshot() ([]processSnapshot, bool, error) {
	handle, _, callErr := createToolhelp32Snapshot.Call(toolhelpSnapshotProcess, 0)
	if handle == uintptr(syscall.InvalidHandle) {
		return nil, true, fmt.Errorf("create process snapshot: %w", callErr)
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ok, _, callErr := process32FirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return nil, true, fmt.Errorf("read first process: %w", callErr)
	}
	processes := []processSnapshot{}
	for {
		processes = append(processes, processSnapshot{
			PID: entry.ProcessID, ParentPID: entry.ParentProcessID,
			Name: syscall.UTF16ToString(entry.ExeFile[:]),
		})
		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		ok, _, _ = process32NextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ok == 0 {
			break
		}
	}
	return processes, true, nil
}
