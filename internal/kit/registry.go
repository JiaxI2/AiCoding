package kit

import (
	"encoding/json"
	"errors"
	"os"
	"sort"

	"github.com/JiaxI2/AiCoding/internal/platform"
)

func LoadRegistry(repo string) ([]RegistryKit, error) {
	p := platform.RepoPath(repo, "config/kit-registry.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var reg registry
	if err := json.Unmarshal(b, &reg); err != nil {
		return nil, err
	}
	sort.SliceStable(reg.Kits, func(i, j int) bool { return reg.Kits[i].Order < reg.Kits[j].Order })
	return reg.Kits, nil
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
