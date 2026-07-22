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

type ManifestSnapshot struct {
	entry  RegistryKit
	object registryobject.Snapshot
}

type CatalogSnapshot struct {
	object registryobject.CatalogSnapshot
	kits   []ManifestSnapshot
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

func LoadManifestSnapshot(repo string, entry RegistryKit) (ManifestSnapshot, error) {
	b, err := os.ReadFile(platform.RepoPath(repo, entry.Manifest))
	if err != nil {
		return ManifestSnapshot{}, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return ManifestSnapshot{}, err
	}
	object, err := registryobject.NewSnapshot("kit-manifest", m)
	if err != nil {
		return ManifestSnapshot{}, err
	}
	return ManifestSnapshot{entry: entry, object: object}, nil
}

func LoadManifest(repo, rel string) (Manifest, error) {
	snapshot, err := LoadManifestSnapshot(repo, RegistryKit{Manifest: rel})
	if err != nil {
		return Manifest{}, err
	}
	return snapshot.Manifest()
}

func LoadCatalogSnapshot(repo string) (CatalogSnapshot, error) {
	registry, err := LoadRegistrySnapshot(repo)
	if err != nil {
		return CatalogSnapshot{}, err
	}
	kits := make([]ManifestSnapshot, 0, len(registry.entries))
	entries := make([]registryobject.CatalogEntry, 0, len(registry.entries))
	for _, entry := range registry.entries {
		manifest, err := LoadManifestSnapshot(repo, entry)
		if err != nil {
			return CatalogSnapshot{}, err
		}
		kits = append(kits, manifest)
		entries = append(entries, registryobject.CatalogEntry{
			ID:     entry.ID,
			Path:   entry.Manifest,
			Digest: manifest.Digest(),
		})
	}
	object, err := registryobject.NewCatalogSnapshot("kit-catalog", registry.Object(), entries)
	if err != nil {
		return CatalogSnapshot{}, err
	}
	return CatalogSnapshot{object: object, kits: cloneManifestSnapshots(kits)}, nil
}

func (s ManifestSnapshot) Entry() RegistryKit {
	return s.entry
}

func (s ManifestSnapshot) Digest() string {
	return s.object.Digest()
}

func (s ManifestSnapshot) Manifest() (Manifest, error) {
	var manifest Manifest
	if err := s.object.Decode(&manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (s CatalogSnapshot) Digest() string {
	return s.object.Digest()
}

func (s CatalogSnapshot) RegistryDigest() string {
	return s.object.RegistryDigest()
}

func (s CatalogSnapshot) Kits() []ManifestSnapshot {
	return cloneManifestSnapshots(s.kits)
}

func (s CatalogSnapshot) Select(kit string, all bool) ([]ManifestSnapshot, error) {
	if all && kit != "" {
		return nil, errors.New("use either --all or --kit, not both")
	}
	if !all && kit == "" {
		return nil, errors.New("kit verify/test requires --all or --kit")
	}
	selected := []ManifestSnapshot{}
	for _, item := range s.kits {
		entry := item.Entry()
		if all && entry.Enabled {
			selected = append(selected, item)
		}
		if kit != "" && entry.ID == kit {
			selected = append(selected, item)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no kit matched")
	}
	return cloneManifestSnapshots(selected), nil
}

func cloneManifestSnapshots(snapshots []ManifestSnapshot) []ManifestSnapshot {
	out := make([]ManifestSnapshot, len(snapshots))
	copy(out, snapshots)
	return out
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
			v.Source = clonePinnedSource(m.Source)
			v.SourceIdentity, _ = PinnedSourceIdentity(m.Source)
		}
		views = append(views, v)
	}
	return views
}

func CatalogKitViews(snapshots []ManifestSnapshot) []View {
	views := make([]View, 0, len(snapshots))
	for _, snapshot := range snapshots {
		entry := snapshot.Entry()
		view := View{ID: entry.ID, Enabled: entry.Enabled, Order: entry.Order, Manifest: entry.Manifest}
		if manifest, err := snapshot.Manifest(); err == nil {
			view.Name = manifest.Name
			view.Version = manifest.Version
			view.Kind = append([]string{}, manifest.Kind...)
			view.Mode = manifest.Mode
			view.Source = clonePinnedSource(manifest.Source)
			view.SourceIdentity, _ = PinnedSourceIdentity(manifest.Source)
		}
		views = append(views, view)
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
