package kit

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExportCommandRejectsMissingInclude(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	entry := RegistryKit{ID: "bad-export"}
	manifest := Manifest{ID: entry.ID, Version: "0.1.0"}
	command := CommandDef{Include: []string{"does-not-exist/**"}, OutputName: "${kitId}-${version}.zip"}
	if err := ValidateExportCommand(repo, entry, manifest, command); err == nil || !strings.Contains(err.Error(), "does-not-exist/**") {
		t.Fatalf("missing export include was not reported: %v", err)
	}
}
