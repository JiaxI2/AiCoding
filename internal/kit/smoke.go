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
	return doctorKits(repo, entries, nil)
}

func DoctorCatalogKits(repo string, snapshots []ManifestSnapshot) []string {
	entries := make([]RegistryKit, 0, len(snapshots))
	manifests := make(map[string]Manifest, len(snapshots))
	for _, snapshot := range snapshots {
		entry := snapshot.Entry()
		entries = append(entries, entry)
		manifest, err := snapshot.Manifest()
		if err == nil {
			manifests[entry.ID] = manifest
		}
	}
	return doctorKits(repo, entries, manifests)
}

func doctorKits(repo string, entries []RegistryKit, manifests map[string]Manifest) []string {
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
		m, resolved := manifests[e.ID]
		if !resolved {
			var err error
			m, err = LoadManifest(repo, e.Manifest)
			if err != nil {
				errs = append(errs, e.ID+": cannot load manifest: "+err.Error())
				continue
			}
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
	inputs := make([]lifecycleInput, 0, len(entries))
	for _, entry := range entries {
		inputs = append(inputs, lifecycleInput{entry: entry})
	}
	return smokeKits(repo, inputs)
}

func SmokeCatalogKits(repo string, snapshots []ManifestSnapshot) []SmokeResult {
	inputs := make([]lifecycleInput, 0, len(snapshots))
	for _, snapshot := range snapshots {
		manifest, err := snapshot.Manifest()
		inputs = append(inputs, lifecycleInput{entry: snapshot.Entry(), manifest: manifest, err: err, resolved: true})
	}
	return smokeKits(repo, inputs)
}

func smokeKits(repo string, inputs []lifecycleInput) []SmokeResult {
	tasks := make([]runner.Task, 0, len(inputs))
	for _, input := range inputs {
		input := input
		tasks = append(tasks, runner.Task{
			ID:    input.entry.ID,
			Group: "kit-smoke",
			Run: func(context.Context) runner.TaskResult {
				return runner.TaskResult{ID: input.entry.ID, OK: true, Data: smokeKit(repo, input)}
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

func smokeKit(repo string, input lifecycleInput) SmokeResult {
	e := input.entry
	errs := []string{}
	if !input.resolved && !platform.IsFile(platform.RepoPath(repo, e.Manifest)) {
		errs = append(errs, "manifest missing")
	}
	m, err := lifecycleManifest(repo, input)
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
