package lifecycle

import (
	"context"
	"errors"
	"sort"
	"strings"

	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
	"github.com/JiaxI2/AiCoding/internal/runner"
)

const (
	EffectRead  = "read"
	EffectWrite = "write"
)

type AdapterAction struct {
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

type AdapterDescriptor struct {
	ID         string          `json:"id"`
	InputKind  string          `json:"inputKind"`
	StateOwner string          `json:"stateOwner"`
	Entrypoint string          `json:"entrypoint"`
	Actions    []AdapterAction `json:"actions"`
}

type AdapterCatalogSnapshot struct {
	object      registryobject.Snapshot
	descriptors []AdapterDescriptor
}

type adapterDefinition struct {
	descriptor AdapterDescriptor
	run        func(context.Context, string, Options) AdapterResult
}

var adapterDefinitions = []adapterDefinition{
	{
		descriptor: AdapterDescriptor{
			ID:         ScopeKit,
			InputKind:  "kit-catalog",
			StateOwner: "kit",
			Entrypoint: "go-static",
			Actions: []AdapterAction{
				{Name: "install", Effect: EffectWrite},
				{Name: "update", Effect: EffectWrite},
				{Name: "uninstall", Effect: EffectWrite},
				{Name: "status", Effect: EffectRead},
				{Name: "doctor", Effect: EffectRead},
				{Name: "verify", Effect: EffectRead},
				{Name: "rollback", Effect: EffectWrite},
			},
		},
		run: func(_ context.Context, repo string, opts Options) AdapterResult {
			return runKitAdapter(repo, opts)
		},
	},
	{
		descriptor: AdapterDescriptor{
			ID:         ScopeMCP,
			InputKind:  "mcp-catalog",
			StateOwner: "mcp",
			Entrypoint: "go-static",
			Actions: []AdapterAction{
				{Name: "install", Effect: EffectWrite},
				{Name: "update", Effect: EffectWrite},
				{Name: "uninstall", Effect: EffectWrite},
				{Name: "status", Effect: EffectRead},
				{Name: "doctor", Effect: EffectRead},
				{Name: "verify", Effect: EffectRead},
			},
		},
		run: runMCPAdapter,
	},
	{
		descriptor: AdapterDescriptor{
			ID:         ScopeRuntimeSkill,
			InputKind:  "runtime-skill-registry",
			StateOwner: "runtime-skill",
			Entrypoint: "bounded-process",
			Actions: []AdapterAction{
				{Name: "install", Effect: EffectWrite},
				{Name: "update", Effect: EffectWrite},
				{Name: "uninstall", Effect: EffectWrite},
				{Name: "status", Effect: EffectRead},
				{Name: "doctor", Effect: EffectRead},
				{Name: "verify", Effect: EffectRead},
			},
		},
		run: func(ctx context.Context, repo string, opts Options) AdapterResult {
			return runRuntimeSkillAdapter(ctx, repo, opts, defaultCommandExecutor)
		},
	},
}

func LoadAdapterCatalogSnapshot() (AdapterCatalogSnapshot, error) {
	descriptors := make([]AdapterDescriptor, 0, len(adapterDefinitions))
	seen := map[string]bool{}
	for _, definition := range adapterDefinitions {
		descriptor := cloneAdapterDescriptor(definition.descriptor)
		if err := validateAdapterDescriptor(descriptor); err != nil {
			return AdapterCatalogSnapshot{}, err
		}
		if seen[descriptor.ID] {
			return AdapterCatalogSnapshot{}, errors.New("duplicate lifecycle adapter: " + descriptor.ID)
		}
		seen[descriptor.ID] = true
		descriptors = append(descriptors, descriptor)
	}
	object, err := registryobject.NewSnapshot("lifecycle-adapter-catalog", descriptors)
	if err != nil {
		return AdapterCatalogSnapshot{}, err
	}
	return AdapterCatalogSnapshot{object: object, descriptors: cloneAdapterDescriptors(descriptors)}, nil
}

func (s AdapterCatalogSnapshot) Digest() string {
	return s.object.Digest()
}

func (s AdapterCatalogSnapshot) Descriptors() []AdapterDescriptor {
	return cloneAdapterDescriptors(s.descriptors)
}

func selectedAdapterDefinitions(scope, action string) ([]adapterDefinition, error) {
	selected := []adapterDefinition{}
	for _, definition := range adapterDefinitions {
		if scope != ScopeAll && definition.descriptor.ID != scope {
			continue
		}
		if !adapterSupports(definition.descriptor, action) {
			return nil, errors.New(definition.descriptor.ID + " lifecycle adapter does not support action: " + action)
		}
		selected = append(selected, definition)
	}
	if len(selected) == 0 {
		return nil, errors.New("unsupported lifecycle scope: " + scope)
	}
	return selected, nil
}

func buildExecutionPlan(repo string, opts Options, catalogDigest string, definitions []adapterDefinition, execute commandExecutor) (runner.ExecutionPlan, error) {
	tasks := make([]runner.Task, 0, len(definitions))
	for _, definition := range definitions {
		definition := definition
		runAdapter := definition.run
		if definition.descriptor.ID == ScopeRuntimeSkill && execute != nil {
			runAdapter = func(ctx context.Context, repo string, opts Options) AdapterResult {
				return runRuntimeSkillAdapter(ctx, repo, opts, execute)
			}
		}
		tasks = append(tasks, runner.Task{
			ID:         definition.descriptor.ID,
			Action:     opts.Action,
			Group:      "lifecycle",
			Parameters: lifecycleParameters(opts, definition.descriptor.ID, catalogDigest),
			Critical:   true,
			Run: func(ctx context.Context) runner.TaskResult {
				adapter := runAdapter(ctx, repo, opts)
				return runner.TaskResult{
					ID:       adapter.ID,
					Group:    "lifecycle",
					OK:       adapter.OK,
					Warnings: append([]string{}, adapter.Warnings...),
					Errors:   append([]string{}, adapter.Errors...),
					Data:     adapter,
				}
			},
		})
	}
	return runner.NewExecutionPlan(tasks...)
}

func lifecycleParameters(opts Options, scope, catalogDigest string) map[string]string {
	parameters := map[string]string{
		"adapterCatalogDigest": catalogDigest,
		"selection":            stableSelection(opts, scope),
		"dryRun":               boolString(opts.DryRun),
	}
	if opts.Action == "verify" {
		parameters["verifyProfile"] = opts.VerifyProfile
	}
	if scope == ScopeMCP {
		if opts.Action == "verify" {
			parameters["includeConfigured"] = boolString(opts.IncludeConfigured)
		}
		if opts.CodexConfig != "" {
			parameters["codexConfig"] = opts.CodexConfig
		}
	}
	if scope == ScopeRuntimeSkill {
		parameters["migrateUnmanaged"] = boolString(opts.MigrateUnmanaged)
	}
	return parameters
}

func validateAdapterDescriptor(descriptor AdapterDescriptor) error {
	if strings.TrimSpace(descriptor.ID) == "" || strings.TrimSpace(descriptor.InputKind) == "" ||
		strings.TrimSpace(descriptor.StateOwner) == "" || strings.TrimSpace(descriptor.Entrypoint) == "" {
		return errors.New("lifecycle adapter id, inputKind, stateOwner, and entrypoint are required")
	}
	if len(descriptor.Actions) == 0 {
		return errors.New(descriptor.ID + " lifecycle adapter requires actions")
	}
	seen := map[string]bool{}
	for _, action := range descriptor.Actions {
		if action.Name == "" || (action.Effect != EffectRead && action.Effect != EffectWrite) {
			return errors.New(descriptor.ID + " lifecycle adapter action requires a name and read/write effect")
		}
		if seen[action.Name] {
			return errors.New(descriptor.ID + " lifecycle adapter has duplicate action: " + action.Name)
		}
		seen[action.Name] = true
	}
	return nil
}

func adapterSupports(descriptor AdapterDescriptor, action string) bool {
	for _, candidate := range descriptor.Actions {
		if candidate.Name == action {
			return true
		}
	}
	return false
}

func stableSelection(opts Options, scope string) string {
	switch scope {
	case ScopeKit:
		if opts.All || opts.Scope == ScopeAll {
			return "all"
		}
		return opts.KitID
	case ScopeMCP:
		if opts.All || opts.Scope == ScopeAll {
			return "all"
		}
		return opts.ComponentID
	case ScopeRuntimeSkill:
		parts := []string{opts.RuntimeProfile, opts.RuntimeSkill, opts.StandaloneRoot}
		return strings.Join(parts, ":")
	default:
		return scope
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func cloneAdapterDescriptors(descriptors []AdapterDescriptor) []AdapterDescriptor {
	out := make([]AdapterDescriptor, len(descriptors))
	for index, descriptor := range descriptors {
		out[index] = cloneAdapterDescriptor(descriptor)
	}
	return out
}

func cloneAdapterDescriptor(descriptor AdapterDescriptor) AdapterDescriptor {
	descriptor.Actions = append([]AdapterAction{}, descriptor.Actions...)
	sort.SliceStable(descriptor.Actions, func(i, j int) bool {
		return descriptor.Actions[i].Name < descriptor.Actions[j].Name
	})
	return descriptor
}
