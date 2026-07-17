package asset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type BasicAdapter struct{ AssetType Type }
func (a BasicAdapter) Type() Type { return a.AssetType }
func (a BasicAdapter) Validate(_ context.Context, m Manifest, root string) error {
	if m.Type != a.AssetType { return fmt.Errorf("adapter type mismatch: %s", m.Type) }
	if st, err := os.Stat(filepath.Join(root, m.Paths.Payload)); err != nil || !st.IsDir() { return fmt.Errorf("payload directory not found: %s", m.Paths.Payload) }
	return nil
}
func (a BasicAdapter) AfterInstall(context.Context, Manifest, string, map[string]any) error { return nil }
func (a BasicAdapter) BeforeUninstall(context.Context, LockEntry, string) error { return nil }
func (a BasicAdapter) Verify(_ context.Context, _ Manifest, root string) error { _, err := os.Stat(root); return err }

func DefaultAdapters() []Adapter {
	return []Adapter{
		BasicAdapter{TypeKit}, BasicAdapter{TypeSkill}, BasicAdapter{TypeMCP},
		BasicAdapter{TypeTemplate}, BasicAdapter{TypeRuleset}, BasicAdapter{TypeProfile},
	}
}
