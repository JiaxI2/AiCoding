package kit

import (
	"encoding/json"
	"errors"
	"os"
	"sort"

	"github.com/JiaxI2/AiCoding/internal/platform"
	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

type RegistrySnapshot struct {
	object  registryobject.Snapshot
	entries []RegistryKit
}

func LoadRegistrySnapshot(repo string) (RegistrySnapshot, error) {
	p := platform.RepoPath(repo, "config/kit-registry.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return RegistrySnapshot{}, err
	}
	var reg registry
	if err := json.Unmarshal(b, &reg); err != nil {
		return RegistrySnapshot{}, err
	}
	sort.SliceStable(reg.Kits, func(i, j int) bool { return reg.Kits[i].Order < reg.Kits[j].Order })
	object, err := registryobject.NewSnapshot("kit-registry", reg)
	if err != nil {
		return RegistrySnapshot{}, err
	}
	return RegistrySnapshot{object: object, entries: cloneRegistryKits(reg.Kits)}, nil
}

func LoadRegistry(repo string) ([]RegistryKit, error) {
	snapshot, err := LoadRegistrySnapshot(repo)
	if err != nil {
		return nil, err
	}
	return snapshot.Entries(), nil
}

func (s RegistrySnapshot) Digest() string {
	return s.object.Digest()
}

func (s RegistrySnapshot) Object() registryobject.Snapshot {
	return s.object
}

func (s RegistrySnapshot) Entries() []RegistryKit {
	return cloneRegistryKits(s.entries)
}

func cloneRegistryKits(entries []RegistryKit) []RegistryKit {
	out := make([]RegistryKit, len(entries))
	copy(out, entries)
	return out
}

func LoadManifest(repo, rel string) (Manifest, error) {
	b, err := os.ReadFile(platform.RepoPath(repo, rel))
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func LoadKitViews(repo string, entries []RegistryKit) []View {
	views := []View{}
	for _, e := range entries {
		v := View{ID: e.ID, Enabled: e.Enabled, Order: e.Order, Manifest: e.Manifest}
		if m, err := LoadManifest(repo, e.Manifest); err == nil {
			v.Name = m.Name
			v.Version = m.Version
			v.Kind = m.Kind
			v.Mode = m.Mode
		}
		views = append(views, v)
	}
	return views
}

func SelectKits(entries []RegistryKit, kit string, all bool) ([]RegistryKit, error) {
	if all && kit != "" {
		return nil, errors.New("use either --all or --kit, not both")
	}
	if !all && kit == "" {
		return nil, errors.New("kit verify/test requires --all or --kit")
	}
	selected := []RegistryKit{}
	for _, e := range entries {
		if all && e.Enabled {
			selected = append(selected, e)
		}
		if kit != "" && e.ID == kit {
			selected = append(selected, e)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no kit matched")
	}
	return selected, nil
}
