package assetkit

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const SchemaVersion = 1

var validTypes = map[string]bool{"kit": true, "skill": true, "mcp": true, "template": true, "ruleset": true, "profile": true}
var validModes = map[string]bool{"managed": true, "editable": true}

type Manifest struct {
	SchemaVersion int                    `json:"schemaVersion"`
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Version       string                 `json:"version"`
	Name          string                 `json:"name,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Mode          string                 `json:"mode,omitempty"`
	Payload       string                 `json:"payload,omitempty"`
	Platforms     []string               `json:"platforms,omitempty"`
	Capabilities  []string               `json:"capabilities,omitempty"`
	Dependencies  Dependencies           `json:"dependencies,omitempty"`
	Config        ConfigSpec             `json:"config,omitempty"`
	Metadata      map[string]any         `json:"metadata,omitempty"`
}

type Dependencies struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

type ConfigSpec struct {
	Defaults string `json:"defaults,omitempty"`
	Schema   string `json:"schema,omitempty"`
}

type LockRecord struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Version       string    `json:"version"`
	Mode          string    `json:"mode"`
	Source        string    `json:"source"`
	Checksum      string    `json:"checksum"`
	InstalledAt   time.Time `json:"installedAt"`
}

type Store struct {
	Root string
}

func DefaultStore(repoRoot string) Store {
	return Store{Root: filepath.Join(repoRoot, ".aicoding", "assets")}
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil { return Manifest{}, err }
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil { return Manifest{}, fmt.Errorf("parse manifest: %w", err) }
	if err := ValidateManifest(m); err != nil { return Manifest{}, err }
	return m, nil
}

func ValidateManifest(m Manifest) error {
	var problems []string
	if m.SchemaVersion != SchemaVersion { problems = append(problems, fmt.Sprintf("schemaVersion must be %d", SchemaVersion)) }
	if !validID(m.ID) { problems = append(problems, "id must use lowercase letters, digits, dots, dashes, or underscores") }
	if !validTypes[m.Type] { problems = append(problems, "type must be kit, skill, mcp, template, ruleset, or profile") }
	if strings.TrimSpace(m.Version) == "" { problems = append(problems, "version is required") }
	if m.Mode != "" && !validModes[m.Mode] { problems = append(problems, "mode must be managed or editable") }
	if m.Payload != "" && (filepath.IsAbs(m.Payload) || strings.Contains(filepath.Clean(m.Payload), "..")) { problems = append(problems, "payload must be a safe relative path") }
	if len(problems) > 0 { return errors.New(strings.Join(problems, "; ")) }
	return nil
}

func validID(s string) bool {
	if s == "" { return false }
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' { continue }
		return false
	}
	return true
}

func Pack(assetDir, outPath string) (string, error) {
	manifestPath := filepath.Join(assetDir, "asset.json")
	if _, err := LoadManifest(manifestPath); err != nil { return "", err }
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil { return "", err }
	tmp := outPath + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil { return "", err }
	zw := zip.NewWriter(f)
	var files []string
	err = filepath.WalkDir(assetDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }
		if d.IsDir() { return nil }
		rel, err := filepath.Rel(assetDir, path)
		if err != nil { return err }
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, ".git/") || strings.Contains(rel, "/.git/") || strings.HasSuffix(rel, ".tmp") { return nil }
		files = append(files, rel)
		return nil
	})
	if err != nil { _ = zw.Close(); _ = f.Close(); return "", err }
	sort.Strings(files)
	for _, rel := range files {
		path := filepath.Join(assetDir, filepath.FromSlash(rel))
		info, err := os.Stat(path); if err != nil { return "", err }
		h, err := zip.FileInfoHeader(info); if err != nil { return "", err }
		h.Name = rel; h.Method = zip.Deflate; h.SetModTime(time.Unix(0, 0).UTC())
		w, err := zw.CreateHeader(h); if err != nil { return "", err }
		r, err := os.Open(path); if err != nil { return "", err }
		_, copyErr := io.Copy(w, r); closeErr := r.Close()
		if copyErr != nil { return "", copyErr }; if closeErr != nil { return "", closeErr }
	}
	if err := zw.Close(); err != nil { _ = f.Close(); return "", err }
	if err := f.Close(); err != nil { return "", err }
	if err := os.Rename(tmp, outPath); err != nil { return "", err }
	return FileSHA256(outPath)
}

func FileSHA256(path string) (string, error) {
	f, err := os.Open(path); if err != nil { return "", err }; defer f.Close()
	h := sha256.New(); if _, err := io.Copy(h, f); err != nil { return "", err }
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s Store) Import(packagePath string) (Manifest, string, error) {
	checksum, err := FileSHA256(packagePath); if err != nil { return Manifest{}, "", err }
	stage, err := os.MkdirTemp("", "aicoding-asset-import-"); if err != nil { return Manifest{}, "", err }; defer os.RemoveAll(stage)
	if err := unzipSafe(packagePath, stage); err != nil { return Manifest{}, "", err }
	m, err := LoadManifest(filepath.Join(stage, "asset.json")); if err != nil { return Manifest{}, "", err }
	dst := filepath.Join(s.Root, "packages", m.ID, m.Version)
	if err := replaceDir(stage, dst); err != nil { return Manifest{}, "", err }
	if err := os.WriteFile(filepath.Join(dst, ".package.sha256"), []byte(checksum+"\n"), 0o644); err != nil { return Manifest{}, "", err }
	return m, checksum, nil
}

func (s Store) Install(id, version, mode string) (LockRecord, error) {
	if !validModes[mode] { return LockRecord{}, fmt.Errorf("unsupported mode %q", mode) }
	src := filepath.Join(s.Root, "packages", id, version)
	m, err := LoadManifest(filepath.Join(src, "asset.json")); if err != nil { return LockRecord{}, err }
	payload := src; if m.Payload != "" { payload = filepath.Join(src, m.Payload) }
	dst := filepath.Join(s.Root, "installed", id)
	if mode == "editable" { dst = filepath.Join(s.Root, "editable", id) }
	if err := replaceDir(payload, dst); err != nil { return LockRecord{}, err }
	checksumData, _ := os.ReadFile(filepath.Join(src, ".package.sha256"))
	record := LockRecord{SchemaVersion: 1, ID: id, Type: m.Type, Version: version, Mode: mode, Source: src, Checksum: strings.TrimSpace(string(checksumData)), InstalledAt: time.Now().UTC()}
	if err := writeJSONAtomic(filepath.Join(s.Root, "state", id+".lock.json"), record); err != nil { return LockRecord{}, err }
	return record, nil
}

func (s Store) Update(id, version string) (LockRecord, error) {
	record, err := s.Inspect(id); if err != nil { return LockRecord{}, err }
	if record.Mode == "editable" { return LockRecord{}, errors.New("editable assets require explicit re-import or manual merge") }
	return s.Install(id, version, record.Mode)
}

func (s Store) Uninstall(id string, purge bool) error {
	record, err := s.Inspect(id)
	if err != nil && !os.IsNotExist(err) { return err }
	if record.Mode == "editable" { _ = os.RemoveAll(filepath.Join(s.Root, "editable", id)) } else { _ = os.RemoveAll(filepath.Join(s.Root, "installed", id)) }
	_ = os.Remove(filepath.Join(s.Root, "state", id+".lock.json"))
	if purge {
		_ = os.RemoveAll(filepath.Join(s.Root, "packages", id))
		_ = os.Remove(filepath.Join(s.Root, "config", id+".override.json"))
	}
	return nil
}

func (s Store) Inspect(id string) (LockRecord, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, "state", id+".lock.json")); if err != nil { return LockRecord{}, err }
	var r LockRecord; if err := json.Unmarshal(data, &r); err != nil { return LockRecord{}, err }; return r, nil
}

func (s Store) List() ([]LockRecord, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, "state")); if os.IsNotExist(err) { return nil, nil }; if err != nil { return nil, err }
	var out []LockRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".lock.json") { continue }
		id := strings.TrimSuffix(e.Name(), ".lock.json"); r, err := s.Inspect(id); if err != nil { return nil, err }; out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s Store) EffectiveConfig(id string) (map[string]any, error) {
	r, err := s.Inspect(id); if err != nil { return nil, err }
	m, err := LoadManifest(filepath.Join(r.Source, "asset.json")); if err != nil { return nil, err }
	result := map[string]any{}
	if m.Config.Defaults != "" {
		defaults, err := readJSONObject(filepath.Join(r.Source, m.Config.Defaults)); if err != nil { return nil, err }
		result = merge(result, defaults)
	}
	for _, path := range []string{filepath.Join(s.Root, "config", "global.json"), filepath.Join(s.Root, "config", id+".override.json"), filepath.Join(s.Root, "local", id+".json")} {
		obj, err := readJSONObject(path); if os.IsNotExist(err) { continue }; if err != nil { return nil, err }; result = merge(result, obj)
	}
	return result, nil
}

func (s Store) SetConfig(id, dottedKey string, value any) error {
	path := filepath.Join(s.Root, "config", id+".override.json")
	obj, err := readJSONObject(path); if os.IsNotExist(err) { obj = map[string]any{} } else if err != nil { return err }
	parts := strings.Split(dottedKey, "."); if len(parts) == 0 || dottedKey == "" { return errors.New("config key is required") }
	cursor := obj
	for _, p := range parts[:len(parts)-1] {
		next, ok := cursor[p].(map[string]any); if !ok { next = map[string]any{}; cursor[p] = next }; cursor = next
	}
	cursor[parts[len(parts)-1]] = value
	return writeJSONAtomic(path, obj)
}

func merge(base, override map[string]any) map[string]any {
	out := map[string]any{}; for k, v := range base { out[k] = v }
	for k, v := range override {
		if vm, ok := v.(map[string]any); ok { if bm, ok := out[k].(map[string]any); ok { out[k] = merge(bm, vm); continue } }
		out[k] = v
	}
	return out
}

func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path); if err != nil { return nil, err }
	var out map[string]any; if err := json.Unmarshal(data, &out); err != nil { return nil, fmt.Errorf("parse %s: %w", path, err) }; return out, nil
}

func unzipSafe(path, dst string) error {
	r, err := zip.OpenReader(path); if err != nil { return err }; defer r.Close()
	for _, f := range r.File {
		clean := filepath.Clean(filepath.FromSlash(f.Name)); if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." { return fmt.Errorf("unsafe archive path %q", f.Name) }
		target := filepath.Join(dst, clean)
		if f.FileInfo().IsDir() { if err := os.MkdirAll(target, 0o755); err != nil { return err }; continue }
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { return err }
		rc, err := f.Open(); if err != nil { return err }
		wf, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode().Perm()); if err != nil { rc.Close(); return err }
		_, copyErr := io.Copy(wf, rc); closeRead := rc.Close(); closeWrite := wf.Close()
		if copyErr != nil { return copyErr }; if closeRead != nil { return closeRead }; if closeWrite != nil { return closeWrite }
	}
	return nil
}

func replaceDir(src, dst string) error {
	parent := filepath.Dir(dst); if err := os.MkdirAll(parent, 0o755); err != nil { return err }
	stage, err := os.MkdirTemp(parent, ".asset-stage-"); if err != nil { return err }; defer os.RemoveAll(stage)
	if err := copyTree(src, stage); err != nil { return err }
	backup := dst + ".previous"; _ = os.RemoveAll(backup)
	if _, err := os.Stat(dst); err == nil { if err := os.Rename(dst, backup); err != nil { return err } }
	if err := os.Rename(stage, dst); err != nil { _ = os.Rename(backup, dst); return err }
	_ = os.RemoveAll(backup); return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }; rel, err := filepath.Rel(src, path); if err != nil { return err }; target := filepath.Join(dst, rel)
		if d.IsDir() { return os.MkdirAll(target, 0o755) }
		in, err := os.Open(path); if err != nil { return err }; defer in.Close()
		info, err := d.Info(); if err != nil { return err }; out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm()); if err != nil { return err }
		_, copyErr := io.Copy(out, in); closeErr := out.Close(); if copyErr != nil { return copyErr }; return closeErr
	})
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	data, err := json.MarshalIndent(value, "", "  "); if err != nil { return err }; data = append(data, '\n')
	tmp := path + ".tmp"; if err := os.WriteFile(tmp, data, 0o644); err != nil { return err }; return os.Rename(tmp, path)
}
