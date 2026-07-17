package assetkit

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManifest(t *testing.T) {
	good := Manifest{SchemaVersion: 1, ID: "c99-standard-c", Type: "skill", Version: "1.0.0", Mode: "managed"}
	if err := ValidateManifest(good); err != nil { t.Fatalf("valid manifest rejected: %v", err) }
	bad := good; bad.ID = "Bad ID"
	if err := ValidateManifest(bad); err == nil { t.Fatal("invalid id accepted") }
}

func TestPackIsDeterministic(t *testing.T) {
	dir := makeAsset(t, "demo-skill", "1.0.0")
	one := filepath.Join(t.TempDir(), "one.zip")
	two := filepath.Join(t.TempDir(), "two.zip")
	h1, err := Pack(dir, one); if err != nil { t.Fatal(err) }
	h2, err := Pack(dir, two); if err != nil { t.Fatal(err) }
	if h1 != h2 { t.Fatalf("package checksum differs: %s != %s", h1, h2) }
}

func TestLifecycleManagedAndConfigMerge(t *testing.T) {
	dir := makeAsset(t, "demo-skill", "1.0.0")
	pkg := filepath.Join(t.TempDir(), "demo.aicoding.zip")
	if _, err := Pack(dir, pkg); err != nil { t.Fatal(err) }
	store := Store{Root: filepath.Join(t.TempDir(), "store")}
	m, _, err := store.Import(pkg); if err != nil { t.Fatal(err) }
	if m.ID != "demo-skill" { t.Fatalf("wrong id %q", m.ID) }
	r, err := store.Install(m.ID, m.Version, "managed"); if err != nil { t.Fatal(err) }
	if r.Mode != "managed" { t.Fatalf("wrong mode %q", r.Mode) }
	if err := os.WriteFile(filepath.Join(store.Root, "config", "global.json"), []byte(`{"rules":{"encoding":"UTF-8","strict":true}}`), 0o644); err != nil { t.Fatal(err) }
	if err := store.SetConfig(m.ID, "rules.encoding", "GBK"); err != nil { t.Fatal(err) }
	effective, err := store.EffectiveConfig(m.ID); if err != nil { t.Fatal(err) }
	rules := effective["rules"].(map[string]any)
	if rules["encoding"] != "GBK" { t.Fatalf("override not applied: %#v", rules) }
	if rules["strict"] != true { t.Fatalf("global config lost: %#v", rules) }
	if err := store.Uninstall(m.ID, false); err != nil { t.Fatal(err) }
	if _, err := store.Inspect(m.ID); !os.IsNotExist(err) { t.Fatalf("lock still exists: %v", err) }
	if _, err := os.Stat(filepath.Join(store.Root, "config", m.ID+".override.json")); err != nil { t.Fatalf("user override should be retained: %v", err) }
}

func TestEditableAssetRejectsManagedUpdate(t *testing.T) {
	dir := makeAsset(t, "editable-skill", "1.0.0")
	pkg := filepath.Join(t.TempDir(), "editable.zip")
	if _, err := Pack(dir, pkg); err != nil { t.Fatal(err) }
	store := Store{Root: filepath.Join(t.TempDir(), "store")}
	if _, _, err := store.Import(pkg); err != nil { t.Fatal(err) }
	if _, err := store.Install("editable-skill", "1.0.0", "editable"); err != nil { t.Fatal(err) }
	if _, err := store.Update("editable-skill", "1.0.0"); err == nil { t.Fatal("editable update should require explicit merge") }
}

func TestImportRejectsZipSlip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsafe.zip")
	f, err := os.Create(path); if err != nil { t.Fatal(err) }
	zw := zip.NewWriter(f)
	w, err := zw.Create("../escape.txt"); if err != nil { t.Fatal(err) }
	if _, err := w.Write([]byte("bad")); err != nil { t.Fatal(err) }
	if err := zw.Close(); err != nil { t.Fatal(err) }; if err := f.Close(); err != nil { t.Fatal(err) }
	store := Store{Root: filepath.Join(t.TempDir(), "store")}
	if _, _, err := store.Import(path); err == nil { t.Fatal("zip slip archive accepted") }
}

func makeAsset(t *testing.T, id, version string) string {
	t.Helper()
	dir := t.TempDir()
	m := Manifest{SchemaVersion: 1, ID: id, Type: "skill", Version: version, Name: id, Mode: "managed", Payload: "payload", Config: ConfigSpec{Defaults: "defaults/config.json"}}
	data, err := json.MarshalIndent(m, "", "  "); if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "asset.json"), append(data, '\n'), 0o644); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(filepath.Join(dir, "payload"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "payload", "SKILL.md"), []byte("---\nname: demo\n---\n"), 0o644); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(filepath.Join(dir, "defaults"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "defaults", "config.json"), []byte(`{"rules":{"encoding":"UTF-8","strict":false}}`), 0o644); err != nil { t.Fatal(err) }
	return dir
}
