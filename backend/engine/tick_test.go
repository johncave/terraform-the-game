package engine_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johncave/terraform-the-game/engine"
	"github.com/johncave/terraform-the-game/models"
	"github.com/johncave/terraform-the-game/parser"
)

func newState() *models.GameState {
	return models.NewGameState(uuid.New())
}

// buildGenesisFactory applies the genesis factory to the given state and marks all
// machines as already built so they tick immediately.
func buildGenesisFactory(t *testing.T, state *models.GameState) {
	t.Helper()
	const yaml = `
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
	result, err := parser.Parse("genesis", yaml)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if errs := parser.Validate(result, state); len(errs) > 0 {
		t.Fatalf("validation errors: %v", errs)
	}
	parser.Apply(result, state)
	past := time.Now().Add(-5 * time.Minute)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		m.BuiltAt = &past
		state.Machines[key] = m
	}
}

// TestTickProduces verifies that a complete miner→smelter→builder chain
// produces iron_plate in inventory after a 2-minute elapsed time.
func TestTickProduces(t *testing.T) {
	state := newState()
	buildGenesisFactory(t, state)

	// Tick 2 minutes of elapsed time
	engine.Tick(state, 2*time.Minute)

	plates := state.Inventory.Items[models.IronPlate]
	if plates == 0 {
		t.Error("expected iron_plate in inventory after 2-minute tick")
	}
}

// TestChangingBuilderRecipeFromPlateToBlock applies the genesis config, then
// re-applies with iron_block recipe and verifies iron_block is produced.
func TestChangingBuilderRecipeFromPlateToBlock(t *testing.T) {
	state := newState()
	buildGenesisFactory(t, state)

	// Tick enough to fill smelter's output with iron_ingot
	engine.Tick(state, 2*time.Minute)

	// Now update builder recipe to iron_block
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
	result, err := parser.Parse("genesis", updatedYAML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if errs := parser.Validate(result, state); len(errs) > 0 {
		t.Fatalf("validation errors after recipe change: %v", errs)
	}
	// Apply carries BuiltAt over to result machines for existing ones
	parser.Apply(result, state)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// Tick again
	engine.Tick(state, 2*time.Minute)

	blocks := state.Inventory.Items[models.IronBlock]
	if blocks == 0 {
		t.Error("expected iron_block in inventory after changing builder recipe")
	}
	// Should not have produced MORE plates since recipe changed
	// (some plates may have been produced before the recipe change)
}

// TestBasePowerAvailable verifies a new game starts with BasePowerMW.
func TestBasePowerAvailable(t *testing.T) {
	state := newState()
	avail := state.TotalPowerAvailableMW()
	if avail != models.BasePowerMW {
		t.Errorf("expected BasePowerMW=%d, got %d", models.BasePowerMW, avail)
	}
}

// TestSolarPanelIncreasePower verifies that producing solar panels increases available power.
func TestSolarPanelIncreasePower(t *testing.T) {
	state := newState()
	before := state.TotalPowerAvailableMW()

	// Simulate producing 2 solar panels
	state.PowerGeneration += 2

	after := state.TotalPowerAvailableMW()
	expected := before + 2*models.SolarPanelGenMW
	if after != expected {
		t.Errorf("expected power %d after 2 solar panels, got %d", expected, after)
	}
}

// TestPowerTrip verifies that power tripping stops machine production.
func TestPowerTrip(t *testing.T) {
	state := newState()
	buildGenesisFactory(t, state)

	// Manually trip the grid
	state.PowerTripped = true

	initialPlates := state.Inventory.Items[models.IronPlate]
	engine.Tick(state, 2*time.Minute)

	// No plates should be produced when power is tripped
	finalPlates := state.Inventory.Items[models.IronPlate]
	if finalPlates != initialPlates {
		t.Errorf("no plates should be produced when power is tripped (before=%d, after=%d)", initialPlates, finalPlates)
	}
}

// TestPowerTripOccursWhenDemandExceedsSupply verifies auto-tripping.
func TestPowerTripOccursWhenDemandExceedsSupply(t *testing.T) {
	state := newState()

	// Build many machines to exceed base power
	past := time.Now().Add(-5 * time.Minute)
	for i := 0; i < 20; i++ {
		key := "smelter.big_smelter_" + string(rune('a'+i))
		state.Machines[key] = &models.Machine{
			ID:           "big_smelter_" + string(rune('a'+i)),
			Type:         models.MachineTypeSmelter,
			Recipe:       "iron_ingot",
			PowerUsageMW: models.MachinePowerUsageMW[models.MachineTypeSmelter],
			BuiltAt:      &past,
			InputSlots:   make(map[string]*models.Slot),
			OutputSlots:  make(map[string]*models.Slot),
			Routes:       []models.Route{},
		}
	}

	engine.Tick(state, time.Second)

	if !state.PowerTripped {
		t.Error("expected power grid to trip when demand exceeds supply")
	}
}

// TestPowerTripReset verifies that clearing PowerTripped lets machines run again.
func TestPowerTripReset(t *testing.T) {
	state := newState()
	buildGenesisFactory(t, state)

	state.PowerTripped = true
	engine.Tick(state, 2*time.Minute)
	initialPlates := state.Inventory.Items[models.IronPlate]

	// Reset power grid
	state.PowerTripped = false
	engine.Tick(state, 2*time.Minute)

	finalPlates := state.Inventory.Items[models.IronPlate]
	if finalPlates <= initialPlates {
		t.Error("expected production to resume after power grid reset")
	}
}

// TestItemEffectsGeneric verifies that ItemEffects is used for solar panels and explorers.
func TestItemEffectsGeneric(t *testing.T) {
	// Verify solar panel effect
	effect, ok := models.ItemEffects[models.SolarPanel]
	if !ok {
		t.Fatal("SolarPanel not in ItemEffects")
	}
	if effect.SolarPanels != 1 {
		t.Errorf("SolarPanel SolarPanels: expected 1, got %d", effect.SolarPanels)
	}

	// Verify explorer effect
	explorerEffect, ok := models.ItemEffects[models.Explorer]
	if !ok {
		t.Fatal("Explorer not in ItemEffects")
	}
	if explorerEffect.Explorers != 1 {
		t.Errorf("Explorer Explorers: expected 1, got %d", explorerEffect.Explorers)
	}
}

// TestMachinesNotYetBuiltDontTick verifies that machines with future BuiltAt don't produce.
func TestMachinesNotYetBuiltDontTick(t *testing.T) {
	state := newState()
	future := time.Now().Add(60 * time.Second)
	state.Machines["miner.extractor"] = &models.Machine{
		ID:           "extractor",
		Type:         models.MachineTypeMiner,
		PowerUsageMW: models.MachinePowerUsageMW[models.MachineTypeMiner],
		BuiltAt:      &future,
		OutputSlots: map[string]*models.Slot{
			"iron_ore": {ItemType: models.IronOre, Capacity: 100},
		},
		InputSlots: make(map[string]*models.Slot),
		Routes:     []models.Route{{Target: "inventory"}},
	}

	engine.Tick(state, time.Minute)

	if state.Inventory.Items[models.IronOre] > 0 {
		t.Error("machine with future BuiltAt should not produce items")
	}
}

// TestExplorerConsumptionContributesToPowerDemand verifies that explorers add to power demand.
func TestExplorerConsumptionContributesToPowerDemand(t *testing.T) {
	state := newState()
	state.Explorers = 5 // 5 explorers at 2MW each = 10MW (same as base)

	// Should be right at limit — tiny push over trips it
	state.Explorers = 6 // 6 * 2 = 12MW > 10MW base

	engine.Tick(state, time.Second)

	if !state.PowerTripped {
		t.Error("expected power trip when explorer consumption exceeds supply")
	}
}
