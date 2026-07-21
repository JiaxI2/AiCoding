package testengine

import "github.com/JiaxI2/AiCoding/internal/cache"

func retainSuccessfulTestResults(repo string, report Report) error {
	if ExitCode(report, nil) != 0 {
		return nil
	}
	_, err := cache.Clean(repo, cache.CleanOptions{
		Scope: cache.ScopeTestResults,
		Keep:  cache.DefaultTestResultKeep,
	})
	return err
}
