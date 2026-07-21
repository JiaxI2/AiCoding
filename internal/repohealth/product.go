package repohealth

import (
	"context"
	"time"

	"github.com/JiaxI2/AiCoding/internal/cache"
	"github.com/JiaxI2/AiCoding/internal/docsync"
	"github.com/JiaxI2/AiCoding/internal/governance"
	"github.com/JiaxI2/AiCoding/internal/kit"
	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
	"github.com/JiaxI2/AiCoding/internal/mcpcontrol"
	"github.com/JiaxI2/AiCoding/internal/repoinit"
	"github.com/JiaxI2/AiCoding/internal/report"
	"github.com/JiaxI2/AiCoding/internal/reuse"
	"github.com/JiaxI2/AiCoding/internal/runner"
)

type ProductOptions struct {
	Profile           string
	CodexConfig       string
	IncludeConfigured bool
	RuntimeProfile    string
	RuntimeSkill      string
	SourceRepository  string
	StandaloneRoot    string
}

func DoctorAll(ctx context.Context, repo string, opts ProductOptions) []report.Check {
	checks := []runner.Task{}
	checks = append(checks, productCheck("doctor.repository", "REPOSITORY", func() (interface{}, []string, []string) {
		status, errorsFound := StatusAll(repo)
		return status, nil, errorsFound
	}))
	checks = append(checks, productCheck("doctor.kits", "LIFECYCLE", func() (interface{}, []string, []string) {
		result := lifecyclecontrol.Run(ctx, repo, lifecyclecontrol.Options{
			Action: "doctor",
			Scope:  lifecyclecontrol.ScopeKit,
			All:    true,
		})
		return result, result.Warnings, result.Errors
	}))
	checks = append(checks, productCheck("doctor.mcp", "MCP", func() (interface{}, []string, []string) {
		return doctorInstalledMCP(ctx, repo, opts.CodexConfig)
	}))
	checks = append(checks, productCheck("doctor.runtime-skills", "SKILL", func() (interface{}, []string, []string) {
		result := lifecyclecontrol.Run(ctx, repo, lifecyclecontrol.Options{
			Action:           "doctor",
			Scope:            lifecyclecontrol.ScopeRuntimeSkill,
			RuntimeProfile:   opts.RuntimeProfile,
			RuntimeSkill:     opts.RuntimeSkill,
			SourceRepository: opts.SourceRepository,
			StandaloneRoot:   opts.StandaloneRoot,
		})
		return result, result.Warnings, result.Errors
	}))
	checks = append(checks, productCheck("doctor.hooks-wired", "REPOSITORY", func() (interface{}, []string, []string) {
		data, warnings := HooksWired(repo)
		return data, warnings, nil
	}))
	checks = append(checks, productCheck("doctor.provisioned", "REPOSITORY", func() (interface{}, []string, []string) {
		// Reads the aicoding.* markers provision wrote into .git/config — an
		// instant, zero-scan state check. Per-clone environment state, so absence
		// is a warning with the fix command, never a doctor failure.
		markers, initialized := repoinit.Status(repo)
		data := map[string]interface{}{"initialized": initialized, "markers": markers}
		if !initialized {
			return data, []string{"repository has not been provisioned; run `aicoding provision` to wire hooks and write local aicoding.* markers"}, nil
		}
		return data, nil, nil
	}))
	checks = append(checks, productCheck("doctor.repo-context", "REPO_CONTEXT", func() (interface{}, []string, []string) {
		result := lifecyclecontrol.Run(ctx, repo, lifecyclecontrol.Options{
			Action: "doctor",
			Scope:  lifecyclecontrol.ScopeRepoContext,
		})
		return result, result.Warnings, result.Errors
	}))
	checks = append(checks, productCheck("doctor.cache-bloat", "CACHE", func() (interface{}, []string, []string) {
		status, err := cache.Status(repo)
		if err != nil {
			return status, nil, []string{err.Error()}
		}
		return status, cache.BloatWarnings(status), nil
	}))
	checks = append(checks, productCheck("doctor.pwsh-budget", "POWERSHELL", func() (interface{}, []string, []string) {
		budget, errorsFound := ScanPwshBudget(repo)
		return budget, nil, errorsFound
	}))
	return executeProductChecks(ctx, checks, 4)
}

func doctorInstalledMCP(ctx context.Context, repo, codexConfig string) (interface{}, []string, []string) {
	catalog, err := mcpcontrol.LoadCatalogSnapshot(repo)
	if err != nil {
		return nil, nil, []string{err.Error()}
	}
	components, err := catalog.Select("", true)
	if err != nil {
		return nil, nil, []string{err.Error()}
	}
	status := mcpcontrol.StatusCatalog(repo, codexConfig, components)
	installed := make([]mcpcontrol.ComponentSnapshot, 0, len(components))
	warnings := []string{}
	errorsFound := []string{}
	for index, item := range status {
		for _, warning := range item.Warnings {
			warnings = append(warnings, item.ID+": "+warning)
		}
		for _, issue := range item.Errors {
			errorsFound = append(errorsFound, item.ID+": "+issue)
		}
		if item.Installed {
			if index < len(components) {
				installed = append(installed, components[index])
			}
			continue
		}
		warnings = append(warnings, item.ID+": not installed in the current repository; doctor command skipped")
	}
	results := []mcpcontrol.CommandResult{}
	if len(installed) > 0 {
		results = mcpcontrol.DoctorCatalogComponentsContext(ctx, repo, installed)
		for index, result := range results {
			id := "component"
			if index < len(installed) {
				id = installed[index].Entry().ID
			}
			for _, issue := range result.Errors {
				errorsFound = append(errorsFound, id+": "+issue)
			}
		}
	}
	return map[string]interface{}{
		"inputDigest": catalog.Digest(),
		"status":      status,
		"results":     results,
	}, warnings, errorsFound
}

func VerifyAll(ctx context.Context, repo string, opts ProductOptions) []report.Check {
	checks := []runner.Task{}
	checks = append(checks, productCheck("verify.hooks", "REPOSITORY", func() (interface{}, []string, []string) {
		data, errorsFound := VerifyHooks(repo)
		return data, nil, errorsFound
	}))
	checks = append(checks, productCheck("verify.repo-text", "REPOSITORY", func() (interface{}, []string, []string) {
		data, errorsFound := VerifyRepoText(repo)
		warnings := []string{}
		for _, item := range data {
			for _, warning := range item.Warnings {
				warnings = append(warnings, item.Path+": "+warning)
			}
		}
		return data, warnings, errorsFound
	}))
	checks = append(checks, productCheck("verify.release-notes", "RELEASE", func() (interface{}, []string, []string) {
		data, errorsFound := VerifyReleaseNotes(repo)
		return data, nil, errorsFound
	}))
	checks = append(checks, productCheck("verify.governance", "GOVERNANCE", func() (interface{}, []string, []string) {
		errorsFound := governance.Lint(repo, "verify", "")
		return map[string]interface{}{"mode": "verify"}, nil, errorsFound
	}))
	checks = append(checks, productCheck("verify.dependencies", "GOVERNANCE", func() (interface{}, []string, []string) {
		data := governance.CheckDependencies(repo)
		return data, data.Warnings, data.Errors
	}))
	checks = append(checks, productCheck("verify.layout", "GOVERNANCE", func() (interface{}, []string, []string) {
		data := governance.CheckLayout(repo)
		return data, nil, data.Errors
	}))
	checks = append(checks, productCheck("verify.reuse", "GOVERNANCE", func() (interface{}, []string, []string) {
		data := reuse.Verify(repo)
		return data, data.Warnings, data.Errors
	}))
	checks = append(checks, productCheck("verify.docsync", "DOCSYNC", func() (interface{}, []string, []string) {
		mode := "all"
		if opts.Profile == "Release" {
			mode = "release"
		}
		data := docsync.Check(repo, mode)
		return data, data.Warnings, data.Errors
	}))
	checks = append(checks, productCheck("verify.kit-lifecycle", "LIFECYCLE", func() (interface{}, []string, []string) {
		data := lifecyclecontrol.Run(ctx, repo, lifecyclecontrol.Options{
			Action: "verify",
			Scope:  lifecyclecontrol.ScopeKit,
			All:    true,
		})
		return data, data.Warnings, data.Errors
	}))
	checks = append(checks, productCheck("verify.skills", "SKILL", func() (interface{}, []string, []string) {
		catalog, err := kit.LoadCatalogSnapshot(repo)
		if err != nil {
			return nil, nil, []string{err.Error()}
		}
		selected, err := catalog.Select("", true)
		if err != nil {
			return nil, nil, []string{err.Error()}
		}
		data := kit.VerifyCatalogSkills(repo, selected, opts.Profile)
		return data, data.Warnings, data.Errors
	}))
	checks = append(checks, productCheck("verify.mcp-registry", "MCP", func() (interface{}, []string, []string) {
		errorsFound := mcpcontrol.DoctorRegistry(repo)
		return map[string]interface{}{"registry": "config/mcp-registry.json"}, nil, errorsFound
	}))
	checks = append(checks, productCheck("verify.runtime-skills", "SKILL", func() (interface{}, []string, []string) {
		data := lifecyclecontrol.Run(ctx, repo, lifecyclecontrol.Options{
			Action:           "verify",
			Scope:            lifecyclecontrol.ScopeRuntimeSkill,
			RuntimeProfile:   opts.RuntimeProfile,
			RuntimeSkill:     opts.RuntimeSkill,
			SourceRepository: opts.SourceRepository,
			StandaloneRoot:   opts.StandaloneRoot,
		})
		return data, data.Warnings, data.Errors
	}))
	checks = append(checks, productCheck("verify.repo-context", "REPO_CONTEXT", func() (interface{}, []string, []string) {
		data := lifecyclecontrol.Run(ctx, repo, lifecyclecontrol.Options{
			Action: "verify",
			Scope:  lifecyclecontrol.ScopeRepoContext,
		})
		return data, data.Warnings, data.Errors
	}))
	if opts.IncludeConfigured || opts.CodexConfig != "" {
		checks = append(checks, productCheck("verify.mcp-config", "MCP", func() (interface{}, []string, []string) {
			inventory, err := mcpcontrol.ListInventory(repo, opts.CodexConfig)
			if err != nil {
				return nil, nil, []string{err.Error()}
			}
			return inventory, inventory.Warnings, nil
		}))
	}
	return executeProductChecks(ctx, checks, 4)
}

func productCheck(id, category string, run func() (interface{}, []string, []string)) runner.Task {
	return runner.Task{
		ID:     id,
		Action: "repohealth.product-check",
		Group:  category,
		Run: func(context.Context) runner.TaskResult {
			started := time.Now()
			details, warnings, errorsFound := run()
			check := report.NewCheck(id, category, started, details, warnings, errorsFound)
			return runner.TaskResult{OK: check.OK, Warnings: check.Warnings, Errors: check.Errors, Data: check}
		},
	}
}

func executeProductChecks(ctx context.Context, tasks []runner.Task, maxParallel int) []report.Check {
	results := runner.Run(ctx, tasks, runner.Options{MaxParallel: maxParallel})
	checks := make([]report.Check, len(results))
	for index, result := range results {
		if check, ok := result.Data.(report.Check); ok {
			checks[index] = check
			continue
		}
		checks[index] = report.Check{
			ID:       tasks[index].ID,
			Category: tasks[index].Group,
			OK:       false,
			Status:   "FAIL",
			Warnings: append([]string{}, result.Warnings...),
			Errors:   append([]string{}, result.Errors...),
		}
		if len(checks[index].Errors) == 0 {
			checks[index].Errors = []string{"product check returned no result"}
		}
	}
	return checks
}
