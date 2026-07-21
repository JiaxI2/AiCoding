package repohealth

import (
	"os"
	"strings"
	"testing"
)

func TestProductVerifyDoesNotOwnTestOrReleaseExecution(t *testing.T) {
	source, err := os.ReadFile("product.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{
		"internal/testengine",
		"mcpcontrol.Verify(",
		"releasegate",
		"visio_smoke.py",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("product verification crossed into test/release execution: %s", forbidden)
		}
	}
	for _, required := range []string{
		"VerifyHooks(",
		"VerifyRepoText(",
		"CheckDependencies(",
		"DoctorRegistry(",
		"ScopeRuntimeSkill",
		"cache.Status(",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("product verification is missing static boundary %q", required)
		}
	}
}
