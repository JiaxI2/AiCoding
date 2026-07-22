package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/JiaxI2/AiCoding/internal/kit"
	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

type kitPrefetchJob struct {
	Status  string `json:"status"`
	PID     int    `json:"pid"`
	LogPath string `json:"logPath"`
	Command string `json:"command"`
}

type kitRegisterResult struct {
	Registration kit.RegisterReport `json:"registration"`
	Prefetch     *kitPrefetchJob    `json:"prefetch,omitempty"`
}

var startKitPrefetchJob = startDetachedKitPrefetch

func runKitRegister(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("kit register")
	repoArg := fs.String("repo-root", "", "repository root")
	manifestArg := fs.String("manifest", "", "repository-local pinned Kit manifest")
	prefetchArg := fs.Bool("prefetch", false, "start registration-time pin prefetch in the background")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("kit register", start, "cannot resolve repo root", nil, err.Error()), err
	}
	registration, err := kit.Register(repo, *manifestArg)
	data := kitRegisterResult{Registration: registration}
	if err != nil {
		result := report.Fail("kit register", start, "pinned Kit registration failed", data, err.Error())
		result.RepoRoot = repo
		result = report.WithDecision(result, report.CategoryValidation, "fix the pinned manifest and rerun `aicoding kit register --manifest "+*manifestArg+" --json`")
		return result, err
	}
	if *prefetchArg {
		job, startErr := startKitPrefetchJob(repo, registration.ID)
		data.Prefetch = &job
		if startErr != nil {
			result := report.Fail("kit register", start, "Kit registered but background prefetch could not start", data, startErr.Error())
			result.RepoRoot = repo
			result = report.WithDecision(result, report.CategoryTransient, "aicoding kit prefetch --id "+registration.ID+" --json")
			return result, startErr
		}
	}
	return report.Result{
		SchemaVersion: 1, Command: "kit register", OK: true, Message: "content-pinned Kit registered",
		RepoRoot: repo, InputDigest: registration.SourceIdentity, Data: data, ElapsedMS: report.Elapsed(start),
	}, nil
}

func runKitPrefetch(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("kit prefetch")
	repoArg := fs.String("repo-root", "", "repository root")
	idArg := fs.String("id", "", "registered Kit id")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args); err != nil {
		return report.Result{}, err
	}
	if *idArg == "" {
		return report.Result{}, usageErrorf("kit prefetch requires --id")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("kit prefetch", start, "cannot resolve repo root", nil, err.Error()), err
	}
	status, err := kit.PrefetchRegisteredKit(context.Background(), repo, *idArg)
	if err != nil {
		result := report.Fail("kit prefetch", start, "pin prefetch failed", status, err.Error())
		result.RepoRoot = repo
		var cacheMiss *kit.PinCacheMissError
		if errors.As(err, &cacheMiss) {
			result = report.WithDecision(result, report.CategoryEvidenceMissing, cacheMiss.RequiredAction)
		} else {
			result = report.WithDecision(result, report.CategoryValidation, "verify source.url and the immutable 40-hex commit, then rerun `aicoding kit prefetch --id "+*idArg+" --json`")
		}
		return result, err
	}
	return report.Result{
		SchemaVersion: 1, Command: "kit prefetch", OK: true, Message: "pinned Kit source is locally resolved",
		RepoRoot: repo, InputDigest: status.Identity, Data: status, ElapsedMS: report.Elapsed(start),
	}, nil
}

func startDetachedKitPrefetch(repo, id string) (kitPrefetchJob, error) {
	executable, err := os.Executable()
	if err != nil {
		return kitPrefetchJob{}, err
	}
	cacheRoot, err := kit.PinCacheRoot(repo)
	if err != nil {
		return kitPrefetchJob{}, err
	}
	jobsRoot := filepath.Join(cacheRoot, ".jobs")
	if err := os.MkdirAll(jobsRoot, 0o755); err != nil {
		return kitPrefetchJob{}, err
	}
	logPath := filepath.Join(jobsRoot, id+".json")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return kitPrefetchJob{}, err
	}
	command := "aicoding kit prefetch --id " + id + " --json"
	job := kitPrefetchJob{Status: "starting", LogPath: logFile.Name(), Command: command}
	cmd := exec.Command(executable, "kit", "prefetch", "--id", id, "--repo-root", repo, "--json")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return job, err
	}
	job.Status = "started"
	job.PID = cmd.Process.Pid
	if err := logFile.Close(); err != nil {
		_ = cmd.Process.Release()
		return job, err
	}
	if err := cmd.Process.Release(); err != nil {
		return job, err
	}
	return job, nil
}

func lifecyclePinRequiredAction(result lifecyclecontrol.Report) string {
	for _, adapter := range result.Adapters {
		switch data := adapter.Data.(type) {
		case kit.ActionReport:
			if data.RequiredAction != "" {
				return data.RequiredAction
			}
		case kit.LifecyclePlan:
			if data.RequiredAction != "" {
				return data.RequiredAction
			}
		}
	}
	return ""
}
