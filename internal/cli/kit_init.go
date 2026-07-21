package cli

import (
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

func runKitInit(args []string, start time.Time) (report.Result, error) {
	fs := newFlagSet("kit init")
	repoArg := fs.String("repo-root", "", "repository root")
	externalArg := fs.Bool("external", false, "generate an external-wrapper boundary card")
	dryRunArg := fs.Bool("dry-run", false, "report planned files and digests without writing")
	_ = fs.Bool("json", false, "json output")

	id := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		flagArgs = args[1:]
	}
	if err := parseNoPositionals(fs, flagArgs); err != nil {
		return report.Result{}, err
	}
	if id == "" {
		return report.Result{}, usageErrorf("kit init requires an id before flags")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("kit init", start, "cannot resolve repo root", nil, err.Error()), err
	}

	initReport, initErr := kit.Init(repo, id, kit.InitOptions{External: *externalArg, DryRun: *dryRunArg})
	result := report.Result{
		SchemaVersion: report.SchemaVersion,
		Command:       "kit init",
		OK:            initReport.OK,
		Message:       "Kit scaffold",
		RepoRoot:      repo,
		Data:          initReport,
		Errors:        append([]string(nil), initReport.Errors...),
		ElapsedMS:     report.Elapsed(start),
	}
	if initErr != nil {
		result.ErrorKind = report.ErrorKindValidation
		return result, report.BoolErr(result.Errors)
	}
	return result, nil
}
