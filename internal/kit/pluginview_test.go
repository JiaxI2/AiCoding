package kit_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JiaxI2/AiCoding/internal/kit"
	"github.com/JiaxI2/AiCoding/internal/lifecycle"
)

func TestCatalogPluginViewsAreCompleteDetachedAndDeterministic(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := kit.LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := catalog.Select("", true)
	if err != nil {
		t.Fatal(err)
	}
	adapter := testKitPluginAdapter(t)
	policy := kit.PluginProjectionPolicy{
		Adapter: adapter,
		Quickstarts: []kit.PluginQuickstartRoute{{
			Operation: "status",
			Command:   []string{"aicoding", "lifecycle", "status", "--scope", "kit", "--kit", "{kit}", "--json"},
		}},
	}

	first, err := kit.ProjectCatalogPluginViews(repo, selected, policy, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := kit.ProjectCatalogPluginViews(repo, selected, policy, false)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("plugin view JSON changed for the same detached inputs")
	}
	if len(first) != len(selected) {
		t.Fatalf("projected %d kits, want %d", len(first), len(selected))
	}

	actionEffects := map[string]string{}
	for _, action := range adapter.Actions {
		actionEffects[action.Name] = action.Effect
	}
	for index, view := range first {
		if view.State != nil {
			t.Fatalf("state was read without --with-state for %s", view.ID)
		}
		manifest, err := selected[index].Manifest()
		if err != nil {
			t.Fatal(err)
		}
		wantSkills, skillErrors := kit.Skills(manifest)
		if len(skillErrors) > 0 {
			t.Fatalf("parse skills for %s: %v", view.ID, skillErrors)
		}
		if !reflect.DeepEqual(view.Skills, wantSkills) {
			t.Fatalf("%s skills did not reuse the manifest parser", view.ID)
		}
		if view.Quickstart.Purpose != manifest.Description {
			t.Fatalf("%s quickstart purpose did not follow the manifest description", view.ID)
		}
		wantCommand := "aicoding lifecycle status --scope kit --kit " + view.ID + " --json"
		if view.Quickstart.Command != wantCommand {
			t.Fatalf("%s quickstart command = %q, want %q", view.ID, view.Quickstart.Command, wantCommand)
		}
		if len(view.Quickstart.Skills) != len(wantSkills) {
			t.Fatalf("%s quickstart skills = %d, want %d", view.ID, len(view.Quickstart.Skills), len(wantSkills))
		}
		for skillIndex, skill := range wantSkills {
			quickstartSkill := view.Quickstart.Skills[skillIndex]
			if quickstartSkill.ID != skill.ID || quickstartSkill.Description != skill.Description {
				t.Fatalf("%s quickstart skill did not follow manifest skill %s", view.ID, skill.ID)
			}
		}
		for _, operation := range view.Operations {
			if effect, exists := actionEffects[operation.Name]; exists && operation.Effect != effect {
				t.Fatalf("%s operation %s effect = %s, adapter = %s", view.ID, operation.Name, operation.Effect, effect)
			}
		}
	}

	withState, err := kit.ProjectCatalogPluginViews(repo, selected, policy, true)
	if err != nil {
		t.Fatal(err)
	}
	stateJSON, err := json.Marshal(withState)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stateJSON), "installedAt") || strings.Contains(string(stateJSON), "updatedAt") {
		t.Fatal("plugin state leaked nondeterministic timestamps")
	}
	for _, view := range withState {
		if view.State == nil {
			t.Fatalf("--with-state omitted state for %s", view.ID)
		}
	}
}

func TestCatalogPluginViewSelectionKeepsStableUnknownIDError(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := kit.LoadCatalogSnapshot(repo)
	if err != nil {
		t.Fatal(err)
	}
	_, first := catalog.Select("does-not-exist", false)
	_, second := catalog.Select("does-not-exist", false)
	if first == nil || second == nil || first.Error() != "no kit matched" || first.Error() != second.Error() {
		t.Fatalf("unknown kit error is unstable: %v / %v", first, second)
	}
}

func testKitPluginAdapter(t *testing.T) kit.PluginAdapter {
	t.Helper()
	catalog, err := lifecycle.LoadAdapterCatalogSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range catalog.Descriptors() {
		if descriptor.ID != lifecycle.ScopeKit {
			continue
		}
		adapter := kit.PluginAdapter{Scope: descriptor.ID, StateOwner: descriptor.StateOwner, Entrypoint: descriptor.Entrypoint}
		for _, action := range descriptor.Actions {
			adapter.Actions = append(adapter.Actions, kit.PluginAdapterAction{Name: action.Name, Effect: action.Effect})
		}
		return adapter
	}
	t.Fatal("kit lifecycle adapter is missing")
	return kit.PluginAdapter{}
}
