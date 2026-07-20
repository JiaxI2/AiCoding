package cli

import (
	"errors"

	"github.com/JiaxI2/AiCoding/internal/kit"
	lifecyclecontrol "github.com/JiaxI2/AiCoding/internal/lifecycle"
	registryobject "github.com/JiaxI2/AiCoding/internal/registry"
)

var typedCommandNamesForPluginProjection func() []string

func init() {
	typedCommandNamesForPluginProjection = func() []string {
		commands := Catalog().Commands
		names := make([]string, 0, len(commands))
		for _, command := range commands {
			names = append(names, command.Name)
		}
		return names
	}
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

	if typedCommandNamesForPluginProjection == nil {
		return kit.PluginProjectionPolicy{}, "", errors.New("typed command catalog is unavailable")
	}
	typedCommands := typedCommandNamesForPluginProjection()
	input, err := registryobject.NewSnapshot("kit-plugin-view-input", struct {
		KitCatalogDigest     string `json:"kitCatalogDigest"`
		AdapterCatalogDigest string `json:"adapterCatalogDigest"`
	}{KitCatalogDigest: kitCatalogDigest, AdapterCatalogDigest: adapterCatalog.Digest()})
	if err != nil {
		return kit.PluginProjectionPolicy{}, "", err
	}
	return kit.PluginProjectionPolicy{Adapter: adapter, TypedCommands: typedCommands}, input.Digest(), nil
}
