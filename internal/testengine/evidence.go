package testengine

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/registry"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

const evidenceImplVersion = 4

var executeTestCases = runAll

var captureValidationSubject = func(repository validationevidence.Repository) (validationevidence.Subject, error) {
	return repository.Capture(validationevidence.TargetAuto)
}

type normalizedTestCase struct {
	ID                 string   `json:"id"`
	Kind               string   `json:"kind"`
	Command            []string `json:"command,omitempty"`
	Severity           Severity `json:"severity"`
	TimeoutKind        string   `json:"timeoutKind,omitempty"`
	ExpectJSON         bool     `json:"expectJSON,omitempty"`
	OptionalPath       string   `json:"optionalPath,omitempty"`
	NetworkFailureWarn bool     `json:"networkFailureWarn,omitempty"`
}

type evidenceOptions struct {
	TimeoutMS     int64 `json:"timeoutMs"`
	LongTimeoutMS int64 `json:"longTimeoutMs"`
	Concurrency   int   `json:"concurrency"`
	Strict        bool  `json:"strict"`
	IncludeMutate bool  `json:"includeMutate"`
	NoJSONCheck   bool  `json:"noJSONCheck"`
	AllowDirty    bool  `json:"allowDirty"`
}

// EvidenceSpec returns the deterministic semantic inputs shared by test and
// validation check. Reuse-control flags are intentionally excluded.
func EvidenceSpec(cfg Config) (validationevidence.FingerprintSpec, error) {
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		return validationevidence.FingerprintSpec{}, err
	}
	planDigest, err := RegistryDigest(normalized)
	if err != nil {
		return validationevidence.FingerprintSpec{}, err
	}
	catalogDigest := strings.TrimSpace(normalized.CommandCatalogDigest)
	if catalogDigest == "" {
		fallback, snapshotErr := registry.NewSnapshot("command-catalog", struct {
			Entrypoint string `json:"entrypoint"`
		}{Entrypoint: "testengine-direct"})
		if snapshotErr != nil {
			return validationevidence.FingerprintSpec{}, snapshotErr
		}
		catalogDigest = fallback.Digest()
	}
	if !validSHA256Digest(catalogDigest) {
		return validationevidence.FingerprintSpec{}, fmt.Errorf("invalid command catalog digest")
	}
	engineDigest, err := engineSemanticDigest(catalogDigest, planDigest, evidenceImplVersion)
	if err != nil {
		return validationevidence.FingerprintSpec{}, err
	}
	optionsSnapshot, err := registry.NewSnapshot("validation-options", evidenceOptions{
		TimeoutMS: normalized.Timeout.Milliseconds(), LongTimeoutMS: normalized.LongTimeout.Milliseconds(),
		Concurrency: normalized.Concurrency, Strict: normalized.Strict, IncludeMutate: normalized.IncludeMutate,
		NoJSONCheck: normalized.NoJSONCheck, AllowDirty: normalized.AllowDirty,
	})
	if err != nil {
		return validationevidence.FingerprintSpec{}, err
	}
	return validationevidence.FingerprintSpec{
		Profile: normalized.Profile, ValidationPlanDigest: planDigest, EngineSemanticDigest: engineDigest,
		OptionsDigest: optionsSnapshot.Digest(), ConfigPaths: evidenceConfigPaths(normalized.Profile),
	}, nil
}

// RegistryDigest hashes only TestCases selected by the normalized profile and
// removes worktree-specific absolute command paths.
func RegistryDigest(cfg Config) (string, error) {
	selected := make([]normalizedTestCase, 0)
	for _, testCase := range Registry(cfg) {
		if !profileEnabled(testCase, cfg.Profile) {
			continue
		}
		command := append([]string(nil), testCase.Command...)
		for index := range command {
			command[index] = normalizeEvidenceCommand(cfg.Repo, command[index])
		}
		selected = append(selected, normalizedTestCase{
			ID: testCase.ID, Kind: testCase.Kind, Command: command, Severity: testCase.Severity,
			TimeoutKind: testCase.TimeoutKind, ExpectJSON: testCase.ExpectJSON, OptionalPath: filepath.ToSlash(testCase.OptionalPath),
			NetworkFailureWarn: testCase.NetworkFailureWarn,
		})
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	snapshot, err := registry.NewSnapshot("validation-plan-"+cfg.Profile, selected)
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

func engineSemanticDigest(catalogDigest, registryDigest string, implVersion int) (string, error) {
	snapshot, err := registry.NewSnapshot("validation-engine-semantics", struct {
		CommandCatalogDigest string `json:"commandCatalogDigest"`
		RegistryDigest       string `json:"registryDigest"`
		ImplVersion          int    `json:"implVersion"`
	}{catalogDigest, registryDigest, implVersion})
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

func evidenceConfigPaths(profile Profile) []string {
	paths := []string{
		".github/repository-governance.toml",
		"config/codex-kit.json",
		"config/dependency-governance.json",
		"config/kit-registry.json",
		"config/mcp-registry.json",
		"config/repository-navigation.json",
	}
	if profile == ProfileRelease {
		paths = append(paths, ".github/RELEASE_TEMPLATE.md")
	}
	return paths
}

func normalizeEvidenceCommand(repo, value string) string {
	cleanRepo := filepath.Clean(repo)
	cleanValue := filepath.Clean(value)
	if strings.EqualFold(cleanValue, cleanRepo) {
		return "<repo>"
	}
	prefix := cleanRepo + string(filepath.Separator)
	if strings.HasPrefix(strings.ToLower(cleanValue), strings.ToLower(prefix)) {
		rel, err := filepath.Rel(cleanRepo, cleanValue)
		if err == nil {
			return "<repo>/" + filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(value)
}

func finalizeEvidence(
	ctx context.Context,
	cfg Config,
	testCases []TestCase,
	store validationevidence.Repository,
	startSubject validationevidence.Subject,
	startFingerprint validationevidence.Fingerprint,
	decision validationevidence.ReuseDecision,
	report *Report,
) {
	if report.ValidationCode == validationevidence.CodeReuseAuditMismatch {
		report.ReusableReason = "executed per-case statuses do not match the reusable Receipt"
		return
	}
	if ctx.Err() != nil {
		report.ReusableReason = ctx.Err().Error()
		return
	}
	endSubject, err := captureValidationSubject(store)
	if err != nil {
		report.ValidationCode = validationevidence.CodeContentChangedDuringRun
		report.ReusableReason = "cannot capture end validation identity: " + err.Error()
		return
	}
	endSpec, err := EvidenceSpec(cfg)
	if err != nil {
		report.ValidationCode = validationevidence.CodeContentChangedDuringRun
		report.ReusableReason = "cannot compute end validation identity: " + err.Error()
		return
	}
	endFingerprint, err := store.Fingerprint(endSubject, endSpec)
	if err != nil {
		report.ValidationCode = validationevidence.CodeContentChangedDuringRun
		report.ReusableReason = "cannot compute end validation identity: " + err.Error()
		return
	}
	if startFingerprint.Identity != endFingerprint.Identity {
		report.ValidationCode = validationevidence.CodeContentChangedDuringRun
		report.ReusableReason = "validation content or semantics changed during execution"
		return
	}
	reusable, reason, code := receiptEligible(cfg, testCases, report.Results, startSubject)
	if !reusable {
		report.ValidationCode = code
		report.ReusableReason = reason
		return
	}
	report.Reusable = true
	if decision.Hit {
		report.ValidationCode = validationevidence.CodeReceiptHit
	}
	if err := Write(cfg.Out, *report); err != nil {
		report.Reusable = false
		report.ValidationCode = validationevidence.CodeStoreError
		report.ReusableReason = err.Error()
		return
	}
	bundle, err := loadEvidenceBundle(cfg.Out)
	if err != nil {
		report.Reusable = false
		report.ValidationCode = validationevidence.CodeStoreError
		report.ReusableReason = err.Error()
		return
	}
	receipt, err := store.Put(validationevidence.Receipt{
		ValidationIdentity: startFingerprint.Identity,
		Fingerprint:        startFingerprint,
		Conclusion:         "PASS",
		ResultsDigest:      report.ResultsDigest,
		Reusable:           true,
		Scope:              startSubject.Scope,
	}, bundle)
	if err != nil {
		report.Reusable = false
		var evidenceError *validationevidence.Error
		if errors.As(err, &evidenceError) {
			report.ValidationCode = evidenceError.Code
		} else {
			report.ValidationCode = validationevidence.CodeStoreError
		}
		report.ReusableReason = err.Error()
		return
	}
	report.ReceiptID = receipt.ReceiptID
	if startSubject.Mode == validationevidence.SubjectHead {
		if bindErr := store.BindCommit("HEAD", receipt); bindErr != nil {
			report.ValidationCode = validationevidence.CodeStoreError
			report.ReusableReason = "Receipt created but HEAD alias is unavailable: " + bindErr.Error()
		}
	}
}

func receiptEligible(cfg Config, testCases []TestCase, results []Result, subject validationevidence.Subject) (bool, string, validationevidence.ErrorCode) {
	if !subject.Reusable {
		return false, subject.ReusableReason, validationevidence.CodeSubjectNotReusable
	}
	resultByID := make(map[string]Result, len(results))
	for _, result := range results {
		resultByID[result.ID] = result
		if result.Status == Fail {
			return false, "at least one validation case failed", ""
		}
	}
	for _, testCase := range testCases {
		if !profileEnabled(testCase, cfg.Profile) {
			continue
		}
		result, exists := resultByID[testCase.ID]
		if !exists {
			return false, "selected validation case has no result: " + testCase.ID, ""
		}
		if testCase.Severity == Required && result.Status != Pass {
			return false, "required validation case did not pass: " + testCase.ID, ""
		}
		if result.Status == Skip && testCase.OptionalPath == "" {
			return false, "selected validation case was unexpectedly skipped: " + testCase.ID, ""
		}
	}
	return true, "", ""
}

func reusedReport(cfg Config, testCases []TestCase, subject validationevidence.Subject, fingerprint validationevidence.Fingerprint, decision validationevidence.ReuseDecision) (Report, error) {
	if decision.Receipt == nil || decision.ReportBundle == nil {
		return Report{}, fmt.Errorf("matching Receipt has no retained report")
	}
	var report Report
	if err := json.Unmarshal(decision.ReportBundle.ResultsJSON, &report); err != nil {
		return Report{}, fmt.Errorf("decode retained report: %w", err)
	}
	actualResultsDigest, err := resultStatusDigest(cfg, testCases, report.Results)
	if err != nil {
		return Report{}, fmt.Errorf("digest retained result statuses: %w", err)
	}
	if report.ResultsDigest != actualResultsDigest || decision.Receipt.ResultsDigest != actualResultsDigest {
		return Report{}, fmt.Errorf("retained per-case statuses do not match Receipt")
	}
	for index := range report.Results {
		report.Results[index].StdoutFile = ""
		report.Results[index].StderrFile = ""
		report.Results[index].MetaFile = ""
	}
	report.ExecutionMode = "reused"
	report.ReceiptID = decision.Receipt.ReceiptID
	report.ValidationIdentity = fingerprint.Identity
	report.SubjectTreeOID = subject.TreeOID
	report.SubjectMode = subject.Mode
	report.Reusable = true
	report.ReusableReason = ""
	report.ValidationCode = validationevidence.CodeReceiptHit
	report.CheckDurationMS = decision.CheckDurationMS
	cacheHitRatio := 1.0
	report.Summary.CacheHitRatio = &cacheHitRatio
	report.Summary.ReceiptInvalidReason = ""
	if err := os.MkdirAll(cfg.Out, 0o755); err != nil {
		return Report{}, err
	}
	if err := Write(cfg.Out, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func resultStatusDigest(cfg Config, testCases []TestCase, results []Result) (string, error) {
	selected := make(map[string]struct{})
	for _, testCase := range testCases {
		if profileEnabled(testCase, cfg.Profile) {
			selected[testCase.ID] = struct{}{}
		}
	}
	type resultStatus struct {
		ID     string `json:"id"`
		Status Status `json:"status"`
	}
	statuses := make([]resultStatus, 0, len(selected))
	for _, result := range results {
		if _, ok := selected[result.ID]; ok {
			statuses = append(statuses, resultStatus{ID: result.ID, Status: result.Status})
		}
	}
	sort.Slice(statuses, func(i, j int) bool {
		if statuses[i].ID == statuses[j].ID {
			return statuses[i].Status < statuses[j].Status
		}
		return statuses[i].ID < statuses[j].ID
	})
	snapshot, err := registry.NewSnapshot("validation-result-statuses", struct {
		Profile  Profile        `json:"profile"`
		Statuses []resultStatus `json:"statuses"`
	}{Profile: cfg.Profile, Statuses: statuses})
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

func loadEvidenceBundle(outDir string) (validationevidence.ReportBundle, error) {
	read := func(name string) ([]byte, error) {
		content, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			return nil, fmt.Errorf("read %s for Receipt: %w", name, err)
		}
		return content, nil
	}
	results, err := read("results.json")
	if err != nil {
		return validationevidence.ReportBundle{}, err
	}
	summary, err := read("summary.json")
	if err != nil {
		return validationevidence.ReportBundle{}, err
	}
	markdown, err := read("report.md")
	if err != nil {
		return validationevidence.ReportBundle{}, err
	}
	return validationevidence.ReportBundle{ResultsJSON: results, SummaryJSON: summary, ReportMarkdown: markdown}, nil
}

func evidenceConclusion(report Report) string {
	if report.Summary.Fail > 0 || report.Summary.Conclusion == "FAIL" {
		return "FAIL"
	}
	return "PASS"
}

func validSHA256Digest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
