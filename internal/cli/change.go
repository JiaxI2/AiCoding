package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

const impactPolicyPath = "config/impact-policy.json"

type changeImpactRule = testengine.ChangeImpactRule
type changeImpactPolicy = testengine.ChangeImpactPolicy
type changeImpactMatch = testengine.ChangeImpactMatch

type changeVerifyStep struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	DurationMS int64  `json:"durationMs"`
	Detail     string `json:"detail"`
}

type changeReceiptDecision struct {
	Hit             bool                           `json:"hit"`
	Code            validationevidence.ErrorCode   `json:"code"`
	Reason          string                         `json:"reason"`
	CheckDurationMS int64                          `json:"checkDurationMs"`
	ReceiptID       string                         `json:"receiptID,omitempty"`
	Subject         validationevidence.Subject     `json:"subject"`
	Fingerprint     validationevidence.Fingerprint `json:"fingerprint"`
}

type changeVerifyData struct {
	Mode          string                    `json:"mode"`
	Since         string                    `json:"since,omitempty"`
	PolicyPath    string                    `json:"policyPath"`
	Paths         []string                  `json:"paths"`
	Matches       []changeImpactMatch       `json:"matches"`
	ChosenProfile string                    `json:"chosenProfile"`
	Target        validationevidence.Target `json:"target"`
	ExecutionMode string                    `json:"executionMode"`
	ExecutedCases int                       `json:"executedCases"`
	ReusedCases   int                       `json:"reusedCases"`
	Receipt       changeReceiptDecision     `json:"receipt"`
	Steps         []changeVerifyStep        `json:"steps"`
	TestReport    *testengine.Report        `json:"testReport,omitempty"`
}

type detectedChanges struct {
	Mode       string
	Since      string
	Paths      []string
	Target     validationevidence.Target
	Subject    validationevidence.Subject
	AllowDirty bool
}

type changeReceiptProbe struct {
	Subject     validationevidence.Subject
	Fingerprint validationevidence.Fingerprint
	Decision    validationevidence.ReuseDecision
}

var probeChangeReceipt = inspectChangeReceipt

func runChange(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("change requires subcommand: verify")
	}
	if _, err := resolveCatalogSubcommandID(CommandChange, args[0]); err != nil {
		return report.Result{}, err
	}
	fs := newFlagSet("change verify")
	repoArg := fs.String("repo-root", "", "repository root")
	stagedArg := fs.Bool("staged", false, "verify the staged Git index")
	sinceArg := fs.String("since", "", "verify committed changes since revision")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	if *stagedArg && strings.TrimSpace(*sinceArg) != "" {
		return report.Result{}, usageErrorf("change verify accepts either --staged or --since, not both")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return changeFailure(start, "", "cannot resolve repo root", nil, report.CategoryUsage, "aicoding change verify --help", err)
	}

	data := changeVerifyData{PolicyPath: impactPolicyPath, Paths: []string{}, Matches: []changeImpactMatch{}, Steps: []changeVerifyStep{}}
	detectStarted := time.Now()
	changes, err := detectChanges(repo, *stagedArg, *sinceArg)
	if err != nil {
		data.Steps = append(data.Steps, changeVerifyStep{Name: "changes.detect", OK: false, DurationMS: time.Since(detectStarted).Milliseconds(), Detail: err.Error()})
		var inputErr *changeInputError
		if errors.As(err, &inputErr) {
			return changeFailure(start, repo, "cannot select change verification subject", data, inputErr.Category, inputErr.NextAction, err)
		}
		return changeFailure(start, repo, "cannot detect repository changes", data, report.CategoryInternal, "git status --short", err)
	}
	data.Mode, data.Since, data.Paths, data.Target = changes.Mode, changes.Since, changes.Paths, changes.Target
	data.Steps = append(data.Steps, changeVerifyStep{Name: "changes.detect", OK: true, DurationMS: time.Since(detectStarted).Milliseconds(), Detail: fmt.Sprintf("%s selected %d path(s)", changes.Mode, len(changes.Paths))})

	impactStarted := time.Now()
	policy, err := loadChangeImpactPolicy(repo)
	if err != nil {
		data.Steps = append(data.Steps, changeVerifyStep{Name: "impact.select", OK: false, DurationMS: time.Since(impactStarted).Milliseconds(), Detail: err.Error()})
		return changeFailure(start, repo, "cannot load change impact policy", data, report.CategoryInternal, "aicoding governance lint --json", err)
	}
	profile, matches, err := selectChangeProfile(policy, changes.Paths)
	if err != nil {
		data.Steps = append(data.Steps, changeVerifyStep{Name: "impact.select", OK: false, DurationMS: time.Since(impactStarted).Milliseconds(), Detail: err.Error()})
		return changeFailure(start, repo, "cannot classify change impact", data, report.CategoryInternal, "aicoding governance lint --json", err)
	}
	data.ChosenProfile, data.Matches = displayTestProfile(profile), matches
	data.Steps = append(data.Steps, changeVerifyStep{Name: "impact.select", OK: true, DurationMS: time.Since(impactStarted).Milliseconds(), Detail: "selected " + data.ChosenProfile})

	receiptStarted := time.Now()
	probe, err := probeChangeReceipt(repo, profile, changes.Subject)
	if err != nil {
		data.Steps = append(data.Steps, changeVerifyStep{Name: "receipt.check", OK: false, DurationMS: time.Since(receiptStarted).Milliseconds(), Detail: err.Error()})
		return changeFailure(start, repo, "cannot inspect validation evidence", data, report.CategoryInternal, "aicoding validation status --json", err)
	}
	data.Receipt = changeReceiptView(probe.Subject, probe.Fingerprint, probe.Decision)
	data.Steps = append(data.Steps, changeVerifyStep{Name: "receipt.check", OK: true, DurationMS: time.Since(receiptStarted).Milliseconds(), Detail: string(probe.Decision.Code)})
	if probe.Decision.Hit {
		data.ExecutionMode = "receipt-hit"
		return report.Result{
			SchemaVersion: report.SchemaVersion, Command: "change verify", OK: true,
			Message: "change verification satisfied by exact Receipt", RepoRoot: repo,
			InputDigest: probe.Fingerprint.Identity, Data: data, ElapsedMS: report.Elapsed(start),
		}, nil
	}

	runStarted := time.Now()
	outDir := filepath.Join(repo, "test-results", "aicoding-change-verify-"+time.Now().Format("20060102-150405.000000000"))
	reuseMode := testengine.ReuseAuto
	if changes.AllowDirty {
		reuseMode = testengine.ReuseOff
	}
	cfg := testengine.Config{
		Repo: repo, Out: outDir, Profile: profile,
		Timeout: 180 * time.Second, LongTimeout: 600 * time.Second, Concurrency: 4,
		Reuse: reuseMode, AllowDirty: changes.AllowDirty, CommandCatalogDigest: commandCatalogEvidenceDigest,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	fileReport, runErr := runTestEngine(ctx, cfg)
	data.TestReport = &fileReport
	data.ExecutionMode = fileReport.ExecutionMode
	data.ExecutedCases, data.ReusedCases = changeCaseCounts(fileReport.Results)
	ok := runErr == nil && ctx.Err() != context.DeadlineExceeded && testengine.ExitCode(fileReport, nil) == 0
	detail := fileReport.Summary.Conclusion
	if runErr != nil {
		detail = runErr.Error()
	}
	data.Steps = append(data.Steps, changeVerifyStep{Name: "test.run", OK: ok, DurationMS: time.Since(runStarted).Milliseconds(), Detail: detail})
	result := report.Result{
		SchemaVersion: report.SchemaVersion, Command: "change verify", OK: ok,
		Message: "deterministic change impact verification", RepoRoot: repo,
		InputDigest: probe.Fingerprint.Identity, Data: data, ElapsedMS: report.Elapsed(start),
	}
	if ok {
		return result, nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.Errors = []string{"change verification timed out"}
		result = report.WithDecision(result, report.CategoryTransient, changeCommand(changes))
		return result, errors.New(result.Errors[0])
	}
	if runErr != nil {
		result.Errors = []string{runErr.Error()}
		result = report.WithDecision(result, report.CategoryInternal, "aicoding doctor --all --json")
		return result, runErr
	}
	result.Errors = []string{"selected " + data.ChosenProfile + " profile failed"}
	result.ErrorKind = report.ErrorKindValidation
	result = report.WithDecision(result, report.CategoryValidation, changeCommand(changes))
	return result, report.BoolErr(result.Errors)
}

type changeInputError struct {
	Message    string
	Category   report.Category
	NextAction string
}

func (e *changeInputError) Error() string { return e.Message }

func detectChanges(repo string, staged bool, since string) (detectedChanges, error) {
	status, err := gitx.StatusSnapshot(repo)
	if err != nil {
		return detectedChanges{}, err
	}
	since = strings.TrimSpace(since)
	if staged {
		if len(status.StagedPaths) == 0 {
			return detectedChanges{}, &changeInputError{Message: "change verify --staged requires staged paths", Category: report.CategoryUsage, NextAction: "git status --short"}
		}
		if status.TrackedModified || status.Untracked || status.SubmoduleDirty || status.Unmerged {
			return detectedChanges{}, &changeInputError{Message: "staged verification rejects worktree-only, untracked, submodule, or unmerged changes", Category: report.CategoryValidation, NextAction: "git status --short"}
		}
		subject, err := changeIndexSubject(repo)
		if err != nil {
			return detectedChanges{}, err
		}
		return detectedChanges{Mode: "staged", Paths: status.StagedPaths, Target: validationevidence.TargetIndex, Subject: subject}, nil
	}
	if since != "" {
		if len(status.Paths) != 0 || status.Unmerged || status.SubmoduleDirty {
			return detectedChanges{}, &changeInputError{Message: "change verify --since requires a clean HEAD", Category: report.CategoryValidation, NextAction: "git status --short"}
		}
		fromTree, err := gitx.TreeOID(repo, since)
		if err != nil {
			return detectedChanges{}, &changeInputError{Message: "cannot resolve --since revision: " + err.Error(), Category: report.CategoryUsage, NextAction: "git rev-parse --verify " + since}
		}
		toTree, err := gitx.TreeOID(repo, "HEAD")
		if err != nil {
			return detectedChanges{}, err
		}
		paths, err := gitx.DiffTreeFiles(repo, fromTree, toTree)
		if err != nil {
			return detectedChanges{}, err
		}
		subject, err := changeHeadSubject(repo)
		if err != nil {
			return detectedChanges{}, err
		}
		return detectedChanges{Mode: "since", Since: since, Paths: paths, Target: validationevidence.TargetHead, Subject: subject}, nil
	}
	if len(status.Paths) != 0 {
		target := validationevidence.TargetAuto
		allowDirty := status.TrackedModified || status.Untracked || status.SubmoduleDirty || status.Unmerged
		var subject validationevidence.Subject
		if status.Staged && !allowDirty {
			target = validationevidence.TargetIndex
			subject, err = changeIndexSubject(repo)
		} else {
			subject, err = changeDirtySubject(repo, status)
		}
		if err != nil {
			return detectedChanges{}, err
		}
		return detectedChanges{Mode: "worktree", Paths: status.Paths, Target: target, Subject: subject, AllowDirty: allowDirty}, nil
	}
	paths, err := gitx.CommitFiles(repo, "HEAD")
	if err != nil {
		return detectedChanges{}, err
	}
	subject, err := changeHeadSubject(repo)
	if err != nil {
		return detectedChanges{}, err
	}
	return detectedChanges{Mode: "head", Paths: paths, Target: validationevidence.TargetHead, Subject: subject}, nil
}

func changeIndexSubject(repo string) (validationevidence.Subject, error) {
	treeOID, err := gitx.WriteTree(repo)
	if err != nil {
		return validationevidence.Subject{}, err
	}
	return validationevidence.Subject{
		TreeOID: treeOID, Mode: validationevidence.SubjectIndex, Reusable: true,
		Scope: validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}, nil
}

func changeHeadSubject(repo string) (validationevidence.Subject, error) {
	treeOID, err := gitx.TreeOID(repo, "HEAD")
	if err != nil {
		return validationevidence.Subject{}, err
	}
	return validationevidence.Subject{
		TreeOID: treeOID, Mode: validationevidence.SubjectHead, Reusable: true,
		Scope: validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}, nil
}

func changeDirtySubject(repo string, status gitx.Status) (validationevidence.Subject, error) {
	var treeOID string
	var err error
	if status.Staged && !status.Unmerged {
		treeOID, err = gitx.WriteTree(repo)
	} else {
		treeOID, err = gitx.TreeOID(repo, "HEAD")
	}
	if err != nil {
		return validationevidence.Subject{}, err
	}
	return validationevidence.Subject{
		TreeOID: treeOID, Mode: validationevidence.SubjectDirty, Reusable: false,
		ReusableReason: "worktree or index contains content outside a reusable Git subject",
		Scope:          validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}, nil
}

func loadChangeImpactPolicy(repo string) (changeImpactPolicy, error) {
	return testengine.LoadChangeImpactPolicy(repo)
}

func selectChangeProfile(policy changeImpactPolicy, paths []string) (string, []changeImpactMatch, error) {
	return testengine.SelectChangeProfile(policy, paths)
}

func displayTestProfile(profile string) string {
	if profile == "" {
		return ""
	}
	return strings.ToUpper(profile[:1]) + profile[1:]
}

func inspectChangeReceipt(repo, profile string, subject validationevidence.Subject) (changeReceiptProbe, error) {
	store, err := validationevidence.Open(repo)
	if err != nil {
		return changeReceiptProbe{}, err
	}
	spec, err := testengine.EvidenceSpec(validationTestConfig(repo, profile))
	if err != nil {
		return changeReceiptProbe{}, err
	}
	fingerprint, err := store.Fingerprint(subject, spec)
	if err != nil {
		return changeReceiptProbe{}, err
	}
	return changeReceiptProbe{Subject: subject, Fingerprint: fingerprint, Decision: store.Check(subject, fingerprint)}, nil
}

func changeReceiptView(subject validationevidence.Subject, fingerprint validationevidence.Fingerprint, decision validationevidence.ReuseDecision) changeReceiptDecision {
	view := changeReceiptDecision{
		Hit: decision.Hit, Code: decision.Code, Reason: decision.Reason,
		CheckDurationMS: decision.CheckDurationMS, Subject: subject, Fingerprint: fingerprint,
	}
	if decision.Receipt != nil {
		view.ReceiptID = decision.Receipt.ReceiptID
	}
	return view
}

func changeCaseCounts(results []testengine.Result) (executed, reused int) {
	for _, result := range results {
		if strings.HasPrefix(result.Reason, "reused-from-node:") {
			reused++
		} else {
			executed++
		}
	}
	return executed, reused
}

func changeCommand(changes detectedChanges) string {
	switch changes.Mode {
	case "staged":
		return "aicoding change verify --staged --json"
	case "since":
		return "aicoding change verify --since " + changes.Since + " --json"
	default:
		return "aicoding change verify --json"
	}
}

func changeFailure(start time.Time, repo, message string, data any, category report.Category, nextAction string, err error) (report.Result, error) {
	result := report.Fail("change verify", start, message, data, err.Error())
	result.RepoRoot = repo
	result = report.WithDecision(result, category, nextAction)
	switch category {
	case report.CategoryUsage:
		result.ErrorKind = report.ErrorKindUsage
	case report.CategoryValidation, report.CategoryEvidenceMissing:
		result.ErrorKind = report.ErrorKindValidation
	}
	return result, err
}
