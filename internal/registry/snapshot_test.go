package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotDigestUsesCanonicalContent(t *testing.T) {
	first, err := NewSnapshot("command-registry", map[string]interface{}{
		"commands": []string{"doctor", "verify"},
		"metadata": map[string]string{"layer": "platform", "owner": "cli"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSnapshot("command-registry", map[string]interface{}{
		"metadata": map[string]string{"owner": "cli", "layer": "platform"},
		"commands": []string{"doctor", "verify"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() || !strings.HasPrefix(first.Digest(), "sha256:") {
		t.Fatalf("canonical digests differ: %q != %q", first.Digest(), second.Digest())
	}

	changed, _ := NewSnapshot("command-registry", map[string]interface{}{"commands": []string{"verify", "doctor"}})
	if first.Digest() == changed.Digest() {
		t.Fatal("ordered content change did not change digest")
	}
}

func TestSnapshotViewIsInspectableAndDetached(t *testing.T) {
	snapshot, err := NewSnapshot("kit-registry", map[string]string{"name": "default"})
	if err != nil {
		t.Fatal(err)
	}
	view := snapshot.View()
	view.Content[0] = '['

	var decoded map[string]string
	if err := snapshot.Decode(&decoded); err != nil {
		t.Fatalf("snapshot content was mutated through view: %v", err)
	}
	if decoded["name"] != "default" {
		t.Fatalf("unexpected decoded content: %#v", decoded)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"kind":"kit-registry"`) || !strings.Contains(string(data), `"digest":"sha256:`) {
		t.Fatalf("snapshot is not inspectable: %s", data)
	}
}

func TestSnapshotRejectsMissingKind(t *testing.T) {
	if _, err := NewSnapshot(" ", map[string]string{}); err == nil {
		t.Fatal("missing kind was accepted")
	}
}
