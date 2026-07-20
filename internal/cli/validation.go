package cli

import (
	"errors"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

type validationStatusData struct {
	Subject      validationevidence.Subject `json:"subject"`
	ReceiptCount int                        `json:"receiptCount"`
}

type validationCheckData struct {
	Profile          string                           `json:"profile"`
	Target           validationevidence.Target        `json:"target"`
	Subject          validationevidence.Subject       `json:"subject"`
	Fingerprint      validationevidence.Fingerprint   `json:"fingerprint"`
	Decision         validationevidence.ReuseDecision `json:"decision"`
	CommitAliasBound bool                             `json:"commitAliasBound,omitempty"`
}

type validationListData struct {
	Profile      string                       `json:"profile,omitempty"`
	ReceiptCount int                          `json:"receiptCount"`
	Receipts     []validationevidence.Receipt `json:"receipts"`
}

type validationCleanData struct {
	Profile         string `json:"profile,omitempty"`
	RemovedReceipts int    `json:"removedReceipts"`
}

type validationErrorData struct {
	Code           validationevidence.ErrorCode `json:"code"`
	Reason         string                       `json:"reason"`
	RequiredAction string                       `json:"requiredAction,omitempty"`
}

func runValidation(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("validation requires subcommand: status, check, list or clean")
	}
	switch strings.ToLower(args[0]) {
	case "status":
		return runValidationStatus(args[1:], start)
	case "check":
		return runValidationCheck(args[1:], start)
	case "list":
		return runValidationList(args[1:], start)
	case "clean":
		return runValidationClean(args[1:], start)
	default:
		return report.Result{}, usageErrorf("unsupported validation subcommand: %s", args[0])
	}
}

func runValidationStatus(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("validation status")
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	repo, store, err := openValidationRepository(*repoArg)
	if err != nil {
		return validationFailure("validation status", repo, start, "cannot open validation evidence", err)
	}
	subject, err := store.Capture(validationevidence.TargetAuto)
	if err != nil {
		return validationFailure("validation status", repo, start, "cannot capture validation subject", err)
	}
	receipts, err := store.List("")
	if err != nil {
		return validationFailure("validation status", repo, start, "cannot read validation evidence", err)
	}
	data := validationStatusData{Subject: subject, ReceiptCount: len(receipts)}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "validation status",
		OK:            true,
		Message:       "validation evidence status",
		RepoRoot:      repo,
		Data:          data,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func runValidationCheck(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("validation check")
	repoArg := fs.String("repo-root", "", "repository root")
	profileArg := fs.String("profile", "", "Smoke, Full or Release")
	targetArg := fs.String("target", "", "HEAD or INDEX")
	bindAliasArg := fs.Bool("bind-alias", false, "bind a matching HEAD Receipt to the current commit")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	profile, _, err := normalizeTestProfile(*profileArg)
	if err != nil {
		return report.Result{}, err
	}
	target, err := normalizeValidationTarget(*targetArg)
	if err != nil {
		return report.Result{}, err
	}
	if *bindAliasArg && target != validationevidence.TargetHead {
		return report.Result{}, usageErrorf("--bind-alias requires --target HEAD")
	}
	repo, store, err := openValidationRepository(*repoArg)
	if err != nil {
		return validationFailure("validation check", repo, start, "cannot open validation evidence", err)
	}
	subject, err := store.Capture(target)
	if err != nil {
		return validationFailure("validation check", repo, start, "cannot capture validation subject", err)
	}
	spec, err := testengine.EvidenceSpec(validationTestConfig(repo, profile))
	if err != nil {
		return validationFailure("validation check", repo, start, "cannot compute validation semantics", err)
	}
	fingerprint, err := store.Fingerprint(subject, spec)
	if err != nil {
		return validationFailure("validation check", repo, start, "cannot compute validation identity", err)
	}
	decision := store.Check(subject, fingerprint)
	aliasBound := false
	if decision.Hit && *bindAliasArg {
		if decision.Receipt == nil {
			return validationFailure("validation check", repo, start, "cannot bind validation alias", errors.New("matching Receipt is unavailable"))
		}
		if err := store.BindCommit("HEAD", *decision.Receipt); err != nil {
			return validationFailure("validation check", repo, start, "cannot bind validation alias", err)
		}
		aliasBound = true
	}
	data := validationCheckData{
		Profile: profile, Target: target, Subject: subject,
		Fingerprint: fingerprint, Decision: decision, CommitAliasBound: aliasBound,
	}
	errs := []string{}
	if !decision.Hit {
		errs = append(errs, string(decision.Code)+": "+decision.Reason)
	}
	result := report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "validation check",
		OK:            decision.Hit,
		Message:       "validation Receipt check",
		RepoRoot:      repo,
		InputDigest:   fingerprint.Identity,
		Data:          data,
		Errors:        errs,
		ElapsedMS:     report.Elapsed(start),
	}
	if !result.OK {
		result.ErrorKind = report.ErrorKindValidation
	}
	return result, report.BoolErr(errs)
}

func runValidationList(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("validation list")
	repoArg := fs.String("repo-root", "", "repository root")
	profileArg := fs.String("profile", "", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	profile, err := normalizeOptionalValidationProfile(*profileArg)
	if err != nil {
		return report.Result{}, err
	}
	repo, store, err := openValidationRepository(*repoArg)
	if err != nil {
		return validationFailure("validation list", repo, start, "cannot open validation evidence", err)
	}
	receipts, err := store.List(profile)
	if err != nil {
		return validationFailure("validation list", repo, start, "cannot list validation evidence", err)
	}
	data := validationListData{Profile: profile, ReceiptCount: len(receipts), Receipts: receipts}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "validation list",
		OK:            true,
		Message:       "validation Receipts",
		RepoRoot:      repo,
		Data:          data,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func runValidationClean(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("validation clean")
	repoArg := fs.String("repo-root", "", "repository root")
	profileArg := fs.String("profile", "", "Smoke, Full or Release")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	profile, err := normalizeOptionalValidationProfile(*profileArg)
	if err != nil {
		return report.Result{}, err
	}
	repo, store, err := openValidationRepository(*repoArg)
	if err != nil {
		return validationFailure("validation clean", repo, start, "cannot open validation evidence", err)
	}
	removed, err := store.Clean(profile)
	if err != nil {
		return validationFailure("validation clean", repo, start, "cannot clean validation evidence", err)
	}
	data := validationCleanData{Profile: profile, RemovedReceipts: removed}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "validation clean",
		OK:            true,
		Message:       "validation evidence cleaned",
		RepoRoot:      repo,
		Data:          data,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func openValidationRepository(repoArg string) (string, validationevidence.Repository, error) {
	repo, err := platform.ResolveRepoRoot(repoArg)
	if err != nil {
		return "", validationevidence.Repository{}, err
	}
	store, err := validationevidence.Open(repo)
	return repo, store, err
}

func validationTestConfig(repo, profile string) testengine.Config {
	return testengine.Config{
		Repo: repo, Profile: profile,
		Timeout: 180 * time.Second, LongTimeout: 600 * time.Second, Concurrency: 4,
		Reuse: testengine.ReuseOff, CommandCatalogDigest: commandCatalogEvidenceDigest,
	}
}

func normalizeValidationTarget(value string) (validationevidence.Target, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(validationevidence.TargetHead):
		return validationevidence.TargetHead, nil
	case string(validationevidence.TargetIndex):
		return validationevidence.TargetIndex, nil
	case "":
		return "", usageErrorf("validation check requires --target HEAD|INDEX")
	default:
		return "", usageErrorf("unsupported validation target: %s", value)
	}
}

func normalizeOptionalValidationProfile(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	profile, _, err := normalizeTestProfile(value)
	return profile, err
}

func validationFailure(command, repo string, start time.Time, message string, err error) (report.Result, error) {
	var evidenceErr *validationevidence.Error
	if !errors.As(err, &evidenceErr) {
		result := report.Fail(command, start, message, nil, err.Error())
		result.RepoRoot = repo
		return result, err
	}
	data := validationErrorData{
		Code: evidenceErr.Code, Reason: evidenceErr.Message, RequiredAction: evidenceErr.RequiredAction,
	}
	result := report.Fail(command, start, message, data, string(data.Code)+": "+data.Reason)
	result.RepoRoot = repo
	result.ErrorKind = report.ErrorKindValidation
	return result, report.BoolErr(result.Errors)
}
