// Package workstate owns worktree-local Loop Engineering session state.
package workstate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/JiaxI2/AiCoding/internal/loopkit/transition"
)

const schemaVersion = 1

// Snapshot is the replaceable projection beside the immutable attempt log.
type Snapshot struct {
	SchemaVersion      int                 `json:"schemaVersion"`
	WorkID             string              `json:"workID"`
	SpecDigest         string              `json:"specDigest"`
	SpecFile           string              `json:"specFile"`
	Attempts           int                 `json:"attempts"`
	LastSubjectTreeOID string              `json:"lastSubjectTreeOID,omitempty"`
	LastDecision       transition.Decision `json:"lastDecision"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

type Session struct {
	Exists   bool                 `json:"exists"`
	Root     string               `json:"root"`
	Snapshot Snapshot             `json:"snapshot"`
	History  []transition.Attempt `json:"history"`
}

// Load reads one state.json and one attempts.jsonl without scanning other work IDs.
func Load(repo, workID string) (Session, error) {
	root := sessionRoot(repo, workID)
	statePath := filepath.Join(root, "state.json")
	logPath := filepath.Join(root, "attempts.jsonl")
	session := Session{Root: root, History: []transition.Attempt{}}

	stateData, stateErr := os.ReadFile(statePath)
	if stateErr == nil {
		if err := decodeStrict(stateData, &session.Snapshot); err != nil {
			return Session{}, fmt.Errorf("decode state.json: %w", err)
		}
		if session.Snapshot.SchemaVersion != schemaVersion || session.Snapshot.WorkID != workID {
			return Session{}, errors.New("state.json identity mismatch")
		}
		session.Exists = true
	} else if !os.IsNotExist(stateErr) {
		return Session{}, fmt.Errorf("read state.json: %w", stateErr)
	}

	file, logErr := os.Open(logPath)
	if logErr == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for line := 1; scanner.Scan(); line++ {
			var attempt transition.Attempt
			if err := decodeStrict(scanner.Bytes(), &attempt); err != nil {
				return Session{}, fmt.Errorf("decode attempts.jsonl line %d: %w", line, err)
			}
			session.History = append(session.History, attempt)
		}
		if err := scanner.Err(); err != nil {
			return Session{}, fmt.Errorf("read attempts.jsonl: %w", err)
		}
	} else if !os.IsNotExist(logErr) {
		return Session{}, fmt.Errorf("read attempts.jsonl: %w", logErr)
	}

	if len(session.History) > 0 && !session.Exists {
		return Session{}, errors.New("attempts.jsonl exists without state.json")
	}
	if session.Exists && session.Snapshot.Attempts != len(session.History) {
		return Session{}, errors.New("state.json attempt count does not match attempts.jsonl")
	}
	return session, nil
}

// Record appends one immutable attempt and atomically refreshes state.json.
func Record(repo, workID, specDigest, specFile string, attempt transition.Attempt, decision transition.Decision) (Session, error) {
	session, err := Load(repo, workID)
	if err != nil {
		return Session{}, err
	}
	if session.Exists && session.Snapshot.SpecDigest != specDigest {
		return Session{}, errors.New("work spec digest differs from the recorded session")
	}
	expected := len(session.History) + 1
	if attempt.Number != expected {
		return Session{}, fmt.Errorf("attempt number %d does not match expected %d", attempt.Number, expected)
	}
	if attempt.SubjectTreeOID == "" || attempt.StartedAt.IsZero() || attempt.EndedAt.IsZero() {
		return Session{}, errors.New("attempt tree and timestamps are required")
	}
	if attempt.EndedAt.Before(attempt.StartedAt) {
		return Session{}, errors.New("attempt endedAt precedes startedAt")
	}

	if err := os.MkdirAll(session.Root, 0o755); err != nil {
		return Session{}, fmt.Errorf("create work state directory: %w", err)
	}
	line, err := json.Marshal(attempt)
	if err != nil {
		return Session{}, fmt.Errorf("encode attempt: %w", err)
	}
	logPath := filepath.Join(session.Root, "attempts.jsonl")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Session{}, fmt.Errorf("open attempts.jsonl: %w", err)
	}
	if _, err = logFile.Write(append(line, '\n')); err == nil {
		err = logFile.Sync()
	}
	closeErr := logFile.Close()
	if err != nil {
		return Session{}, fmt.Errorf("append attempts.jsonl: %w", err)
	}
	if closeErr != nil {
		return Session{}, fmt.Errorf("close attempts.jsonl: %w", closeErr)
	}

	snapshot := Snapshot{
		SchemaVersion:      schemaVersion,
		WorkID:             workID,
		SpecDigest:         specDigest,
		SpecFile:           specFile,
		Attempts:           expected,
		LastSubjectTreeOID: attempt.SubjectTreeOID,
		LastDecision:       decision,
		UpdatedAt:          attempt.EndedAt.UTC(),
	}
	if err := writeStateAtomic(filepath.Join(session.Root, "state.json"), snapshot); err != nil {
		return Session{}, err
	}
	session.Exists = true
	session.Snapshot = snapshot
	session.History = append(session.History, attempt)
	return session, nil
}

func sessionRoot(repo, workID string) string {
	return filepath.Join(repo, ".aicoding", "state", "work", workID)
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeStateAtomic(path string, state Snapshot) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state.json: %w", err)
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(path), "state-*.tmp")
	if err != nil {
		return fmt.Errorf("create state temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish state.json: %w", err)
	}
	return nil
}
