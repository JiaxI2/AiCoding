package profile

type Profile struct {
	ID                      string   `json:"id"`
	Domain                  string   `json:"domain"`
	AllowedControlModes     []string `json:"allowedControlModes"`
	RequiredGates           []string `json:"requiredGates"`
	AdvisoryGates           []string `json:"advisoryGates"`
	DefaultHumanCheckpoints []string `json:"defaultHumanCheckpoints"`
}

func Builtins() []Profile {
	return []Profile{
		{ID: "project-development", Domain: "project-development", AllowedControlModes: []string{"turn", "goal"}, RequiredGates: []string{"scope", "project-build", "project-test"}, DefaultHumanCheckpoints: []string{"architecture-change", "merge"}},
		{ID: "aicoding-repository-maintenance", Domain: "repository-maintenance", AllowedControlModes: []string{"turn", "goal", "time", "proactive"}, RequiredGates: []string{"governance-dependencies", "governance-layout", "verify-smoke"}, AdvisoryGates: []string{"docsync"}, DefaultHumanCheckpoints: []string{"architecture-unfreeze", "submodule-pin-change", "merge", "release"}},
	}
}
