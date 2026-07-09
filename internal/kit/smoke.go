package kit

import (
	"context"

	"github.com/JiaxI2/AiCoding/internal/platform"
	"github.com/JiaxI2/AiCoding/internal/runner"
)

var allowedManifestModes = map[string]bool{
	"go-builtin":           true,
	"external-cli":         true,
	"powershell-specialty": true,
	"declarative":          true,
}

func DoctorKits(repo string, entries []RegistryKit) []string {
	errs := []string{}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.ID == "" {
			errs = append(errs, "registry kit id is empty")
		}
		if seen[e.ID] {
			errs = append(errs, "duplicate kit id: "+e.ID)
		}
		seen[e.ID] = true
		if e.Manifest == "" {
			errs = append(errs, e.ID+": manifest is empty")
			continue
		}
		m, err := LoadManifest(repo, e.Manifest)
		if err != nil {
			errs = append(errs, e.ID+": cannot load manifest: "+err.Error())
			continue
		}
		if m.ID != e.ID {
			errs = append(errs, e.ID+": manifest id mismatch: "+m.ID)
		}
		if !allowedManifestModes[m.Mode] {
			errs = append(errs, e.ID+": invalid mode: "+m.Mode)
		}
		if len(m.Kind) == 0 {
			errs = append(errs, e.ID+": empty kind")
		}
		if len(m.Commands) == 0 {
			errs = append(errs, e.ID+": empty commands")
		}
	}
	return errs
}

func SmokeKits(repo string, entries []RegistryKit) []SmokeResult {
	tasks := make([]runner.Task, 0, len(entries))
	for _, e := range entries {
		e := e
		tasks = append(tasks, runner.Task{
			ID:    e.ID,
			Group: "kit-smoke",
			Run: func(context.Context) runner.TaskResult {
				return runner.TaskResult{ID: e.ID, OK: true, Data: smokeKit(repo, e)}
			},
		})
	}

	results := []SmokeResult{}
	for _, taskResult := range runner.Run(context.Background(), tasks, runner.Options{}) {
		result, ok := taskResult.Data.(SmokeResult)
		if !ok {
			result = SmokeResult{ID: taskResult.ID, OK: false, Status: "failed", Errors: []string{"invalid smoke result"}}
		}
		results = append(results, result)
	}
	return results
}

func smokeKit(repo string, e RegistryKit) SmokeResult {
	errs := []string{}
	if !platform.IsFile(platform.RepoPath(repo, e.Manifest)) {
		errs = append(errs, "manifest missing")
	}
	m, err := LoadManifest(repo, e.Manifest)
	if err != nil {
		errs = append(errs, "manifest parse failed: "+err.Error())
	} else {
		if m.ID != e.ID {
			errs = append(errs, "manifest id mismatch: "+m.ID)
		}
		if !allowedManifestModes[m.Mode] {
			errs = append(errs, "invalid mode: "+m.Mode)
		}
		if len(m.Kind) == 0 {
			errs = append(errs, "empty kind")
		}
		for action, c := range m.Commands {
			switch c.Type {
			case "specialty-pwsh":
				if c.Path == "" {
					errs = append(errs, action+": specialty-pwsh path is empty")
				} else if !platform.IsFile(platform.RepoPath(repo, c.Path)) {
					errs = append(errs, action+": missing specialty script: "+c.Path)
				}
			case "builtin-check", "builtin-lifecycle":
				for _, rel := range c.RequiredPaths {
					if !platform.Exists(platform.RepoPath(repo, rel)) {
						errs = append(errs, action+": missing required path: "+rel)
					}
				}
			case "go-composed", "external-command", "builtin-package", "unsupported":
				// Smoke validates manifest shape only; external tools stay out of the hot path.
			default:
				errs = append(errs, action+": unsupported command type in manifest: "+c.Type)
			}
		}
	}
	status := "smoke"
	if len(errs) > 0 {
		status = "failed"
	}
	return SmokeResult{ID: e.ID, OK: len(errs) == 0, Status: status, Manifest: e.Manifest, Errors: errs}
}
