package registry

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Snapshot struct {
	kind    string
	digest  string
	content []byte
}

type View struct {
	Kind    string          `json:"kind"`
	Digest  string          `json:"digest"`
	Content json.RawMessage `json:"content"`
}

func NewSnapshot(kind string, value interface{}) (Snapshot, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return Snapshot{}, errors.New("snapshot kind is required")
	}
	content, err := json.Marshal(value)
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal %s snapshot: %w", kind, err)
	}
	payload, err := json.Marshal(struct {
		Kind    string          `json:"kind"`
		Content json.RawMessage `json:"content"`
	}{Kind: kind, Content: content})
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal %s digest payload: %w", kind, err)
	}
	sum := sha256.Sum256(payload)
	return Snapshot{
		kind:    kind,
		digest:  fmt.Sprintf("sha256:%x", sum),
		content: append([]byte(nil), content...),
	}, nil
}

func (s Snapshot) Kind() string {
	return s.kind
}

func (s Snapshot) Digest() string {
	return s.digest
}

func (s Snapshot) Decode(target interface{}) error {
	if target == nil {
		return errors.New("snapshot decode target is required")
	}
	if len(s.content) == 0 {
		return errors.New("snapshot content is empty")
	}
	return json.Unmarshal(s.content, target)
}

func (s Snapshot) View() View {
	return View{
		Kind:    s.kind,
		Digest:  s.digest,
		Content: append(json.RawMessage(nil), s.content...),
	}
}

func (s Snapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.View())
}
