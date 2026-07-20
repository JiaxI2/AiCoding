package validationevidence

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Check performs an O(1) exact-path lookup and validates Receipt and report
// integrity. Any corruption is a miss, never a reusable hit.
func (r Repository) Check(subject Subject, fingerprint Fingerprint) (decision ReuseDecision) {
	start := time.Now()
	decision = ReuseDecision{Hit: false, Code: CodeReceiptMiss, Reason: "no reusable Receipt exists", RequiredAction: "run the selected validation profile"}
	defer func() { decision.CheckDurationMS = time.Since(start).Milliseconds() }()
	if !subject.Reusable {
		decision.Code = CodeSubjectNotReusable
		decision.Reason = subject.ReusableReason
		decision.RequiredAction = "use a clean HEAD or index-only subject, or run without reuse"
		return decision
	}
	if subject.TreeOID != fingerprint.SubjectTreeOID || fingerprint.RepositoryID != r.repositoryID || !validFingerprint(fingerprint) {
		decision.Code = CodeFingerprintInvalid
		decision.Reason = "subject and fingerprint do not match"
		decision.RequiredAction = "recompute the validation identity"
		return decision
	}
	path := r.receiptPath(fingerprint.Profile, fingerprint.Identity)
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			decision.Code = CodeStoreError
			decision.Reason = err.Error()
			decision.RequiredAction = "check validation store permissions"
		}
		return decision
	}
	receipt, bundle, err := r.readReceipt(fingerprint.Profile, fingerprint.Identity)
	if err != nil {
		decision.Code = CodeReceiptInvalid
		decision.Reason = fmt.Sprintf("Receipt integrity validation failed: %v", err)
		decision.RequiredAction = "rerun the selected validation profile without reuse"
		if errors.Is(err, os.ErrNotExist) {
			decision.Code = CodeReceiptMiss
			decision.Reason = "Receipt disappeared during lookup"
		}
		return decision
	}
	decision.Hit = true
	decision.Code = CodeReceiptHit
	decision.Reason = "reusable PASS Receipt matches the validation identity"
	decision.RequiredAction = ""
	decision.Receipt = &receipt
	decision.ReportBundle = &bundle
	return decision
}
