package kit

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type PluginView struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Quickstart  PluginQuickstart  `json:"quickstart"`
	Identity    PluginIdentity    `json:"identity"`
	Skills      []SkillEntry      `json:"skills"`
	Operations  []PluginOperation `json:"operations"`
	Lifecycle   []PluginLifecycle `json:"lifecycleActions"`
	Workflows   []PluginWorkflow  `json:"workflows"`
	Source      PluginSource      `json:"source"`
	State       *PluginState      `json:"state,omitempty"`
}

type PluginQuickstart struct {
	Purpose string                  `json:"purpose"`
	Command string                  `json:"command"`
	Skills  []PluginQuickstartSkill `json:"skills"`
}

type PluginQuickstartSkill struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type PluginIdentity struct {
	Enabled bool     `json:"enabled"`
	Order   int      `json:"order"`
	Version string   `json:"version"`
	Kind    []string `json:"kind"`
	Mode    string   `json:"mode"`
	Trust   string   `json:"trust,omitempty"`
}

type PluginOperation struct {
	Name       string `json:"name"`
	Effect     string `json:"effect"`
	StateOwner string `json:"stateOwner"`
	Entrypoint string `json:"entrypoint"`
}

type PluginLifecycle struct {
	Name       string `json:"name"`
	Effect     string `json:"effect"`
	Scope      string `json:"scope"`
	StateOwner string `json:"stateOwner"`
	Entrypoint string `json:"entrypoint"`
}

type PluginWorkflow struct {
	Skill    string   `json:"skill"`
	Path     string   `json:"path"`
	Sections []string `json:"sections"`
}

type PluginSource struct {
	Manifest string        `json:"manifest"`
	Pin      *PinnedSource `json:"pin,omitempty"`
	Identity string        `json:"identity,omitempty"`
}

type PluginState struct {
	KitID          string `json:"kitId"`
	Version        string `json:"version"`
	Action         string `json:"action,omitempty"`
	Installed      bool   `json:"installed"`
	SourceIdentity string `json:"sourceIdentity,omitempty"`
}

type PluginAdapter struct {
	Scope      string                `json:"scope"`
	StateOwner string                `json:"stateOwner"`
	Entrypoint string                `json:"entrypoint"`
	Actions    []PluginAdapterAction `json:"actions"`
}

type PluginAdapterAction struct {
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

type PluginProjectionPolicy struct {
	Adapter       PluginAdapter
	TypedCommands []string
}

var manifestCommandEffect = map[string]string{
	"doctor":        "read",
	"export":        "write",
	"install":       "write",
	"skills":        "read",
	"status":        "read",
	"test":          "read",
	"uninstall":     "write",
	"update":        "write",
	"verify":        "read",
	"verify-skills": "read",
}

func ProjectCatalogPluginViews(repo string, snapshots []ManifestSnapshot, adapter PluginAdapter, withState bool) ([]PluginView, error) {
	if err := validatePluginAdapter(adapter); err != nil {
		return nil, err
	}
	identities := CatalogKitViews(snapshots)
	if len(identities) != len(snapshots) {
		return nil, errors.New("kit identity projection count mismatch")
	}

	views := make([]PluginView, 0, len(snapshots))
	for index, snapshot := range snapshots {
		manifest, err := snapshot.Manifest()
		if err != nil {
			return nil, fmt.Errorf("%s: decode manifest: %w", snapshot.Entry().ID, err)
		}
		skills, skillErrors := Skills(manifest)
		if len(skillErrors) > 0 {
			return nil, fmt.Errorf("%s: parse skills: %s", snapshot.Entry().ID, strings.Join(skillErrors, "; "))
		}
		operations, err := projectPluginOperations(manifest.Commands, adapter)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", snapshot.Entry().ID, err)
		}
		workflows, err := projectPluginWorkflows(repo, manifest, skills)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", snapshot.Entry().ID, err)
		}

		identity := identities[index]
		view := PluginView{
			ID:          identity.ID,
			Name:        identity.Name,
			Description: manifest.Description,
			Quickstart:  projectPluginQuickstart(identity.ID, manifest.Description, operations, skills),
			Identity: PluginIdentity{
				Enabled: identity.Enabled,
				Order:   identity.Order,
				Version: identity.Version,
				Kind:    append([]string{}, identity.Kind...),
				Mode:    identity.Mode,
				Trust:   pluginTrustLevel(manifest),
			},
			Skills:     cloneSkillEntries(skills),
			Operations: operations,
			Lifecycle:  projectPluginLifecycle(adapter),
			Workflows:  workflows,
			Source:     PluginSource{Manifest: identity.Manifest, Pin: clonePinnedSource(manifest.Source), Identity: identity.SourceIdentity},
		}
		if withState {
			state, err := projectPluginState(repo, snapshot.Entry(), manifest)
			if err != nil {
				return nil, fmt.Errorf("%s: read state: %w", snapshot.Entry().ID, err)
			}
			view.State = state
		}
		views = append(views, view)
	}
	return views, nil
}

func projectPluginQuickstart(id, description string, operations []PluginOperation, skills []SkillEntry) PluginQuickstart {
	quickstart := PluginQuickstart{
		Purpose: strings.TrimSpace(description),
		Skills:  make([]PluginQuickstartSkill, 0, len(skills)),
	}
	for _, operation := range operations {
		if operation.Effect == "read" {
			quickstart.Command = pluginQuickstartCommand(id, operation.Name)
			break
		}
	}
	for _, skill := range skills {
		quickstart.Skills = append(quickstart.Skills, PluginQuickstartSkill{
			ID:          skill.ID,
			Description: strings.TrimSpace(skill.Description),
		})
	}
	return quickstart
}

func pluginQuickstartCommand(id, operation string) string {
	selector := " --scope kit --kit " + id + " --json"
	switch operation {
	case "doctor", "status", "verify":
		return "aicoding lifecycle " + operation + selector
	case "test":
		return "aicoding kit test --kit " + id + " --profile Smoke --json"
	case "skills", "verify-skills":
		return "aicoding skill verify --kit " + id + " --profile Smoke --json"
	default:
		return ""
	}
}

func cloneSkillEntries(skills []SkillEntry) []SkillEntry {
	if skills == nil {
		return nil
	}
	out := make([]SkillEntry, len(skills))
	for index, skill := range skills {
		skill.Tags = append([]string(nil), skill.Tags...)
		out[index] = skill
	}
	return out
}

func Skills(manifest Manifest) ([]SkillEntry, []string) {
	return parseSkillEntries(manifest)
}

func projectPluginOperations(commands map[string]CommandDef, adapter PluginAdapter) ([]PluginOperation, error) {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	operations := make([]PluginOperation, 0, len(names))
	for _, name := range names {
		effect, ok := pluginOperationEffect(name, adapter.Actions)
		if !ok {
			return nil, fmt.Errorf("manifest command has no read/write effect: %s", name)
		}
		if effect == "write" && strings.TrimSpace(adapter.StateOwner) == "" {
			return nil, fmt.Errorf("write operation has no state owner: %s", name)
		}
		operations = append(operations, PluginOperation{
			Name:       name,
			Effect:     effect,
			StateOwner: adapter.StateOwner,
			Entrypoint: adapter.Entrypoint,
		})
	}
	return operations, nil
}

func pluginOperationEffect(name string, actions []PluginAdapterAction) (string, bool) {
	for _, action := range actions {
		if action.Name == name {
			return action.Effect, action.Effect == "read" || action.Effect == "write"
		}
	}
	effect, ok := manifestCommandEffect[name]
	return effect, ok
}

func projectPluginLifecycle(adapter PluginAdapter) []PluginLifecycle {
	actions := append([]PluginAdapterAction{}, adapter.Actions...)
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	items := make([]PluginLifecycle, 0, len(actions))
	for _, action := range actions {
		items = append(items, PluginLifecycle{
			Name:       action.Name,
			Effect:     action.Effect,
			Scope:      adapter.Scope,
			StateOwner: adapter.StateOwner,
			Entrypoint: adapter.Entrypoint,
		})
	}
	return items
}

func projectPluginWorkflows(repo string, manifest Manifest, skills []SkillEntry) ([]PluginWorkflow, error) {
	workflows := make([]PluginWorkflow, 0, len(skills))
	for _, skill := range skills {
		document, errs := readManifestSkillDocument(repo, manifest, skill.Path)
		if len(errs) > 0 {
			return nil, fmt.Errorf("%s: %s", skill.ID, strings.Join(errs, "; "))
		}
		workflows = append(workflows, PluginWorkflow{
			Skill:    skill.ID,
			Path:     skill.Path,
			Sections: append([]string{}, document.Sections...),
		})
	}
	sort.Slice(workflows, func(i, j int) bool { return workflows[i].Skill < workflows[j].Skill })
	return workflows, nil
}

func projectPluginState(repo string, entry RegistryKit, manifest Manifest) (*PluginState, error) {
	state := &PluginState{KitID: entry.ID, Version: manifest.Version, Installed: false}
	installed, err := readInstallState(statePath(repo, manifest, entry.ID))
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	state.KitID = installed.KitID
	state.Version = installed.Version
	state.Action = installed.Action
	state.Installed = true
	state.SourceIdentity = installed.SourceIdentity
	return state, nil
}

func validatePluginAdapter(adapter PluginAdapter) error {
	if strings.TrimSpace(adapter.Scope) == "" || strings.TrimSpace(adapter.StateOwner) == "" || strings.TrimSpace(adapter.Entrypoint) == "" {
		return errors.New("plugin adapter scope, stateOwner, and entrypoint are required")
	}
	seen := map[string]bool{}
	for _, action := range adapter.Actions {
		if strings.TrimSpace(action.Name) == "" || (action.Effect != "read" && action.Effect != "write") {
			return errors.New("plugin adapter actions require a name and read/write effect")
		}
		if seen[action.Name] {
			return fmt.Errorf("duplicate plugin adapter action: %s", action.Name)
		}
		seen[action.Name] = true
	}
	return nil
}

func pluginTrustLevel(manifest Manifest) string {
	value, _ := manifest.Trust["level"].(string)
	return value
}
