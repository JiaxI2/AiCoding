package registry

import (
	"errors"
	"sort"
	"strings"
)

type CatalogEntry struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type CatalogSnapshot struct {
	object         Snapshot
	registryDigest string
	entries        []CatalogEntry
}

func NewCatalogSnapshot(kind string, registry Snapshot, entries []CatalogEntry) (CatalogSnapshot, error) {
	if registry.Digest() == "" {
		return CatalogSnapshot{}, errors.New("catalog registry snapshot is required")
	}
	normalized := cloneCatalogEntries(entries)
	for index, entry := range normalized {
		entry.ID = strings.TrimSpace(entry.ID)
		entry.Path = strings.TrimSpace(entry.Path)
		entry.Digest = strings.TrimSpace(entry.Digest)
		if entry.ID == "" || entry.Path == "" || entry.Digest == "" {
			return CatalogSnapshot{}, errors.New("catalog entry id, path, and digest are required")
		}
		normalized[index] = entry
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].ID == normalized[j].ID {
			return normalized[i].Path < normalized[j].Path
		}
		return normalized[i].ID < normalized[j].ID
	})
	seen := map[string]bool{}
	for _, entry := range normalized {
		if seen[entry.ID] {
			return CatalogSnapshot{}, errors.New("duplicate catalog entry: " + entry.ID)
		}
		seen[entry.ID] = true
	}
	object, err := NewSnapshot(kind, struct {
		RegistryDigest string         `json:"registryDigest"`
		Entries        []CatalogEntry `json:"entries"`
	}{RegistryDigest: registry.Digest(), Entries: normalized})
	if err != nil {
		return CatalogSnapshot{}, err
	}
	return CatalogSnapshot{
		object:         object,
		registryDigest: registry.Digest(),
		entries:        normalized,
	}, nil
}

func (s CatalogSnapshot) Digest() string {
	return s.object.Digest()
}

func (s CatalogSnapshot) RegistryDigest() string {
	return s.registryDigest
}

func (s CatalogSnapshot) Object() Snapshot {
	return s.object
}

func (s CatalogSnapshot) Entries() []CatalogEntry {
	return cloneCatalogEntries(s.entries)
}

func cloneCatalogEntries(entries []CatalogEntry) []CatalogEntry {
	out := make([]CatalogEntry, len(entries))
	copy(out, entries)
	return out
}
