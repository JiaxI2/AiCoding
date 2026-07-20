package validationevidence

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

const validationPolicyPath = "config/validation-policy.json"

// Policy maps remote push contexts to the validation profile they require.
type Policy struct {
	SchemaVersion   int           `json:"schemaVersion"`
	UnmatchedAction string        `json:"unmatchedAction"`
	Contexts        []PushContext `json:"contexts"`
}

// PushContext is one ordered remote-ref rule in the Context Gate.
type PushContext struct {
	ID                 string `json:"id"`
	RemoteRef          string `json:"remoteRef,omitempty"`
	RemoteRefPrefix    string `json:"remoteRefPrefix,omitempty"`
	RequiredProfile    string `json:"requiredProfile"`
	RequireFastForward bool   `json:"requireFastForward"`
	AllowDelete        bool   `json:"allowDelete"`
}

// PushGateResult records the decision for one pre-push update.
type PushGateResult struct {
	LocalRef           string    `json:"localRef"`
	LocalOID           string    `json:"localOID"`
	RemoteRef          string    `json:"remoteRef"`
	RemoteOID          string    `json:"remoteOID"`
	ContextID          string    `json:"contextID,omitempty"`
	RequiredProfile    string    `json:"requiredProfile,omitempty"`
	SubjectTreeOID     string    `json:"subjectTreeOID,omitempty"`
	ValidationIdentity string    `json:"validationIdentity,omitempty"`
	ReceiptID          string    `json:"receiptID,omitempty"`
	Allowed            bool      `json:"allowed"`
	Code               ErrorCode `json:"code,omitempty"`
	Reason             string    `json:"reason"`
}

// PushGateReport keeps deterministic decisions plus an observable duration.
type PushGateReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	OK            bool             `json:"ok"`
	PolicyPath    string           `json:"policyPath"`
	Updates       []PushGateResult `json:"updates"`
	Required      int              `json:"required"`
	Bypassed      int              `json:"bypassed"`
	Errors        []string         `json:"errors"`
	DurationMS    int64            `json:"durationMs"`
}

// LoadPolicy reads and validates the repository's Context Gate policy.
func LoadPolicy(repo string) (Policy, error) {
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(validationPolicyPath)))
	if err != nil {
		return Policy{}, policyError("read validation policy: " + err.Error())
	}
	var policy Policy
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, policyError("parse validation policy: " + err.Error())
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Policy{}, policyError("parse validation policy: trailing JSON value")
	}
	if err := validatePolicy(policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

// BindCommit stores a profile-specific, mutable commit alias to an immutable
// Receipt. The commit tree must exactly match the Receipt subject tree. HEAD is
// accepted as the sole symbolic input so upper layers need no gitx dependency.
func (r Repository) BindCommit(commitOID string, receipt Receipt) error {
	if strings.EqualFold(strings.TrimSpace(commitOID), "HEAD") {
		resolved, err := gitx.HeadCommit(r.repo)
		if err != nil {
			return &Error{Code: CodeTargetNotFound, Message: err.Error(), RequiredAction: "verify that HEAD exists locally"}
		}
		commitOID = resolved
	}
	commitOID = strings.ToLower(strings.TrimSpace(commitOID))
	if !validTreeOID(commitOID) || !validFingerprint(receipt.Fingerprint) || receipt.ValidationIdentity != receipt.Fingerprint.Identity || receipt.Fingerprint.RepositoryID != r.repositoryID {
		return fingerprintError("commit alias identity is invalid")
	}
	stored, _, err := r.readReceipt(receipt.Fingerprint.Profile, receipt.ValidationIdentity)
	if err != nil {
		return invalidReceipt("commit alias Receipt is unavailable")
	}
	if receipt.ReceiptID != "" && stored.ReceiptID != receipt.ReceiptID {
		return invalidReceipt("commit alias Receipt does not match stored evidence")
	}
	treeOID, err := gitx.TreeOID(r.repo, commitOID)
	if err != nil {
		return &Error{Code: CodeTargetNotFound, Message: err.Error(), RequiredAction: "verify that the commit exists locally"}
	}
	if treeOID != stored.Fingerprint.SubjectTreeOID {
		return fingerprintError("commit tree does not match the Receipt subject tree")
	}
	content := []byte(stored.ValidationIdentity + "\n")
	path := r.aliasPath(stored.Fingerprint.Profile, commitOID)
	if existing, readErr := os.ReadFile(path); readErr == nil && bytes.Equal(existing, content) {
		return nil
	}
	if err := atomicWriteFile(path, content); err == nil {
		return nil
	}
	// Windows cannot rename over an existing file. Removing only this exact,
	// reproducible alias creates a fail-closed miss if the second publish fails.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return &Error{Code: CodeStoreError, Message: "replace commit alias: " + err.Error(), RequiredAction: "check validation store permissions"}
	}
	if err := atomicWriteFile(path, content); err != nil {
		return &Error{Code: CodeStoreError, Message: "write commit alias: " + err.Error(), RequiredAction: "check validation store permissions"}
	}
	return nil
}

// GatePush checks only policy-selected refs and requires an integrity-checked
// Receipt alias for the actual local object named by each push update.
func (r Repository) GatePush(policy Policy, updates []gitx.PushUpdate) (report PushGateReport) {
	start := time.Now()
	report = PushGateReport{SchemaVersion: 1, OK: true, PolicyPath: validationPolicyPath, Updates: make([]PushGateResult, 0, len(updates)), Errors: []string{}}
	defer func() { report.DurationMS = time.Since(start).Milliseconds() }()
	if err := validatePolicy(policy); err != nil {
		report.OK = false
		report.Errors = append(report.Errors, err.Error())
		return report
	}
	for _, update := range updates {
		result := r.gatePushUpdate(policy, update)
		report.Updates = append(report.Updates, result)
		if result.ContextID == "" {
			report.Bypassed++
		} else {
			report.Required++
		}
		if !result.Allowed {
			report.OK = false
			report.Errors = append(report.Errors, result.RemoteRef+": "+string(result.Code)+": "+result.Reason)
		}
	}
	return report
}

func (r Repository) gatePushUpdate(policy Policy, update gitx.PushUpdate) PushGateResult {
	result := PushGateResult{
		LocalRef: update.LocalRef, LocalOID: update.LocalOID, RemoteRef: update.RemoteRef, RemoteOID: update.RemoteOID,
	}
	context, matched := matchPushContext(policy, update.RemoteRef)
	if !matched {
		result.Allowed = policy.UnmatchedAction == "allow"
		if result.Allowed {
			result.Reason = "remote ref is outside the validation policy"
			return result
		}
		result.Code = CodePushContextRejected
		result.Reason = "validation policy rejects unmatched remote refs"
		return result
	}
	result.ContextID = context.ID
	result.RequiredProfile = context.RequiredProfile
	if zeroObjectID(update.LocalOID) {
		result.Allowed = context.AllowDelete
		if result.Allowed {
			result.Reason = "policy allows deletion for this context"
			return result
		}
		result.Code = CodePushContextRejected
		result.Reason = "policy forbids deletion for this context"
		return result
	}

	var treeOID string
	var treeErr error
	ancestor := true
	var ancestorErr error
	checkAncestor := context.RequireFastForward && !zeroObjectID(update.RemoteOID)
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		treeOID, treeErr = gitx.TreeOID(r.repo, update.LocalOID)
	}()
	if checkAncestor {
		wait.Add(1)
		go func() {
			defer wait.Done()
			ancestor, ancestorErr = gitx.IsAncestor(r.repo, update.RemoteOID, update.LocalOID)
		}()
	}
	wait.Wait()
	if treeErr != nil {
		result.Code = CodeTargetNotFound
		result.Reason = treeErr.Error()
		return result
	}
	result.SubjectTreeOID = treeOID
	if ancestorErr != nil {
		result.Code = CodePushContextRejected
		result.Reason = "cannot verify fast-forward ancestry: " + ancestorErr.Error()
		return result
	}
	if !ancestor {
		result.Code = CodePushContextRejected
		result.Reason = "policy requires a fast-forward update"
		return result
	}

	decision := r.checkCommitAlias(context.RequiredProfile, update.LocalOID, treeOID)
	result.Code = decision.Code
	result.Reason = decision.Reason
	result.Allowed = decision.Hit
	if decision.Receipt != nil {
		result.ValidationIdentity = decision.Receipt.ValidationIdentity
		result.ReceiptID = decision.Receipt.ReceiptID
	}
	return result
}

func (r Repository) checkCommitAlias(profile, commitOID, treeOID string) (decision ReuseDecision) {
	start := time.Now()
	decision = ReuseDecision{Code: CodeReceiptMiss, Reason: "no commit alias exists for the required profile", RequiredAction: "run the required validation profile on the exact commit"}
	defer func() { decision.CheckDurationMS = time.Since(start).Milliseconds() }()
	raw, err := os.ReadFile(r.aliasPath(profile, commitOID))
	if err != nil {
		if !os.IsNotExist(err) {
			decision.Code = CodeStoreError
			decision.Reason = err.Error()
			decision.RequiredAction = "check validation store permissions"
		}
		return decision
	}
	identity := strings.TrimSpace(string(raw))
	if !validDigest(identity) {
		decision.Code = CodeReceiptInvalid
		decision.Reason = "commit alias identity is invalid"
		decision.RequiredAction = "rerun the required validation profile"
		return decision
	}
	receipt, bundle, err := r.readReceipt(profile, identity)
	if err != nil {
		decision.Code = CodeReceiptInvalid
		decision.Reason = "commit alias Receipt integrity validation failed: " + err.Error()
		decision.RequiredAction = "rerun the required validation profile"
		return decision
	}
	if receipt.Fingerprint.SubjectTreeOID != treeOID {
		decision.Code = CodeReceiptInvalid
		decision.Reason = "commit alias Receipt is bound to a different tree"
		decision.RequiredAction = "rerun the required validation profile on the exact commit"
		return decision
	}
	decision.Hit = true
	decision.Code = CodeReceiptHit
	decision.Reason = "required profile Receipt matches the pushed commit tree"
	decision.RequiredAction = ""
	decision.Receipt = &receipt
	decision.ReportBundle = &bundle
	return decision
}

func (r Repository) aliasPath(profile, commitOID string) string {
	return filepath.Join(r.root, "aliases", profile, commitOID)
}

func validatePolicy(policy Policy) error {
	if policy.SchemaVersion != 1 {
		return policyError("unsupported validation policy schemaVersion")
	}
	if policy.UnmatchedAction != "allow" && policy.UnmatchedAction != "deny" {
		return policyError("unmatchedAction must be allow or deny")
	}
	if len(policy.Contexts) == 0 {
		return policyError("validation policy has no contexts")
	}
	seen := make(map[string]struct{}, len(policy.Contexts))
	for _, context := range policy.Contexts {
		if strings.TrimSpace(context.ID) == "" {
			return policyError("validation context id is required")
		}
		if _, exists := seen[context.ID]; exists {
			return policyError("duplicate validation context id: " + context.ID)
		}
		seen[context.ID] = struct{}{}
		if (context.RemoteRef == "") == (context.RemoteRefPrefix == "") {
			return policyError("validation context " + context.ID + " must set exactly one remote ref selector")
		}
		selector := context.RemoteRef
		if selector == "" {
			selector = context.RemoteRefPrefix
		}
		if !strings.HasPrefix(selector, "refs/") {
			return policyError("validation context " + context.ID + " has an invalid remote ref selector")
		}
		if context.RequiredProfile != "smoke" && context.RequiredProfile != "full" && context.RequiredProfile != "release" {
			return policyError("validation context " + context.ID + " has an invalid requiredProfile")
		}
	}
	return nil
}

func matchPushContext(policy Policy, remoteRef string) (PushContext, bool) {
	for _, context := range policy.Contexts {
		if context.RemoteRef != "" && remoteRef == context.RemoteRef || context.RemoteRefPrefix != "" && strings.HasPrefix(remoteRef, context.RemoteRefPrefix) {
			return context, true
		}
	}
	return PushContext{}, false
}

func zeroObjectID(value string) bool {
	value = strings.TrimSpace(value)
	return (len(value) == 40 || len(value) == 64) && strings.Trim(value, "0") == ""
}

func policyError(message string) *Error {
	return &Error{Code: CodePolicyInvalid, Message: message, RequiredAction: "fix config/validation-policy.json before pushing"}
}
