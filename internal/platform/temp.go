package platform

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

const TempDirectoryPrefix = "aicoding-"

// TempRecord is one append-only lifecycle event for a repository-owned
// temporary directory. Later events supersede earlier outcomes for the same
// path, but the ledger itself is never rewritten.
type TempRecord struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
	RepoRoot  string `json:"repoRoot"`
	Outcome   string `json:"outcome"`
	SizeBytes int64  `json:"sizeBytes"`
}

func TempLedgerPath(repo string) (string, error) {
	commonDir, err := gitx.CommonDir(repo)
	if err != nil {
		return "", err
	}
	return filepath.Join(commonDir, "aicoding", "temp-ledger.jsonl"), nil
}

// CreateTempDir creates one direct child of the system temp directory and
// records ownership before returning it. A ledger failure removes the new
// directory so callers never receive an unowned resource.
func CreateTempDir(repo, kind string) (string, error) {
	if !validTempKind(kind) {
		return "", fmt.Errorf("invalid temp kind %q", kind)
	}
	path, err := os.MkdirTemp("", TempDirectoryPrefix+kind+"-")
	if err != nil {
		return "", err
	}
	if err := appendTempRecord(repo, TempRecord{Path: path, Kind: kind, Outcome: "created"}); err != nil {
		_ = os.RemoveAll(path)
		return "", err
	}
	return path, nil
}

// RecordTempOutcome appends a lifecycle event without deleting anything.
func RecordTempOutcome(repo, path, kind, outcome string) error {
	if !validTempKind(kind) {
		return fmt.Errorf("invalid temp kind %q", kind)
	}
	if !validTempOutcome(outcome) {
		return fmt.Errorf("invalid temp outcome %q", outcome)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(filepath.Base(absPath), TempDirectoryPrefix) {
		return fmt.Errorf("refuse to record non-%s temp path: %s", TempDirectoryPrefix, absPath)
	}
	size, err := tempPathSize(absPath)
	if err != nil {
		return err
	}
	return appendTempRecord(repo, TempRecord{Path: absPath, Kind: kind, Outcome: outcome, SizeBytes: size})
}

// ReleaseTempDir removes only a direct aicoding-* child of the system temp
// directory. It records both the intent and the completed release; it never
// expands the deletion boundary to an arbitrary caller-supplied path.
func ReleaseTempDir(repo, path, kind string) error {
	absPath, err := validateSystemTempPath(path)
	if err != nil {
		return err
	}
	if err := RecordTempOutcome(repo, absPath, kind, "releasing"); err != nil {
		return err
	}
	if err := os.RemoveAll(absPath); err != nil {
		return err
	}
	return appendTempRecord(repo, TempRecord{Path: absPath, Kind: kind, Outcome: "released"})
}

func ReadTempLedger(repo string) ([]TempRecord, error) {
	path, err := TempLedgerPath(repo)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return []TempRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records := []TempRecord{}
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var record TempRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("decode temp ledger line %d: %w", line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func appendTempRecord(repo string, record TempRecord) error {
	repoRoot, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	path, err := filepath.Abs(record.Path)
	if err != nil {
		return err
	}
	record.Path = filepath.Clean(path)
	record.RepoRoot = filepath.Clean(repoRoot)
	if record.CreatedAt == "" {
		record.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	ledgerPath, err := TempLedgerPath(repo)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(ledgerPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	payload = append(payload, '\n')
	if _, err := file.Write(payload); err != nil {
		return err
	}
	return file.Sync()
}

func validateSystemTempPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return "", err
	}
	if !sameFilesystemPath(filepath.Dir(absPath), tempRoot) || !strings.HasPrefix(filepath.Base(absPath), TempDirectoryPrefix) {
		return "", fmt.Errorf("refuse to release path outside direct %s children of %s: %s", TempDirectoryPrefix, tempRoot, absPath)
	}
	return filepath.Clean(absPath), nil
}

func validTempKind(kind string) bool {
	if kind == "" || strings.Trim(kind, "-") != kind {
		return false
	}
	for _, char := range kind {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func validTempOutcome(outcome string) bool {
	switch outcome {
	case "created", "failed", "investigating", "adopted", "releasing", "released":
		return true
	default:
		return false
	}
}

func tempPathSize(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, walkErr error) error {
		if os.IsNotExist(walkErr) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return size, err
}

func sameFilesystemPath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
