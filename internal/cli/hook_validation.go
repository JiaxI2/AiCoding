package cli

import (
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/repocontext"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/testengine"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

var hookPrePushInput io.Reader = os.Stdin

type prePushData struct {
	Gate validationevidence.PushGateReport `json:"gate"`
}

type validationAliasRefresh struct {
	CommitOID string   `json:"commitOID,omitempty"`
	TreeOID   string   `json:"treeOID,omitempty"`
	Profiles  []string `json:"profiles"`
	Bound     int      `json:"bound"`
	Missed    int      `json:"missed"`
	Errors    []string `json:"errors,omitempty"`
}

type postCommitData struct {
	RepoContext     repocontext.Report     `json:"repoContext"`
	ValidationAlias validationAliasRefresh `json:"validationAlias"`
}

func runHookPrePush(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("hook pre-push")
	repoArg := fs.String("repo-root", "", "repository root")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("hook pre-push", start, "cannot resolve repo root", nil, err.Error()), err
	}
	updates, err := gitx.ParsePushUpdates(hookPrePushInput)
	if err != nil {
		return report.Fail("hook pre-push", start, "cannot parse Git pre-push input", nil, err.Error()), err
	}
	policy, err := validationevidence.LoadPolicy(repo)
	if err != nil {
		return validationFailure("hook pre-push", repo, start, "cannot load validation policy", err)
	}
	store, err := validationevidence.Open(repo)
	if err != nil {
		return validationFailure("hook pre-push", repo, start, "cannot open validation evidence", err)
	}
	gate := store.GatePush(policy, updates)
	result := report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "hook pre-push",
		OK:            gate.OK,
		Message:       "pre-push validation Receipt gate",
		RepoRoot:      repo,
		Data:          prePushData{Gate: gate},
		Errors:        gate.Errors,
		ElapsedMS:     report.Elapsed(start),
	}
	if !result.OK {
		result.ErrorKind = report.ErrorKindValidation
	}
	return result, report.BoolErr(result.Errors)
}

func refreshHeadValidationAliases(repo string) validationAliasRefresh {
	refresh := validationAliasRefresh{Profiles: []string{}, Errors: []string{}}
	policy, err := validationevidence.LoadPolicy(repo)
	if err != nil {
		refresh.Errors = append(refresh.Errors, err.Error())
		return refresh
	}
	profileSet := make(map[string]struct{})
	for _, context := range policy.Contexts {
		profileSet[context.RequiredProfile] = struct{}{}
	}
	for profile := range profileSet {
		refresh.Profiles = append(refresh.Profiles, profile)
	}
	sort.Strings(refresh.Profiles)
	store, err := validationevidence.Open(repo)
	if err != nil {
		refresh.Errors = append(refresh.Errors, err.Error())
		return refresh
	}
	var commitOID string
	var treeOID string
	var commitErr error
	var treeErr error
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		commitOID, commitErr = gitx.HeadCommit(repo)
	}()
	go func() {
		defer wait.Done()
		treeOID, treeErr = gitx.TreeOID(repo, "HEAD")
	}()
	wait.Wait()
	if commitErr != nil || treeErr != nil {
		if commitErr != nil {
			refresh.Errors = append(refresh.Errors, commitErr.Error())
		}
		if treeErr != nil {
			refresh.Errors = append(refresh.Errors, treeErr.Error())
		}
		return refresh
	}
	refresh.CommitOID = commitOID
	refresh.TreeOID = treeOID
	subject := validationevidence.Subject{
		TreeOID: treeOID, Mode: validationevidence.SubjectHead, Reusable: true,
		Scope: validationevidence.Scope{IgnoredFilesOutOfScope: true},
	}
	for _, profile := range refresh.Profiles {
		spec, specErr := testengine.EvidenceSpec(validationTestConfig(repo, profile))
		if specErr != nil {
			refresh.Errors = append(refresh.Errors, profile+": "+specErr.Error())
			continue
		}
		fingerprint, fingerprintErr := store.Fingerprint(subject, spec)
		if fingerprintErr != nil {
			refresh.Errors = append(refresh.Errors, profile+": "+fingerprintErr.Error())
			continue
		}
		decision := store.Check(subject, fingerprint)
		if !decision.Hit || decision.Receipt == nil {
			refresh.Missed++
			continue
		}
		if bindErr := store.BindCommit(commitOID, *decision.Receipt); bindErr != nil {
			refresh.Errors = append(refresh.Errors, profile+": "+bindErr.Error())
			continue
		}
		refresh.Bound++
	}
	return refresh
}
