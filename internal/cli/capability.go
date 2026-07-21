package cli

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/capability"
	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/report"
)

type capabilityIndexWrite struct {
	Changed []string `json:"changed"`
}

var commandPublicEntryExists func(string) bool

func runCapability(args []string, start time.Time) (report.Result, error) {
	if len(args) < 1 {
		return report.Result{}, usageErrorf("capability requires subcommand: list, describe, or index")
	}
	sub := args[0]
	if !validChoice(sub, "list", "describe", "index") {
		return report.Result{}, usageErrorf("unsupported capability subcommand: %s", sub)
	}
	fs := newFlagSet("capability " + sub)
	repoArg := fs.String("repo-root", "", "repository root")
	idArg := fs.String("id", "", "capability id; describe only")
	typeArg := fs.String("type", "", "capability type filter; list only")
	statusArg := fs.String("status", "", "capability status filter; list only")
	writeArg := fs.Bool("write", false, "write generated indexes; index only")
	_ = fs.Bool("json", false, "json output")
	if err := parseNoPositionals(fs, args[1:]); err != nil {
		return report.Result{}, err
	}
	if sub != "describe" && *idArg != "" {
		return report.Result{}, usageErrorf("--id is only valid for capability describe")
	}
	if sub != "list" && (*typeArg != "" || *statusArg != "") {
		return report.Result{}, usageErrorf("--type and --status are only valid for capability list")
	}
	if sub != "index" && *writeArg {
		return report.Result{}, usageErrorf("--write is only valid for capability index")
	}
	if sub == "index" && !*writeArg {
		return report.Result{}, usageErrorf("capability index requires --write")
	}
	repo, err := platform.ResolveRepoRoot(*repoArg)
	if err != nil {
		return report.Fail("capability "+sub, start, "cannot resolve repo root", nil, err.Error()), err
	}
	catalog, err := capability.Load(repo)
	if err != nil {
		result := report.Fail("capability "+sub, start, "cannot load internal capability registry", nil, err.Error())
		result.RepoRoot = repo
		return result, err
	}

	switch sub {
	case "list":
		selected, selectErr := capability.List(catalog, *typeArg, *statusArg)
		if selectErr != nil {
			return capabilityValidationFailure("capability list", start, repo, catalog.Digest, selectErr)
		}
		return report.Result{SchemaVersion: 1, Command: "capability list", OK: true, Message: "internal capability catalog", RepoRoot: repo, InputDigest: catalog.Digest, Data: selected, ElapsedMS: report.Elapsed(start)}, nil
	case "describe":
		if strings.TrimSpace(*idArg) == "" {
			return report.Result{}, usageErrorf("capability describe requires --id")
		}
		item, describeErr := capability.Describe(catalog, *idArg)
		if describeErr != nil {
			return capabilityValidationFailure("capability describe", start, repo, catalog.Digest, describeErr)
		}
		return report.Result{SchemaVersion: 1, Command: "capability describe", OK: true, Message: "internal capability detail", RepoRoot: repo, InputDigest: catalog.Digest, Data: item, ElapsedMS: report.Elapsed(start)}, nil
	case "index":
		changed, writeErr := writeCapabilityIndex(repo, catalog)
		if writeErr != nil {
			result := report.Fail("capability index", start, "cannot write capability index", nil, writeErr.Error())
			result.RepoRoot = repo
			result.InputDigest = catalog.Digest
			return result, writeErr
		}
		return report.Result{SchemaVersion: 1, Command: "capability index", OK: true, Message: "generated capability indexes", RepoRoot: repo, InputDigest: catalog.Digest, Data: capabilityIndexWrite{Changed: changed}, ElapsedMS: report.Elapsed(start)}, nil
	default:
		return report.Result{}, usageErrorf("unsupported capability subcommand: %s", sub)
	}
}

func runGovernanceCapabilities(repo string, start time.Time) (report.Result, error) {
	catalog, err := capability.Load(repo)
	if err != nil {
		result := report.Fail("governance capabilities", start, "cannot load internal capability registry", nil, err.Error())
		result.RepoRoot = repo
		return result, err
	}
	verification := capability.Verify(repo, catalog, capability.VerifyOptions{
		PublicEntryExists: commandPublicEntryExists,
		CheckGenerated:    true,
	})
	return report.Result{
		SchemaVersion: 1,
		Command:       "governance capabilities",
		OK:            verification.OK,
		Message:       "internal capability orphan and documentation gate",
		RepoRoot:      repo,
		InputDigest:   catalog.Digest,
		Data:          verification,
		Errors:        verification.Errors,
		ElapsedMS:     report.Elapsed(start),
	}, report.BoolErr(verification.Errors)
}

func capabilityValidationFailure(command string, start time.Time, repo, digest string, err error) (report.Result, error) {
	result := report.Fail(command, start, "capability selection failed", nil, err.Error())
	result.ErrorKind = report.ErrorKindValidation
	result.RepoRoot = repo
	result.InputDigest = digest
	return result, report.BoolErr(result.Errors)
}

func writeCapabilityIndex(repo string, catalog capability.Catalog) ([]string, error) {
	readmePath := filepath.Join(repo, "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return nil, err
	}
	rendered, err := capability.RenderIndex(catalog, string(readme))
	if err != nil {
		return nil, err
	}
	changed := []string{}
	if string(readme) != rendered.README {
		if err := os.WriteFile(readmePath, []byte(rendered.README), 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, "README.md")
	}
	documentPath := filepath.Join(repo, filepath.FromSlash(capability.CapabilitiesPath))
	document, readErr := os.ReadFile(documentPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, readErr
	}
	if readErr != nil || string(document) != rendered.Document {
		if err := os.WriteFile(documentPath, []byte(rendered.Document), 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, capability.CapabilitiesPath)
	}
	return changed, nil
}

func catalogHasPublicEntry(entry string) bool {
	wanted := strings.Fields(strings.TrimSpace(entry))
	if len(wanted) < 2 || wanted[0] != "aicoding" {
		return false
	}
	route, ok := commands.lookup(wanted[1])
	if !ok {
		return false
	}
	if len(wanted) == 2 {
		return true
	}
	for _, form := range commands.descriptor.Help {
		if form.Command != route.descriptor.ID {
			continue
		}
		pattern := strings.Fields(form.Usage)
		if publicEntryMatchesUsage(wanted, pattern) {
			return true
		}
	}
	return false
}

func publicEntryMatchesUsage(entry, usage []string) bool {
	if len(entry) > len(usage) {
		return false
	}
	for index, value := range entry {
		matched := false
		for _, option := range strings.Split(usage[index], "|") {
			if value == strings.Trim(option, "()") {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
