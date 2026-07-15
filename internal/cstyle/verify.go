package cstyle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	kitVerifyTimeout    = 6 * time.Minute
	verifyPayloadSchema = "cstylekit.verify.v1"
)

type VerifyOptions struct {
	RepoRoot string
	Profile  string
	Target   string
	Overlays []string
	Timings  bool
}

type VerifyResult struct {
	SkillID    string                 `json:"skillId"`
	KitID      string                 `json:"kitId"`
	KitVersion string                 `json:"kitVersion,omitempty"`
	Profile    string                 `json:"profile"`
	KitRoot    string                 `json:"kitRoot"`
	Config     string                 `json:"config"`
	Target     string                 `json:"target"`
	Overlays   []string               `json:"overlays,omitempty"`
	Timings    bool                   `json:"timings"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	Stderr     string                 `json:"stderr,omitempty"`
	ElapsedMS  int64                  `json:"elapsedMs"`
}

type verifyCommandRunner interface {
	Run(ctx context.Context, repoRoot, kitRoot string, args []string) ([]byte, []byte, error)
}

type osVerifyCommandRunner struct{}

func (osVerifyCommandRunner) Run(
	ctx context.Context,
	repoRoot string,
	kitRoot string,
	args []string,
) ([]byte, []byte, error) {
	goArgs := append([]string{"-C", kitRoot, "run", "./cmd/cstylekit"}, args...)
	cmd := exec.CommandContext(ctx, "go", goArgs...)
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func VerifyBySkill(skillID string, opts VerifyOptions) (VerifyResult, error) {
	return verifyBySkill(skillID, opts, osVerifyCommandRunner{})
}

func verifyBySkill(
	skillID string,
	opts VerifyOptions,
	runner verifyCommandRunner,
) (VerifyResult, error) {
	started := time.Now()
	result := VerifyResult{
		SkillID: normalizeSkillID(skillID),
		Profile: strings.ToLower(strings.TrimSpace(opts.Profile)),
		Timings: opts.Timings,
	}
	if result.Profile == "" {
		result.Profile = "fast"
	}
	if result.Profile != "fast" && result.Profile != "full" {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, fmt.Errorf("unsupported C Kit verify profile %q", opts.Profile)
	}

	repoRoot, err := resolveRepoRoot(opts.RepoRoot)
	if err != nil {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, err
	}
	cfg, err := LoadSkillConfig(repoRoot, result.SkillID)
	if err != nil {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, err
	}
	paths, err := ResolveKitPaths(repoRoot, cfg)
	if err != nil {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, err
	}

	result.KitID = paths.ID
	result.KitVersion = paths.Version
	result.KitRoot = relativeRepoPath(repoRoot, paths.Root)
	result.Config = relativeRepoPath(repoRoot, paths.Config)
	targetPath := paths.QuickTarget
	if strings.TrimSpace(opts.Target) != "" {
		targetPath = resolveRepoRelativePath(repoRoot, opts.Target)
	}
	result.Target = relativeRepoPath(repoRoot, targetPath)

	if !directoryExists(paths.Root) {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, fmt.Errorf("C Kit root not found: %s", result.KitRoot)
	}
	if !fileExists(paths.Config) {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, fmt.Errorf("C Kit config not found: %s", result.Config)
	}
	if !fileExists(targetPath) {
		result.ElapsedMS = time.Since(started).Milliseconds()
		return result, fmt.Errorf("C Kit verify target not found: %s", result.Target)
	}

	args := []string{
		"verify",
		"--config", paths.Config,
		"--target", targetPath,
		"--profile", result.Profile,
		"--json",
	}
	for _, overlay := range opts.Overlays {
		overlayPath := resolveRepoRelativePath(repoRoot, overlay)
		if !fileExists(overlayPath) {
			result.ElapsedMS = time.Since(started).Milliseconds()
			return result, fmt.Errorf("C Kit overlay not found: %s", relativeRepoPath(repoRoot, overlayPath))
		}
		result.Overlays = append(result.Overlays, relativeRepoPath(repoRoot, overlayPath))
		args = append(args, "--overlay", overlayPath)
	}
	if opts.Timings {
		args = append(args, "--timings")
	}

	ctx, cancel := context.WithTimeout(context.Background(), kitVerifyTimeout)
	defer cancel()
	stdout, stderr, runErr := runner.Run(ctx, repoRoot, paths.Root, args)
	result.Stderr = strings.TrimSpace(string(stderr))
	result.ElapsedMS = time.Since(started).Milliseconds()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return result, fmt.Errorf("C Kit verify timed out after %s", kitVerifyTimeout)
	}

	payload, decodeErr := decodeVerifyPayload(stdout)
	if decodeErr != nil {
		if result.Stderr != "" {
			return result, fmt.Errorf("invalid C Kit verify JSON: %w; stderr: %s", decodeErr, result.Stderr)
		}
		return result, fmt.Errorf("invalid C Kit verify JSON: %w", decodeErr)
	}
	result.Payload = payload

	schema, schemaOK := payload["schema"].(string)
	if !schemaOK || schema != verifyPayloadSchema {
		return result, fmt.Errorf("C Kit verify JSON schema must be %q", verifyPayloadSchema)
	}
	payloadProfile, profileOK := payload["profile"].(string)
	if !profileOK || payloadProfile != result.Profile {
		return result, fmt.Errorf("C Kit verify JSON profile must match requested profile %q", result.Profile)
	}
	ok, okType := payload["ok"].(bool)
	if !okType {
		return result, errors.New("C Kit verify JSON field ok must be boolean")
	}
	if !ok {
		if result.Stderr != "" {
			return result, fmt.Errorf("C Kit verify reported ok=false: %s", result.Stderr)
		}
		return result, errors.New("C Kit verify reported ok=false")
	}
	if runErr != nil {
		if result.Stderr != "" {
			return result, fmt.Errorf("C Kit verify process failed: %w: %s", runErr, result.Stderr)
		}
		return result, fmt.Errorf("C Kit verify process failed: %w", runErr)
	}
	return result, nil
}

func decodeVerifyPayload(stdout []byte) (map[string]interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	decoder.UseNumber()
	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, errors.New("JSON root must be an object")
	}
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("multiple JSON values")
		}
		return nil, err
	}
	return payload, nil
}
