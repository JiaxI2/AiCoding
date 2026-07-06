package kit

import "github.com/JiaxI2/AiCoding/internal/platform"

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
		if m.Mode != "script-adapter" && m.Mode != "declarative" {
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
	results := []SmokeResult{}
	for _, e := range entries {
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
			if m.Mode != "script-adapter" && m.Mode != "declarative" {
				errs = append(errs, "invalid mode: "+m.Mode)
			}
			if len(m.Kind) == 0 {
				errs = append(errs, "empty kind")
			}
			for action, c := range m.Commands {
				switch c.Type {
				case "powershell-script":
					if c.Path == "" {
						errs = append(errs, action+": powershell-script path is empty")
					} else if !platform.IsFile(platform.RepoPath(repo, c.Path)) {
						errs = append(errs, action+": missing command script: "+c.Path)
					}
				case "builtin-check":
					for _, rel := range c.RequiredPaths {
						if !platform.Exists(platform.RepoPath(repo, rel)) {
							errs = append(errs, action+": missing required path: "+rel)
						}
					}
				case "composed", "external-command", "builtin-package", "unsupported":
					// Smoke mode validates manifest shape only; external tools stay out of the hot path.
				default:
					errs = append(errs, action+": unsupported command type in manifest: "+c.Type)
				}
			}
		}
		status := "smoke"
		if len(errs) > 0 {
			status = "failed"
		}
		results = append(results, SmokeResult{ID: e.ID, OK: len(errs) == 0, Status: status, Manifest: e.Manifest, Errors: errs})
	}
	return results
}
