package kit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	pinnedCommitPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	pinnedDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

func (source *PinnedSource) UnmarshalJSON(content []byte) error {
	type sourceShape PinnedSource
	var decoded sourceShape
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return fmt.Errorf("invalid pinned source: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("invalid pinned source: trailing JSON value")
	}
	*source = PinnedSource(decoded)
	return source.normalizeAndValidate()
}

func (source *PinnedSource) normalizeAndValidate() error {
	if source == nil {
		return fmt.Errorf("pinned source is required")
	}
	source.Kind = strings.ToLower(strings.TrimSpace(source.Kind))
	source.URL = strings.TrimSpace(source.URL)
	source.Commit = strings.ToLower(strings.TrimSpace(source.Commit))
	source.Digest = strings.TrimSpace(source.Digest)
	switch source.Kind {
	case "git":
		if source.URL == "" || strings.HasPrefix(source.URL, "-") || strings.ContainsAny(source.URL, "\r\n\x00") {
			return fmt.Errorf("source.url must be a non-empty Git locator")
		}
		if !pinnedCommitPattern.MatchString(source.Commit) {
			return fmt.Errorf("source.commit must be an immutable 40-hex commit; branch, tag, and abbreviated SHA references are rejected")
		}
		if source.Digest != "" {
			return fmt.Errorf("git source must not define source.digest")
		}
	case "content":
		if !pinnedDigestPattern.MatchString(source.Digest) {
			return fmt.Errorf("source.digest must be a lowercase sha256 content hash")
		}
		if source.URL != "" || source.Commit != "" {
			return fmt.Errorf("content source must not define source.url or source.commit")
		}
	default:
		return fmt.Errorf("source.kind must be git or content")
	}
	return nil
}

func ValidatePinnedSource(source *PinnedSource) error {
	if source == nil {
		return fmt.Errorf("content-pinned source is required")
	}
	copy := *source
	return copy.normalizeAndValidate()
}

func PinnedSourceIdentity(source *PinnedSource) (string, error) {
	if source == nil {
		return "", fmt.Errorf("content-pinned source is required")
	}
	canonical := *source
	if err := canonical.normalizeAndValidate(); err != nil {
		return "", err
	}
	if canonical.Kind == "content" {
		return canonical.Digest, nil
	}
	content, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func clonePinnedSource(source *PinnedSource) *PinnedSource {
	if source == nil {
		return nil
	}
	copy := *source
	return &copy
}
