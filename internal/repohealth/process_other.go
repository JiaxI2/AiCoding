//go:build !windows

package repohealth

func systemProcessSnapshot() ([]processSnapshot, bool, error) {
	return []processSnapshot{}, false, nil
}
