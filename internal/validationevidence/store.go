package validationevidence

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var reportNames = []string{"report.md", "results.json", "summary.json"}

// Put atomically retains PASS evidence. Existing evidence for the same identity
// stays immutable; concurrent writers converge on the first complete report.
func (r Repository) Put(receipt Receipt, reports ReportBundle) (Receipt, error) {
	if !validFingerprint(receipt.Fingerprint) || receipt.Fingerprint.RepositoryID != r.repositoryID || receipt.ValidationIdentity != receipt.Fingerprint.Identity {
		return Receipt{}, fingerprintError("receipt fingerprint does not match its validation identity")
	}
	if receipt.Conclusion != "PASS" || !receipt.Reusable {
		return Receipt{}, &Error{Code: CodeReceiptInvalid, Message: "only reusable PASS conclusions may produce a Receipt", RequiredAction: "run the complete selected profile successfully"}
	}
	if !receipt.Scope.IgnoredFilesOutOfScope {
		return Receipt{}, &Error{Code: CodeReceiptInvalid, Message: "Receipt scope must declare ignored files out of scope", RequiredAction: "capture the Git-tree evidence boundary explicitly"}
	}
	if existing, err := r.readReceipt(receipt.Fingerprint.Profile, receipt.ValidationIdentity); err == nil {
		return existing, nil
	}
	artifacts, err := r.writeReportDir(receipt.ValidationIdentity, reports)
	if err != nil {
		return Receipt{}, err
	}
	receipt.SchemaVersion = schemaVersion
	receipt.ReceiptID = ""
	receipt.Reports = artifacts
	receipt.ReceiptID = receiptDigest(receipt)
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return Receipt{}, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "rerun validation"}
	}
	encoded = append(encoded, '\n')
	if err := atomicWriteFile(r.receiptPath(receipt.Fingerprint.Profile, receipt.ValidationIdentity), encoded); err != nil {
		return Receipt{}, &Error{Code: CodeStoreError, Message: fmt.Sprintf("write Receipt: %v", err), RequiredAction: "check Git common-dir permissions and rerun validation"}
	}
	return r.readReceipt(receipt.Fingerprint.Profile, receipt.ValidationIdentity)
}

// List returns integrity-checked Receipts in stable identity order.
func (r Repository) List(profile string) ([]Receipt, error) {
	profiles, err := r.profileDirs(profile)
	if err != nil {
		return nil, err
	}
	receipts := make([]Receipt, 0)
	for _, selected := range profiles {
		dir := filepath.Join(r.root, "receipts", selected)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			hexID := strings.TrimSuffix(entry.Name(), ".json")
			if !validHexDigest(hexID) {
				return nil, invalidReceipt("invalid Receipt filename")
			}
			receipt, err := r.readReceipt(selected, "sha256:"+hexID)
			if err != nil {
				return nil, err
			}
			receipts = append(receipts, receipt)
		}
	}
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].ValidationIdentity < receipts[j].ValidationIdentity })
	return receipts, nil
}

// Clean removes finalized Receipt files first, then their now-unreferenced
// report files. It never removes temporary writer directories.
func (r Repository) Clean(profile string) (int, error) {
	profiles, err := r.profileDirs(profile)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, selected := range profiles {
		dir := filepath.Join(r.root, "receipts", selected)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return removed, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			hexID := strings.TrimSuffix(entry.Name(), ".json")
			if !validHexDigest(hexID) {
				continue
			}
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return removed, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "retry validation clean"}
			}
			removed++
			r.removeReportDir("sha256:" + hexID)
		}
		_ = os.Remove(dir)
	}
	return removed, nil
}

func (r Repository) readReceipt(profile, identity string) (Receipt, error) {
	var receipt Receipt
	if !validProfile(profile) || !validDigest(identity) {
		return receipt, invalidReceipt("validation identity is invalid")
	}
	raw, err := os.ReadFile(r.receiptPath(profile, identity))
	if err != nil {
		return receipt, err
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return Receipt{}, invalidReceipt("Receipt JSON is invalid")
	}
	if err := r.validateReceipt(receipt, identity, profile); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func (r Repository) validateReceipt(receipt Receipt, identity, profile string) error {
	if receipt.SchemaVersion != schemaVersion || receipt.ValidationIdentity != identity || receipt.Fingerprint.Profile != profile || receipt.Fingerprint.RepositoryID != r.repositoryID || !validFingerprint(receipt.Fingerprint) {
		return invalidReceipt("Receipt identity or schema is invalid")
	}
	if receipt.ReceiptID != receiptDigest(receipt) {
		return invalidReceipt("Receipt integrity check failed")
	}
	if receipt.Conclusion != "PASS" || !receipt.Reusable || !receipt.Scope.IgnoredFilesOutOfScope {
		return invalidReceipt("Receipt is not reusable PASS evidence")
	}
	actual, err := r.readReportArtifacts(identity)
	if err != nil {
		return invalidReceipt(err.Error())
	}
	if !sameArtifacts(actual, receipt.Reports) {
		return invalidReceipt("retained report integrity check failed")
	}
	return nil
}

func (r Repository) writeReportDir(identity string, reports ReportBundle) ([]ReportArtifact, error) {
	contents := map[string][]byte{
		"results.json": reports.ResultsJSON,
		"summary.json": reports.SummaryJSON,
		"report.md":    reports.ReportMarkdown,
	}
	for _, name := range reportNames {
		if len(contents[name]) == 0 {
			return nil, &Error{Code: CodeStoreError, Message: name + " is empty", RequiredAction: "write the complete test report before the Receipt"}
		}
	}
	finalDir := r.reportDir(identity)
	if artifacts, err := r.readReportArtifacts(identity); err == nil {
		return artifacts, nil
	}
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return nil, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
	}
	tempDir, err := os.MkdirTemp(filepath.Dir(finalDir), ".tmp-report-")
	if err != nil {
		return nil, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
	}
	defer os.RemoveAll(tempDir)
	artifacts := make([]ReportArtifact, 0, len(reportNames))
	for _, name := range reportNames {
		if err := writeSyncedFile(filepath.Join(tempDir, name), contents[name]); err != nil {
			return nil, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "retry validation"}
		}
		artifacts = append(artifacts, ReportArtifact{Name: name, Digest: digestBytes(contents[name])})
	}
	if err := os.Rename(tempDir, finalDir); err != nil {
		if existing, readErr := r.readReportArtifacts(identity); readErr == nil {
			return existing, nil
		}
		return nil, &Error{Code: CodeStoreError, Message: fmt.Sprintf("publish report: %v", err), RequiredAction: "retry validation"}
	}
	return artifacts, nil
}

func (r Repository) readReportArtifacts(identity string) ([]ReportArtifact, error) {
	dir := r.reportDir(identity)
	artifacts := make([]ReportArtifact, 0, len(reportNames))
	for _, name := range reportNames {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, ReportArtifact{Name: name, Digest: digestBytes(content)})
	}
	return artifacts, nil
}

func (r Repository) removeReportDir(identity string) {
	dir := r.reportDir(identity)
	for _, name := range reportNames {
		_ = os.Remove(filepath.Join(dir, name))
	}
	_ = os.Remove(dir)
}

func (r Repository) profileDirs(profile string) ([]string, error) {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile != "" {
		if !validProfile(profile) {
			return nil, fingerprintError("profile is invalid")
		}
		return []string{profile}, nil
	}
	root := filepath.Join(r.root, "receipts")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && validProfile(entry.Name()) {
			profiles = append(profiles, entry.Name())
		}
	}
	sort.Strings(profiles)
	return profiles, nil
}

func (r Repository) receiptPath(profile, identity string) string {
	return filepath.Join(r.root, "receipts", profile, digestHex(identity)+".json")
}

func (r Repository) reportDir(identity string) string {
	return filepath.Join(r.root, "reports", digestHex(identity))
}

func receiptDigest(receipt Receipt) string {
	receipt.ReceiptID = ""
	payload, _ := json.Marshal(receipt)
	return digestBytes(payload)
}

func sameArtifacts(a, b []ReportArtifact) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func validFingerprint(fingerprint Fingerprint) bool {
	if !validDigest(fingerprint.Identity) || !validDigest(fingerprint.RepositoryID) || !validTreeOID(fingerprint.SubjectTreeOID) || !validProfile(fingerprint.Profile) {
		return false
	}
	for _, digest := range []string{fingerprint.ValidationPlanDigest, fingerprint.EngineSemanticDigest, fingerprint.ConfigDigest, fingerprint.ToolchainDigest, fingerprint.OptionsDigest} {
		if !validDigest(digest) {
			return false
		}
	}
	copy := fingerprint
	copy.Identity = ""
	payload, _ := json.Marshal(copy)
	return fingerprint.Identity == digestBytes(payload)
}

func validProfile(profile string) bool {
	if len(profile) < 1 || len(profile) > 32 {
		return false
	}
	for index, char := range profile {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' && index > 0 || (char == '-' || char == '_') && index > 0 {
			continue
		}
		return false
	}
	return true
}

func validTreeOID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func validDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validHexDigest(strings.TrimPrefix(value, "sha256:"))
}

func validHexDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digestHex(identity string) string {
	return strings.TrimPrefix(identity, "sha256:")
}

func invalidReceipt(message string) *Error {
	return &Error{Code: CodeReceiptInvalid, Message: message, RequiredAction: "rerun the selected validation profile without reuse"}
}

func atomicWriteFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-evidence-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		existing, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Equal(existing, content) {
			return nil
		}
		return err
	}
	return nil
}

func writeSyncedFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
