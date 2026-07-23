package cli

import (
	"errors"

	"github.com/JiaxI2/AiCoding/internal/kit"
	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

var (
	typedCommandNamesForPluginProjection    func() []string
	quickstartsForPluginProjection          func() ([]kit.PluginQuickstartRoute, error)
	commandCatalogDigestForPluginProjection func() string
)

func init() {
	typedCommandNamesForPluginProjection = func() []string {
		commands := Catalog().Commands
		names := make([]string, 0, len(commands))
		for _, command := range commands {
			names = append(names, command.Name)
		}
		return names
	}
	quickstartsForPluginProjection = func() ([]kit.PluginQuickstartRoute, error) {
		quickstarts := make([]kit.PluginQuickstartRoute, 0, len(commands.descriptor.Quickstarts))
		for _, form := range commands.descriptor.Quickstarts {
			tokens, err := commands.invocationTokens(form.Command, form.Path, form.Args)
			if err != nil {
				return nil, err
			}
			quickstarts = append(quickstarts, kit.PluginQuickstartRoute{Operation: form.Operation, Command: tokens})
		}
		return quickstarts, nil
	}
	commandCatalogDigestForPluginProjection = func() string { return CatalogSnapshot().Digest() }
}

func loadKitPluginProjection(kitCatalogDigest string) (kit.PluginProjectionPolicy, string, error) {
	adapterCatalog, err := lifecyclecontrol.LoadAdapterCatalogSnapshot()
	if err != nil {
		return kit.PluginProjectionPolicy{}, "", err
	}
	var adapter kit.PluginAdapter
	for _, descriptor := range adapterCatalog.Descriptors() {
		if descriptor.ID != lifecyclecontrol.ScopeKit {
			continue
		}
		adapter = kit.PluginAdapter{
			Scope:      descriptor.ID,
			StateOwner: descriptor.StateOwner,
			Entrypoint: descriptor.Entrypoint,
			Actions:    make([]kit.PluginAdapterAction, 0, len(descriptor.Actions)),
		}
		for _, action := range descriptor.Actions {
			adapter.Actions = append(adapter.Actions, kit.PluginAdapterAction{Name: action.Name, Effect: action.Effect})
		}
		break
	}
	if adapter.Scope == "" {
		return kit.PluginProjectionPolicy{}, "", errors.New("kit lifecycle adapter is missing")
	}

	if typedCommandNamesForPluginProjection == nil || quickstartsForPluginProjection == nil || commandCatalogDigestForPluginProjection == nil {
		return kit.PluginProjectionPolicy{}, "", errors.New("typed command catalog is unavailable")
	}
	typedCommands := typedCommandNamesForPluginProjection()
	quickstarts, err := quickstartsForPluginProjection()
	if err != nil {
		return kit.PluginProjectionPolicy{}, "", err
	}
	input, err := registryobject.NewSnapshot("kit-plugin-view-input", struct {
		KitCatalogDigest     string `json:"kitCatalogDigest"`
		AdapterCatalogDigest string `json:"adapterCatalogDigest"`
		CommandCatalogDigest string `json:"commandCatalogDigest"`
	}{KitCatalogDigest: kitCatalogDigest, AdapterCatalogDigest: adapterCatalog.Digest(), CommandCatalogDigest: commandCatalogDigestForPluginProjection()})
	if err != nil {
		return kit.PluginProjectionPolicy{}, "", err
	}
	return kit.PluginProjectionPolicy{Adapter: adapter, TypedCommands: typedCommands, Quickstarts: quickstarts}, input.Digest(), nil
}
