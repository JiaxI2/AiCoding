package validationevidence

import (
	"fmt"
	"sync"
)

const (
	toolchainSchemaVersion = 1
	receiptSchemaVersion   = 2
)

// Target selects the Git candidate whose tree identifies validated content.
type Target string

const (
	TargetAuto  Target = "AUTO"
	TargetHead  Target = "HEAD"
	TargetIndex Target = "INDEX"
)

// SubjectMode describes how the validation subject was captured.
type SubjectMode string

const (
	SubjectHead  SubjectMode = "head"
	SubjectIndex SubjectMode = "index"
	SubjectDirty SubjectMode = "dirty"
)

// ErrorCode is a stable machine-readable validation-evidence outcome.
type ErrorCode string

const (
	CodeReceiptHit              ErrorCode = "VALIDATION_RECEIPT_HIT"
	CodeReceiptMiss             ErrorCode = "VALIDATION_RECEIPT_MISS"
	CodeReceiptInvalid          ErrorCode = "VALIDATION_RECEIPT_INVALID"
	CodeSubjectNotReusable      ErrorCode = "VALIDATION_SUBJECT_NOT_REUSABLE"
	CodeTargetNotFound          ErrorCode = "VALIDATION_TARGET_NOT_FOUND"
	CodeFingerprintInvalid      ErrorCode = "VALIDATION_FINGERPRINT_INVALID"
	CodeStoreError              ErrorCode = "VALIDATION_STORE_ERROR"
	CodeContentChangedDuringRun ErrorCode = "VALIDATION_CONTENT_CHANGED_DURING_RUN"
	CodeReuseAuditMismatch      ErrorCode = "VALIDATION_REUSE_AUDIT_MISMATCH"
	CodePolicyInvalid           ErrorCode = "VALIDATION_POLICY_INVALID"
	CodePushContextRejected     ErrorCode = "VALIDATION_PUSH_CONTEXT_REJECTED"
)

// Error carries a stable code and the action an Agent should take next.
type Error struct {
	Code           ErrorCode `json:"code"`
	Message        string    `json:"message"`
	RequiredAction string    `json:"requiredAction,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Scope states the deliberate boundary of Git-tree evidence.
type Scope struct {
	IgnoredFilesOutOfScope bool `json:"ignoredFilesOutOfScope"`
}

// Subject is one start/end Git content snapshot.
type Subject struct {
	TreeOID        string      `json:"treeOID"`
	Mode           SubjectMode `json:"mode"`
	Reusable       bool        `json:"reusable"`
	ReusableReason string      `json:"reusableReason,omitempty"`
	Scope          Scope       `json:"scope"`
}

// FingerprintSpec contains semantic inputs owned by the test engine and CLI.
type FingerprintSpec struct {
	Profile              string
	ValidationPlanDigest string
	EngineSemanticDigest string
	OptionsDigest        string
	ConfigPaths          []string
}

// Fingerprint is the full deterministic validation-identity payload.
type Fingerprint struct {
	Identity             string `json:"identity"`
	RepositoryID         string `json:"repositoryID"`
	SubjectTreeOID       string `json:"subjectTreeOID"`
	Profile              string `json:"profile"`
	ValidationPlanDigest string `json:"validationPlanDigest"`
	EngineSemanticDigest string `json:"engineSemanticDigest"`
	ConfigDigest         string `json:"configDigest"`
	ToolchainDigest      string `json:"toolchainDigest"`
	OptionsDigest        string `json:"optionsDigest"`
}

// ReportBundle is the minimal immutable evidence view retained with a Receipt.
type ReportBundle struct {
	ResultsJSON    []byte
	SummaryJSON    []byte
	ReportMarkdown []byte
}

// ReportArtifact binds one retained report file to its content digest.
type ReportArtifact struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

// Receipt is an immutable proof that one validation identity passed.
type Receipt struct {
	SchemaVersion      int              `json:"schemaVersion"`
	ReceiptID          string           `json:"receiptID"`
	ValidationIdentity string           `json:"validationIdentity"`
	Fingerprint        Fingerprint      `json:"fingerprint"`
	Conclusion         string           `json:"conclusion"`
	ResultsDigest      string           `json:"resultsDigest"`
	Reusable           bool             `json:"reusable"`
	ReusableReason     string           `json:"reusableReason,omitempty"`
	Scope              Scope            `json:"scope"`
	Reports            []ReportArtifact `json:"reports"`
}

// ReuseDecision is the fail-closed result of checking one exact identity.
type ReuseDecision struct {
	Hit             bool          `json:"hit"`
	Code            ErrorCode     `json:"code"`
	Reason          string        `json:"reason"`
	RequiredAction  string        `json:"requiredAction,omitempty"`
	CheckDurationMS int64         `json:"checkDurationMs"`
	Receipt         *Receipt      `json:"receipt,omitempty"`
	ReportBundle    *ReportBundle `json:"-"`
}

// Repository owns validation evidence under one Git common directory.
type Repository struct {
	repo         string
	commonDir    string
	root         string
	repositoryID string
	putMu        *sync.Mutex
}
