package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/loopkit/gateref"
	"github.com/JiaxI2/AiCoding/internal/loopkit/transition"
	"github.com/JiaxI2/AiCoding/internal/loopkit/workspec"
	"github.com/JiaxI2/AiCoding/internal/loopkit/workstate"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

type workSpecData struct {
	File   string        `json:"file"`
	Digest string        `json:"digest"`
	Spec   workspec.Spec `json:"spec"`
}

type workScopeData struct {
	Changed    []string `json:"changed"`
	Allowed    []string `json:"allowed"`
	Violations []string `json:"violations"`
}

type workEvaluationData struct {
	File           string                     `json:"file"`
	SpecDigest     string                     `json:"specDigest"`
	Spec           workspec.Spec              `json:"spec"`
	Session        workstate.Session          `json:"session"`
	Subject        validationevidence.Subject `json:"subject"`
	Scope          workScopeData              `json:"scope"`
	Gates          []transition.GateStatus    `json:"gates"`
	Decision       transition.Decision        `json:"decision"`
	RequiredAction string                     `json:"requiredAction,omitempty"`
}

type workRecordData struct {
	Session  workstate.Session   `json:"session"`
	Decision transition.Decision `json:"decision"`
}

func runWork(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("work requires subcommand: validate, next, status, or record")
	}
	switch strings.ToLower(args[0]) {
	case "validate":
		return runWorkValidate(args[1:], start)
	case "next":
		return runWorkEvaluate("next", args[1:], start)
	case "status":
		return runWorkEvaluate("status", args[1:], start)
	case "record":
		return runWorkRecord(args[1:], start)
	default:
		return report.Result{}, usageErrorf("unsupported work subcommand: %s", args[0])
	}
}

func runWorkValidate(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("work validate")
	repoArg := fs.String("repo-root", "", "repository root")
	fileArg := fs.String("file", "", "WorkSpec JSON file")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if strings.TrimSpace(*fileArg) == "" {
		return report.Result{}, usageErrorf("work validate requires --file")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("work validate", start, "cannot resolve repo root", nil, err.Error()), err
	}
	spec, digest, file, err := loadWorkSpec(repo, *fileArg)
	if err != nil {
		return workFailure("work validate", repo, start, "WorkSpec validation failed", err)
	}
	data := workSpecData{File: displayWorkPath(repo, file), Digest: digest, Spec: spec}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "work validate",
		OK:            true,
		Message:       "bounded work specification is valid",
		RepoRoot:      repo,
		InputDigest:   digest,
		Data:          data,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func runWorkEvaluate(subcommand string, args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("work " + subcommand)
	repoArg := fs.String("repo-root", "", "repository root")
	fileArg := fs.String("file", "", "WorkSpec JSON file")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if strings.TrimSpace(*fileArg) == "" {
		return report.Result{}, usageErrorf("work %s requires --file", subcommand)
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("work "+subcommand, start, "cannot resolve repo root", nil, err.Error()), err
	}
	spec, digest, file, err := loadWorkSpec(repo, *fileArg)
	if err != nil {
		return workFailure("work "+subcommand, repo, start, "WorkSpec validation failed", err)
	}
	session, err := workstate.Load(repo, spec.ID)
	if err != nil {
		return workFailure("work "+subcommand, repo, start, "cannot read work state", err)
	}
	if session.Exists && session.Snapshot.SpecDigest != digest {
		return workFailure("work "+subcommand, repo, start, "work state does not match WorkSpec", errors.New("recorded spec digest differs from current WorkSpec"))
	}
	data, err := evaluateWork(repo, spec, digest, displayWorkPath(repo, file), session, session.History, time.Now().UTC())
	if err != nil {
		return workFailure("work "+subcommand, repo, start, "cannot decide bounded work transition", err)
	}
	warnings := workWarnings(data)
	message := "next bounded-work transition"
	if subcommand == "status" {
		message = "bounded-work status"
	}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "work " + subcommand,
		OK:            true,
		Message:       message,
		RepoRoot:      repo,
		InputDigest:   digest,
		Data:          data,
		Warnings:      warnings,
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func runWorkRecord(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("work record")
	repoArg := fs.String("repo-root", "", "repository root")
	fileArg := fs.String("file", "", "WorkSpec JSON file")
	attemptArg := fs.String("attempt", "", "attempt JSON file")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if strings.TrimSpace(*fileArg) == "" || strings.TrimSpace(*attemptArg) == "" {
		return report.Result{}, usageErrorf("work record requires --file and --attempt")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("work record", start, "cannot resolve repo root", nil, err.Error()), err
	}
	spec, digest, file, err := loadWorkSpec(repo, *fileArg)
	if err != nil {
		return workFailure("work record", repo, start, "WorkSpec validation failed", err)
	}
	var attempt transition.Attempt
	attemptFile := resolveWorkPath(repo, *attemptArg)
	if err := decodeWorkFile(attemptFile, &attempt); err != nil {
		return workFailure("work record", repo, start, "attempt validation failed", err)
	}
	resolvedTree, err := gitx.TreeOID(repo, attempt.SubjectTreeOID)
	if err != nil || resolvedTree != attempt.SubjectTreeOID {
		if err == nil {
			err = errors.New("subjectTreeOID must identify a tree object exactly")
		}
		return workFailure("work record", repo, start, "attempt tree is invalid", err)
	}
	store, err := validationevidence.Open(repo)
	if err != nil {
		return workFailure("work record", repo, start, "cannot open validation evidence", err)
	}
	if err := validateAttemptGateRefs(store, &attempt); err != nil {
		return workFailure("work record", repo, start, "attempt gate reference is invalid", err)
	}
	session, err := workstate.Load(repo, spec.ID)
	if err != nil {
		return workFailure("work record", repo, start, "cannot read work state", err)
	}
	if session.Exists {
		if session.Snapshot.SpecDigest != digest {
			return workFailure("work record", repo, start, "work state does not match WorkSpec", errors.New("recorded spec digest differs from current WorkSpec"))
		}
		if session.Snapshot.LastDecision.State != transition.Continue {
			return workFailure("work record", repo, start, "work session is already stopped", fmt.Errorf("last decision is %s", session.Snapshot.LastDecision.State))
		}
	}
	history := append(append([]transition.Attempt(nil), session.History...), attempt)
	data, err := evaluateWork(repo, spec, digest, displayWorkPath(repo, file), session, history, attempt.EndedAt.UTC())
	if err != nil {
		return workFailure("work record", repo, start, "cannot decide recorded transition", err)
	}
	recorded, err := workstate.Record(repo, spec.ID, digest, displayWorkPath(repo, file), attempt, data.Decision)
	if err != nil {
		return workFailure("work record", repo, start, "cannot append work attempt", err)
	}
	return report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "work record",
		OK:            true,
		Message:       "bounded-work attempt recorded",
		RepoRoot:      repo,
		InputDigest:   digest,
		Data:          workRecordData{Session: recorded, Decision: data.Decision},
		Warnings:      workWarnings(data),
		ElapsedMS:     report.Elapsed(start),
	}, nil
}

func evaluateWork(repo string, spec workspec.Spec, digest, file string, session workstate.Session, history []transition.Attempt, now time.Time) (workEvaluationData, error) {
	store, err := validationevidence.Open(repo)
	if err != nil {
		return workEvaluationData{}, err
	}
	subject, err := workSubject(store, history)
	if err != nil {
		return workEvaluationData{}, err
	}
	gates, err := workGateStatuses(repo, store, spec, subject)
	if err != nil {
		return workEvaluationData{}, err
	}
	scope, err := checkWorkScope(repo, spec)
	if err != nil {
		return workEvaluationData{}, err
	}
	if len(scope.Violations) > 0 {
		gates = append(gates, transition.GateStatus{
			Ref: gateref.GateRef{Profile: "write-scope"}, SubjectTreeOID: subject.TreeOID, State: transition.GateViolation,
		})
	}
	decision, err := transition.Decide(spec, history, gates, now)
	if err != nil {
		return workEvaluationData{}, err
	}
	return workEvaluationData{
		File: file, SpecDigest: digest, Spec: spec, Session: session,
		Subject: subject, Scope: scope, Gates: gates, Decision: decision,
		RequiredAction: workRequiredAction(decision),
	}, nil
}

func workSubject(store validationevidence.Repository, history []transition.Attempt) (validationevidence.Subject, error) {
	if len(history) == 0 {
		return store.Capture(validationevidence.TargetAuto)
	}
	return validationevidence.Subject{
		TreeOID:  history[len(history)-1].SubjectTreeOID,
		Mode:     validationevidence.SubjectHead,
		Reusable: true,
		Scope:    validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}, nil
}

func workGateStatuses(repo string, store validationevidence.Repository, spec workspec.Spec, subject validationevidence.Subject) ([]transition.GateStatus, error) {
	statuses := make([]transition.GateStatus, 0, len(spec.Control.Authority.RequiredGates))
	for _, requestedProfile := range spec.Control.Authority.RequiredGates {
		profile, _, err := normalizeTestProfile(requestedProfile)
		if err != nil {
			return nil, fmt.Errorf("required gate %q is not a validation profile: %w", requestedProfile, err)
		}
		evidenceSpec, err := testengine.EvidenceSpec(validationTestConfig(repo, profile))
		if err != nil {
			return nil, fmt.Errorf("compute %s validation semantics: %w", profile, err)
		}
		fingerprint, err := store.Fingerprint(subject, evidenceSpec)
		if err != nil {
			return nil, fmt.Errorf("compute %s validation identity: %w", profile, err)
		}
		check := store.Check(subject, fingerprint)
		status := transition.GateStatus{
			Ref:            gateref.GateRef{Profile: profile, ValidationIdentity: fingerprint.Identity},
			SubjectTreeOID: subject.TreeOID,
			State:          transition.GatePending,
		}
		if check.Hit && check.Receipt != nil {
			status.State = transition.GateSatisfied
			status.Ref.ValidationIdentity = check.Receipt.ValidationIdentity
			status.Ref.ReceiptID = check.Receipt.ReceiptID
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func validateAttemptGateRefs(store validationevidence.Repository, attempt *transition.Attempt) error {
	for index, ref := range attempt.GateRefs {
		profile, _, err := normalizeTestProfile(ref.Profile)
		if err != nil {
			return err
		}
		attempt.GateRefs[index].Profile = profile
		receipts, err := store.List(profile)
		if err != nil {
			return err
		}
		found := false
		for _, receipt := range receipts {
			if receipt.ReceiptID == ref.ReceiptID && receipt.ValidationIdentity == ref.ValidationIdentity && receipt.Fingerprint.SubjectTreeOID == attempt.SubjectTreeOID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("gateRefs[%d] does not identify a matching Receipt", index)
		}
	}
	return nil
}

func loadWorkSpec(repo, fileArg string) (workspec.Spec, string, string, error) {
	file := resolveWorkPath(repo, fileArg)
	var spec workspec.Spec
	if err := decodeWorkFile(file, &spec); err != nil {
		return workspec.Spec{}, "", file, err
	}
	spec = spec.Normalized()
	if err := spec.Validate(); err != nil {
		return workspec.Spec{}, "", file, err
	}
	digest, err := spec.Digest()
	return spec, digest, file, err
}

func resolveWorkPath(repo, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(repo, filepath.Clean(value))
}

func displayWorkPath(repo, file string) string {
	rel, err := filepath.Rel(repo, file)
	if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(file)
}

func decodeWorkFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
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

func checkWorkScope(repo string, spec workspec.Spec) (workScopeData, error) {
	tracked, err := gitx.Run(repo, "diff", "--name-only", "HEAD", "--")
	if err != nil {
		return workScopeData{}, err
	}
	untracked, err := gitx.Run(repo, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return workScopeData{}, err
	}
	changed := compactStrings(append(splitWorkPaths(tracked), splitWorkPaths(untracked)...))
	sort.Strings(changed)
	result := workScopeData{Changed: changed, Allowed: []string{}, Violations: []string{}}
	for _, file := range changed {
		allowed, err := matchesAnyWorkPattern(spec.Control.Authority.WriteScope.Allow, file)
		if err != nil {
			return workScopeData{}, err
		}
		denied, err := matchesAnyWorkPattern(spec.Control.Authority.WriteScope.Deny, file)
		if err != nil {
			return workScopeData{}, err
		}
		if allowed && !denied {
			result.Allowed = append(result.Allowed, file)
		} else {
			result.Violations = append(result.Violations, file)
		}
	}
	return result, nil
}

func splitWorkPaths(output string) []string {
	paths := []string{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(filepath.ToSlash(line))
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}

func matchesAnyWorkPattern(patterns []string, file string) (bool, error) {
	for _, pattern := range patterns {
		matcher, err := regexp.Compile(workGlobRegex(pattern))
		if err != nil {
			return false, fmt.Errorf("invalid work scope pattern %q: %w", pattern, err)
		}
		if matcher.MatchString(filepath.ToSlash(file)) {
			return true, nil
		}
	}
	return false, nil
}

func workGlobRegex(pattern string) string {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	var out strings.Builder
	out.WriteByte('^')
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '*':
			if index+1 < len(pattern) && pattern[index+1] == '*' {
				out.WriteString(".*")
				index++
			} else {
				out.WriteString("[^/]*")
			}
		case '?':
			out.WriteString("[^/]")
		default:
			out.WriteString(regexp.QuoteMeta(string(pattern[index])))
		}
	}
	out.WriteByte('$')
	return out.String()
}

func workRequiredAction(decision transition.Decision) string {
	switch decision.State {
	case transition.Continue:
		for _, gate := range decision.RequiredGates {
			if gate.State != transition.GateSatisfied {
				return "bin\\aicoding.exe test --profile " + displayValidationProfile(gate.Ref.Profile) + " --reuse off --json"
			}
		}
	case transition.StopViolation:
		return "review and restore out-of-scope changes before continuing"
	case transition.Checkpoint:
		return "start a fresh task with the recorded state and request review"
	case transition.StopBudget, transition.StopStalled:
		return "request human review before another attempt"
	}
	return ""
}

func displayValidationProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case testengine.ProfileSmoke:
		return "Smoke"
	case testengine.ProfileFull:
		return "Full"
	case testengine.ProfileRelease:
		return "Release"
	default:
		return profile
	}
}

func workWarnings(data workEvaluationData) []string {
	warnings := []string{}
	if !data.Subject.Reusable && data.Subject.ReusableReason != "" {
		warnings = append(warnings, "validation subject is not reusable: "+data.Subject.ReusableReason)
	}
	if len(data.Scope.Violations) > 0 {
		warnings = append(warnings, "write scope violations: "+strings.Join(data.Scope.Violations, ", "))
	}
	return warnings
}

func workFailure(command, repo string, start time.Time, message string, err error) (report.Result, error) {
	result := report.Fail(command, start, message, nil, err.Error())
	result.RepoRoot = repo
	result.ErrorKind = report.ErrorKindValidation
	return result, report.BoolErr(result.Errors)
}
