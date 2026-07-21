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
	"time"
)

var reportNames = []string{"report.md", "results.json", "summary.json"}

// Put atomically retains PASS evidence. Existing evidence for the same identity
// stays immutable; concurrent writers converge on the first complete report.
func (r Repository) Put(receipt Receipt, reports ReportBundle) (Receipt, error) {
	if receipt.Fingerprint.Node != "" {
		return Receipt{}, fingerprintError("whole-tree Receipt cannot use a node fingerprint")
	}
	return r.put(receipt, reports)
}

// PutNode atomically retains private node evidence in the same fail-closed
// store without exposing it through List or commit aliases.
func (r Repository) PutNode(receipt Receipt, reports ReportBundle) (Receipt, error) {
	if receipt.Fingerprint.Node == "" {
		return Receipt{}, fingerprintError("node Receipt requires a node fingerprint")
	}
	return r.put(receipt, reports)
}

func (r Repository) put(receipt Receipt, reports ReportBundle) (Receipt, error) {
	if !validFingerprint(receipt.Fingerprint) || receipt.Fingerprint.RepositoryID != r.repositoryID || receipt.ValidationIdentity != receipt.Fingerprint.Identity {
		return Receipt{}, fingerprintError("receipt fingerprint does not match its validation identity")
	}
	if receipt.Conclusion != "PASS" || !receipt.Reusable {
		return Receipt{}, &Error{Code: CodeReceiptInvalid, Message: "only reusable PASS conclusions may produce a Receipt", RequiredAction: "run the complete selected profile successfully"}
	}
	if !validDigest(receipt.ResultsDigest) {
		return Receipt{}, &Error{Code: CodeReceiptInvalid, Message: "Receipt results digest is invalid", RequiredAction: "rerun the complete selected profile successfully"}
	}
	if !receipt.Scope.IgnoredFilesOutOfScope {
		return Receipt{}, &Error{Code: CodeReceiptInvalid, Message: "Receipt scope must declare ignored files out of scope", RequiredAction: "capture the Git-tree evidence boundary explicitly"}
	}
	r.putMu.Lock()
	defer r.putMu.Unlock()
	node := receipt.Fingerprint.Node
	if existing, _, err := r.readStoredReceipt(receipt.Fingerprint.Profile, node, receipt.ValidationIdentity); err == nil {
		if existing.ResultsDigest != receipt.ResultsDigest {
			return Receipt{}, &Error{Code: CodeReuseAuditMismatch, Message: "executed per-case statuses do not match the existing Receipt", RequiredAction: "run with --verify-reuse and investigate the changed case statuses"}
		}
		return existing, nil
	}
	artifacts, err := r.writeReportDir(receipt.ValidationIdentity, node, reports)
	if err != nil {
		return Receipt{}, err
	}
	receipt.SchemaVersion = receiptSchemaVersion
	receipt.ReceiptID = ""
	receipt.Reports = artifacts
	receipt.ReceiptID = receiptDigest(receipt)
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return Receipt{}, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "rerun validation"}
	}
	encoded = append(encoded, '\n')
	path := r.receiptPath(receipt.Fingerprint.Profile, receipt.ValidationIdentity)
	if node != "" {
		path = r.nodeReceiptPath(receipt.Fingerprint.Profile, node, receipt.ValidationIdentity)
	}
	if err := atomicWriteFile(path, encoded); err != nil {
		return Receipt{}, &Error{Code: CodeStoreError, Message: fmt.Sprintf("write Receipt: %v", err), RequiredAction: "check Git common-dir permissions and rerun validation"}
	}
	stored, _, err := r.readStoredReceipt(receipt.Fingerprint.Profile, node, receipt.ValidationIdentity)
	return stored, err
}

// List returns integrity-checked Receipts newest-first by Receipt-file mtime.
// Identity is the deterministic tie-breaker for diagnostic consumers.
func (r Repository) List(profile string) ([]Receipt, error) {
	profiles, err := r.profileDirs(profile)
	if err != nil {
		return nil, err
	}
	type listedReceipt struct {
		receipt Receipt
		mtime   time.Time
	}
	listed := make([]listedReceipt, 0)
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
			receipt, _, err := r.readReceipt(selected, "sha256:"+hexID)
			if err != nil {
				return nil, err
			}
			info, err := entry.Info()
			if err != nil {
				return nil, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
			}
			listed = append(listed, listedReceipt{receipt: receipt, mtime: info.ModTime()})
		}
	}
	sort.Slice(listed, func(i, j int) bool {
		if listed[i].mtime.Equal(listed[j].mtime) {
			return listed[i].receipt.ValidationIdentity < listed[j].receipt.ValidationIdentity
		}
		return listed[i].mtime.After(listed[j].mtime)
	})
	receipts := make([]Receipt, len(listed))
	for index := range listed {
		receipts[index] = listed[index].receipt
	}
	return receipts, nil
}

// Clean removes commit aliases first, then finalized Receipts and their
// now-unreferenced reports. It never removes temporary writer directories.
func (r Repository) Clean(profile string) (int, error) {
	profiles, err := r.profileDirs(profile)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, selected := range profiles {
		if err := r.cleanAliasDir(selected); err != nil {
			return removed, err
		}
		nodeRemoved, err := r.cleanNodeReceipts(selected)
		if err != nil {
			return removed, err
		}
		removed += nodeRemoved
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

func (r Repository) cleanNodeReceipts(profile string) (int, error) {
	root := filepath.Join(r.root, "receipts", profile, "nodes")
	nodes, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
	}
	removed := 0
	for _, nodeEntry := range nodes {
		node := nodeEntry.Name()
		if !nodeEntry.IsDir() || !validNodeName(node) {
			continue
		}
		dir := filepath.Join(root, node)
		entries, err := os.ReadDir(dir)
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
			r.removeReportDirFor("sha256:"+hexID, node)
		}
		_ = os.Remove(dir)
	}
	_ = os.Remove(root)
	return removed, nil
}

func (r Repository) cleanAliasDir(profile string) error {
	dir := filepath.Join(r.root, "aliases", profile)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "check validation store permissions"}
	}
	for _, entry := range entries {
		if entry.IsDir() || !validTreeOID(entry.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return &Error{Code: CodeStoreError, Message: err.Error(), RequiredAction: "retry validation clean"}
		}
	}
	_ = os.Remove(dir)
	return nil
}

func (r Repository) readReceipt(profile, identity string) (Receipt, ReportBundle, error) {
	return r.readStoredReceipt(profile, "", identity)
}

func (r Repository) readNodeReceipt(profile, node, identity string) (Receipt, ReportBundle, error) {
	return r.readStoredReceipt(profile, node, identity)
}

func (r Repository) readStoredReceipt(profile, node, identity string) (Receipt, ReportBundle, error) {
	var receipt Receipt
	if !validProfile(profile) || !validDigest(identity) || node != "" && !validNodeName(node) {
		return receipt, ReportBundle{}, invalidReceipt("validation identity is invalid")
	}
	path := r.receiptPath(profile, identity)
	if node != "" {
		path = r.nodeReceiptPath(profile, node, identity)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return receipt, ReportBundle{}, err
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return Receipt{}, ReportBundle{}, invalidReceipt("Receipt JSON is invalid")
	}
	bundle, err := r.validateReceipt(receipt, identity, profile, node)
	if err != nil {
		return Receipt{}, ReportBundle{}, err
	}
	return receipt, bundle, nil
}

func (r Repository) validateReceipt(receipt Receipt, identity, profile, node string) (ReportBundle, error) {
	if receipt.SchemaVersion != receiptSchemaVersion || receipt.ValidationIdentity != identity || receipt.Fingerprint.Profile != profile || receipt.Fingerprint.Node != node || receipt.Fingerprint.RepositoryID != r.repositoryID || !validFingerprint(receipt.Fingerprint) || !validDigest(receipt.ResultsDigest) {
		return ReportBundle{}, invalidReceipt("Receipt identity or schema is invalid")
	}
	if receipt.ReceiptID != receiptDigest(receipt) {
		return ReportBundle{}, invalidReceipt("Receipt integrity check failed")
	}
	if receipt.Conclusion != "PASS" || !receipt.Reusable || !receipt.Scope.IgnoredFilesOutOfScope {
		return ReportBundle{}, invalidReceipt("Receipt is not reusable PASS evidence")
	}
	bundle, actual, err := r.readReportBundleFor(identity, node)
	if err != nil {
		return ReportBundle{}, invalidReceipt(err.Error())
	}
	if !sameArtifacts(actual, receipt.Reports) {
		return ReportBundle{}, invalidReceipt("retained report integrity check failed")
	}
	return bundle, nil
}

func (r Repository) writeReportDir(identity, node string, reports ReportBundle) ([]ReportArtifact, error) {
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
	finalDir := r.reportDirFor(identity, node)
	if artifacts, err := r.readReportArtifacts(identity, node); err == nil {
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
		if existing, readErr := r.readReportArtifacts(identity, node); readErr == nil {
			return existing, nil
		}
		return nil, &Error{Code: CodeStoreError, Message: fmt.Sprintf("publish report: %v", err), RequiredAction: "retry validation"}
	}
	return artifacts, nil
}

func (r Repository) readReportArtifacts(identity, node string) ([]ReportArtifact, error) {
	_, artifacts, err := r.readReportBundleFor(identity, node)
	return artifacts, err
}

func (r Repository) readReportBundleFor(identity, node string) (ReportBundle, []ReportArtifact, error) {
	dir := r.reportDirFor(identity, node)
	bundle := ReportBundle{}
	artifacts := make([]ReportArtifact, 0, len(reportNames))
	for _, name := range reportNames {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return ReportBundle{}, nil, err
		}
		switch name {
		case "results.json":
			bundle.ResultsJSON = content
		case "summary.json":
			bundle.SummaryJSON = content
		case "report.md":
			bundle.ReportMarkdown = content
		}
		artifacts = append(artifacts, ReportArtifact{Name: name, Digest: digestBytes(content)})
	}
	return bundle, artifacts, nil
}

func (r Repository) removeReportDir(identity string) {
	r.removeReportDirFor(identity, "")
}

func (r Repository) removeReportDirFor(identity, node string) {
	dir := r.reportDirFor(identity, node)
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

func (r Repository) nodeReceiptPath(profile, node, identity string) string {
	return filepath.Join(r.root, "receipts", profile, "nodes", node, digestHex(identity)+".json")
}

func (r Repository) reportDir(identity string) string {
	return filepath.Join(r.root, "reports", digestHex(identity))
}

func (r Repository) nodeReportDir(identity string) string {
	return filepath.Join(r.root, "node-reports", digestHex(identity))
}

func (r Repository) reportDirFor(identity, node string) string {
	if node != "" {
		return r.nodeReportDir(identity)
	}
	return r.reportDir(identity)
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
	if !validDigest(fingerprint.Identity) || !validDigest(fingerprint.RepositoryID) || !validProfile(fingerprint.Profile) {
		return false
	}
	if fingerprint.Node == "" {
		if !validTreeOID(fingerprint.SubjectTreeOID) || fingerprint.NodeInputDigest != "" {
			return false
		}
	} else if fingerprint.SubjectTreeOID != "" || !validNodeName(fingerprint.Node) || !validDigest(fingerprint.NodeInputDigest) {
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

func validNodeName(node string) bool {
	if len(node) < 1 || len(node) > 32 {
		return false
	}
	for index, char := range node {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' && index > 0 || (char == '-' || char == '_') && index > 0 {
			continue
		}
		return false
	}
	return true
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
	return &Error{Code: CodeReceiptInvalid, Message: message, RequiredAction: "clean the affected profile evidence, then rerun without reuse"}
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
