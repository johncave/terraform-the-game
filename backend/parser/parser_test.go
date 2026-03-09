package parser_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johncave/terraform-the-game/models"
	"github.com/johncave/terraform-the-game/parser"
)

// genesisYAML is the standard starter factory config used in tests.
const genesisYAML = `
resources:
  miner:
    iron_extractor:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.iron_processor.inputs.iron_ore"
  smelter:
    iron_processor:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.plate_press.inputs.iron_ingot"
  builder:
    plate_press:
      recipe: "iron_plate"
      outputs:
        - target: "inventory"
`

func newTestState() *models.GameState {
	return models.NewGameState(uuid.New())
}

// TestParseGenesisYAML verifies that the genesis config parses correctly.
func TestParseGenesisYAML(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", result.Errors)
	}
	if len(result.Machines) != 3 {
		t.Fatalf("expected 3 machines, got %d", len(result.Machines))
	}

	types := map[models.MachineType]bool{}
	for _, m := range result.Machines {
		types[m.Type] = true
	}
	for _, expected := range []models.MachineType{models.MachineTypeMiner, models.MachineTypeSmelter, models.MachineTypeBuilder} {
		if !types[expected] {
			t.Errorf("expected machine type %q to be present", expected)
		}
	}
}

// TestValidateGenesisYAML verifies that the genesis config validates against a fresh state.
func TestValidateGenesisYAML(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	state := newTestState()
	errors := parser.Validate(result, state)
	if len(errors) > 0 {
		t.Fatalf("validation failed with errors: %v", errors)
	}
}

// TestApplyGenesisYAML verifies that applying the genesis config deducts costs and sets BuiltAt.
func TestApplyGenesisYAML(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	state := newTestState()
	initialPlates := state.Inventory.Items[models.IronPlate]

	parser.Apply(result, state)

	// Costs: smelter=3, builder=5 (miner=free)
	expectedCost := 3 + 5
	gotPlates := state.Inventory.Items[models.IronPlate]
	if gotPlates != initialPlates-expectedCost {
		t.Errorf("expected %d iron_plate after apply, got %d", initialPlates-expectedCost, gotPlates)
	}

	// All machines should have BuiltAt set
	for _, m := range result.Machines {
		if m.BuiltAt == nil {
			t.Errorf("machine %s.%s should have BuiltAt set", m.Type, m.ID)
		}
	}
}

// TestChangingBuilderRecipeIronBlockToIronPlate verifies that after applying the genesis
// config, changing the builder recipe from iron_plate to iron_block works correctly.
func TestChangingBuilderRecipeIronBlockToIronPlate(t *testing.T) {
	// 1. Apply genesis config
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse genesis error: %v", err)
	}
	state := newTestState()
	errors := parser.Validate(result, state)
	if len(errors) > 0 {
		t.Fatalf("genesis validation failed: %v", errors)
	}
	parser.Apply(result, state)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// 2. Change builder recipe to iron_block
	const updatedYAML = `
resources:
  miner:
    iron_extractor:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.iron_processor.inputs.iron_ore"
  smelter:
    iron_processor:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.plate_press.inputs.iron_ingot"
  builder:
    plate_press:
      recipe: "iron_block"
      outputs:
        - target: "inventory"
`
	result2, err := parser.Parse("genesis", updatedYAML)
	if err != nil {
		t.Fatalf("parse updated error: %v", err)
	}
	errors2 := parser.Validate(result2, state)
	if len(errors2) > 0 {
		t.Fatalf("updated validation failed: %v", errors2)
	}

	// Apply (machines already exist, just update recipe)
	parser.Apply(result2, state)
	for _, m := range result2.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// Verify the builder now has recipe iron_block
	builder, exists := state.Machines["builder.plate_press"]
	if !exists {
		t.Fatal("builder.plate_press not found in state")
	}
	if builder.Recipe != "iron_block" {
		t.Errorf("expected recipe iron_block, got %q", builder.Recipe)
	}
	// It should still have BuiltAt from the original apply
	if builder.BuiltAt == nil {
		t.Error("BuiltAt should still be set after recipe update")
	}
}

// TestCountFieldExpansion verifies that `count: N` expands to N machines.
func TestCountFieldExpansion(t *testing.T) {
	const yaml = `
resources:
  miner:
    extractor:
      count: 3
      node_id: "node_alpha"
      outputs:
        - target: "inventory"
`
	result, err := parser.Parse("test", yaml)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Machines) != 3 {
		t.Fatalf("expected 3 machines from count:3, got %d", len(result.Machines))
	}
	for i, m := range result.Machines {
		expected := models.MachineType("miner")
		if m.Type != expected {
			t.Errorf("machine %d: expected type miner, got %s", i, m.Type)
		}
	}
	// Names should be extractor_1, extractor_2, extractor_3
	names := map[string]bool{}
	for _, m := range result.Machines {
		names[m.ID] = true
	}
	for _, n := range []string{"extractor_1", "extractor_2", "extractor_3"} {
		if !names[n] {
			t.Errorf("expected machine ID %q", n)
		}
	}
}

// TestDiffNewFactory verifies Diff returns all machines as to_add for a fresh factory.
func TestDiffNewFactory(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	state := newTestState()
	diff := parser.Diff("genesis", result, state)

	if len(diff.ToAdd) != 3 {
		t.Errorf("expected 3 to_add, got %d: %v", len(diff.ToAdd), diff.ToAdd)
	}
	if len(diff.ToChange) != 0 {
		t.Errorf("expected 0 to_change, got %d", len(diff.ToChange))
	}
	if len(diff.ToDestroy) != 0 {
		t.Errorf("expected 0 to_destroy, got %d", len(diff.ToDestroy))
	}
}

// TestDiffChangedRecipe verifies Diff detects a changed recipe.
func TestDiffChangedRecipe(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	state := newTestState()
	parser.Apply(result, state)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// YAML with changed builder recipe
	const changedYAML = `
resources:
  miner:
    iron_extractor:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.iron_processor.inputs.iron_ore"
  smelter:
    iron_processor:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.plate_press.inputs.iron_ingot"
  builder:
    plate_press:
      recipe: "iron_block"
      outputs:
        - target: "inventory"
`
	result2, err := parser.Parse("genesis", changedYAML)
	if err != nil {
		t.Fatalf("parse changed error: %v", err)
	}
	diff := parser.Diff("genesis", result2, state)

	if len(diff.ToAdd) != 0 {
		t.Errorf("expected 0 to_add, got %d", len(diff.ToAdd))
	}
	if len(diff.ToChange) != 1 {
		t.Errorf("expected 1 to_change (builder), got %d: %v", len(diff.ToChange), diff.ToChange)
	}
	if len(diff.ToDestroy) != 0 {
		t.Errorf("expected 0 to_destroy, got %d", len(diff.ToDestroy))
	}
}

// TestDiffRemovedMachine verifies Diff detects removed machines.
func TestDiffRemovedMachine(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	state := newTestState()
	parser.Apply(result, state)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// YAML without the builder
	const noBuilderYAML = `
resources:
  miner:
    iron_extractor:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.iron_processor.inputs.iron_ore"
  smelter:
    iron_processor:
      recipe: "iron_ingot"
      outputs:
        - target: "inventory"
`
	result2, err := parser.Parse("genesis", noBuilderYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	diff := parser.Diff("genesis", result2, state)

	if len(diff.ToDestroy) != 1 {
		t.Errorf("expected 1 to_destroy (builder), got %d: %v", len(diff.ToDestroy), diff.ToDestroy)
	}
}

// TestPowerUsageMWIsSet verifies parsed machines have PowerUsageMW populated.
func TestPowerUsageMWIsSet(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, m := range result.Machines {
		expected, ok := models.MachinePowerUsageMW[m.Type]
		if !ok {
			t.Errorf("machine type %q not in MachinePowerUsageMW", m.Type)
			continue
		}
		if m.PowerUsageMW != expected {
			t.Errorf("machine %s.%s: expected PowerUsageMW=%d, got %d", m.Type, m.ID, expected, m.PowerUsageMW)
		}
	}
}

// TestValidationInsufficientInventory verifies that applying an expensive config fails
// when inventory is insufficient.
func TestValidationInsufficientInventory(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	state := newTestState()
	// Remove all iron plates
	state.Inventory.Items[models.IronPlate] = 0

	errors := parser.Validate(result, state)
	if len(errors) == 0 {
		t.Error("expected validation errors due to insufficient inventory")
	}
}

// TestApplyDoesNotDuplicateBuiltAt verifies that applying the same factory twice
// doesn't reset BuiltAt on existing machines.
func TestApplyDoesNotDuplicateBuiltAt(t *testing.T) {
	result, err := parser.Parse("genesis", genesisYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	state := newTestState()
	parser.Apply(result, state)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// Capture original BuiltAt for the miner
	origBuiltAt := *state.Machines["miner.iron_extractor"].BuiltAt

	// Apply again with same YAML
	result2, _ := parser.Parse("genesis", genesisYAML)
	parser.Apply(result2, state)

	// BuiltAt should not have changed for existing machine
	newBuiltAt := *state.Machines["miner.iron_extractor"].BuiltAt
	if !origBuiltAt.Equal(newBuiltAt) {
		t.Errorf("BuiltAt changed on re-apply: was %v, now %v", origBuiltAt, newBuiltAt)
	}
}

// TestUnknownMachineTypeErrors verifies invalid machine types produce errors.
func TestUnknownMachineTypeErrors(t *testing.T) {
	const bad = `
resources:
  turret:
    laser_1:
      recipe: "iron_plate"
`
	result, err := parser.Parse("test", bad)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for unknown machine type")
	}
}

// TestMinerMissingNodeID verifies miners without node_id produce errors.
func TestMinerMissingNodeID(t *testing.T) {
	const bad = `
resources:
  miner:
    extractor:
      outputs:
        - target: "inventory"
`
	result, err := parser.Parse("test", bad)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for miner missing node_id")
	}
}

// TestIronBlockRecipeCosts4Ingots verifies that the iron_block recipe now requires 4 ingots.
func TestIronBlockRecipeCosts4Ingots(t *testing.T) {
	recipe, ok := models.Recipes["iron_block"]
	if !ok {
		t.Fatal("iron_block recipe not found")
	}
	got := recipe.Inputs[models.IronIngot]
	if got != 4 {
		t.Errorf("iron_block should cost 4 iron_ingot, got %d", got)
	}
}

// TestIronSheetRecipeCosts1Ingot verifies the new iron_sheet recipe.
func TestIronSheetRecipeCosts1Ingot(t *testing.T) {
	recipe, ok := models.Recipes["iron_sheet"]
	if !ok {
		t.Fatal("iron_sheet recipe not found")
	}
	if got := recipe.Inputs[models.IronIngot]; got != 1 {
		t.Errorf("iron_sheet should cost 1 iron_ingot, got %d", got)
	}
	if _, ok := recipe.Outputs[models.IronSheet]; !ok {
		t.Error("iron_sheet recipe should output iron_sheet")
	}
}

// TestInputOverrideInYAML verifies that `inputs:` in the YAML sets RecipeInputs on the machine.
func TestInputOverrideInYAML(t *testing.T) {
	const yaml = `
resources:
  builder:
    block_press:
      recipe: "iron_block"
      inputs:
        iron_ingot: 2
      outputs:
        - target: "inventory"
`
	result, err := parser.Parse("test", yaml)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(result.Machines))
	}
	m := result.Machines[0]
	if m.RecipeInputs == nil {
		t.Fatal("expected RecipeInputs to be set")
	}
	if got := m.RecipeInputs[models.IronIngot]; got != 2 {
		t.Errorf("expected RecipeInputs[iron_ingot]=2, got %d", got)
	}
}

// TestInputOverrideIsUsedInProcessing and TestDefaultRecipeCostIsUsedWhenNoOverride
// both require engine.Tick and live in backend/engine/tick_test.go.

// TestInputOverrideZeroIsRejected verifies that a zero input override is rejected.
func TestInputOverrideZeroIsRejected(t *testing.T) {
	const yaml = `
resources:
  builder:
    block_press:
      recipe: "iron_block"
      inputs:
        iron_ingot: 0
      outputs:
        - target: "inventory"
`
	result, err := parser.Parse("test", yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected validation error for zero input quantity")
	}
}

// TestInputOverrideDiffDetection verifies that changing input overrides shows up in Diff.
func TestInputOverrideDiffDetection(t *testing.T) {
	const yaml1 = `
resources:
  builder:
    block_press:
      recipe: "iron_block"
      inputs:
        iron_ingot: 2
      outputs:
        - target: "inventory"
`
	state := newTestState()
	r1, _ := parser.Parse("test", yaml1)
	parser.Apply(r1, state)
	for _, m := range r1.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// Change override from 2 → 3
	const yaml2 = `
resources:
  builder:
    block_press:
      recipe: "iron_block"
      inputs:
        iron_ingot: 3
      outputs:
        - target: "inventory"
`
	r2, _ := parser.Parse("test", yaml2)
	diff := parser.Diff("test", r2, state)

	if len(diff.ToChange) != 1 {
		t.Errorf("expected 1 to_change for input override change, got %d: %v", len(diff.ToChange), diff.ToChange)
	}
	if len(diff.ToAdd) != 0 || len(diff.ToDestroy) != 0 {
		t.Errorf("expected no to_add/to_destroy, got add=%v destroy=%v", diff.ToAdd, diff.ToDestroy)
	}
}

// TestDefaultRecipeCostIsUsedWhenNoOverride lives in backend/engine/tick_test.go.

// TestApplyUpdatesExistingMachineRecipe verifies the Apply function updates the
// recipe of an already-built machine when the YAML changes it.
func TestApplyUpdatesExistingMachineRecipe(t *testing.T) {
	// Build initial factory
	result, _ := parser.Parse("genesis", genesisYAML)
	state := newTestState()
	parser.Apply(result, state)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		// Simulate machine being built (past BuiltAt)
		past := time.Now().Add(-5 * time.Minute)
		m.BuiltAt = &past
		state.Machines[key] = m
	}

	const updatedYAML = `
resources:
  miner:
    iron_extractor:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.iron_processor.inputs.iron_ore"
  smelter:
    iron_processor:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.plate_press.inputs.iron_ingot"
  builder:
    plate_press:
      recipe: "iron_block"
      outputs:
        - target: "inventory"
`
	result2, _ := parser.Parse("genesis", updatedYAML)
	parser.Apply(result2, state)
	// Apply carries BuiltAt over to result machines for existing ones
	for _, m := range result2.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	builder := state.Machines["builder.plate_press"]
	if builder == nil {
		t.Fatal("builder.plate_press not found")
	}
	if builder.Recipe != "iron_block" {
		t.Errorf("expected recipe iron_block after update, got %q", builder.Recipe)
	}
	// Should have an input slot for iron_ingot (the new recipe's input)
	if _, ok := builder.InputSlots["iron_ingot"]; !ok {
		t.Error("expected iron_ingot input slot after recipe change to iron_block")
	}
}
