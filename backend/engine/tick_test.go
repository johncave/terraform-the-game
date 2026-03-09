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

// TestIronBlockCosts4IngotsPerCycle verifies the engine processes iron_block at 4 ingots/block.
func TestIronBlockCosts4IngotsPerCycle(t *testing.T) {
	state := newState()
	past := time.Now().Add(-5 * time.Minute)

	// Put a builder with iron_block recipe and 12 ingots in input
	state.Machines["builder.block_press"] = &models.Machine{
		ID:           "block_press",
		Type:         models.MachineTypeBuilder,
		Recipe:       "iron_block",
		PowerUsageMW: models.MachinePowerUsageMW[models.MachineTypeBuilder],
		BuiltAt:      &past,
		InputSlots: map[string]*models.Slot{
			"iron_ingot": {ItemType: models.IronIngot, Count: 12, Capacity: 100},
		},
		OutputSlots: map[string]*models.Slot{
			"iron_block": {ItemType: models.IronBlock, Count: 0, Capacity: 100},
		},
		Routes: []models.Route{{Target: "inventory"}},
	}

	engine.Tick(state, time.Minute)

	blocks := state.Inventory.Items[models.IronBlock]
	// 12 ingots / 4 per block = 3 blocks max in 1 minute (rate 30/min → 30 cycles, but only 3 can complete)
	if blocks == 0 {
		t.Error("expected iron_blocks to be produced")
	}
	if blocks > 3 {
		t.Errorf("expected at most 3 iron_blocks from 12 ingots at 4/block, got %d", blocks)
	}
	// Verify ingots were consumed correctly: 3 blocks × 4 = 12 ingots consumed
	remaining := state.Machines["builder.block_press"].InputSlots["iron_ingot"].Count
	if remaining != 12-blocks*4 {
		t.Errorf("expected %d ingots remaining, got %d", 12-blocks*4, remaining)
	}
}

// TestInputOverrideIsUsedInProcessing verifies that a machine with RecipeInputs consumes
// the overridden quantity per cycle instead of the recipe default.
func TestInputOverrideIsUsedInProcessing(t *testing.T) {
	// iron_block default costs 4 ingots; override to 2
	const yaml = `
resources:
  miner:
    ore_miner:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.smelter1.inputs.iron_ore"
  smelter:
    smelter1:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.block_press.inputs.iron_ingot"
  builder:
    block_press:
      recipe: "iron_block"
      inputs:
        iron_ingot: 2
      outputs:
        - target: "inventory"
`
	state := newState()
	result, err := parser.Parse("test", yaml)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if errs := parser.Validate(result, state); len(errs) > 0 {
		t.Fatalf("validate: %v", errs)
	}
	parser.Apply(result, state)
	past := time.Now().Add(-5 * time.Minute)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		m.BuiltAt = &past
		state.Machines[key] = m
	}

	// Tick 2 minutes — expect iron_blocks produced
	engine.Tick(state, 2*time.Minute)

	blocks := state.Inventory.Items[models.IronBlock]
	if blocks == 0 {
		t.Error("expected iron_blocks in inventory with 2-ingot override")
	}
	// With override=2 and ~60 ingots produced in 2min, expect at least 15 blocks.
	if blocks < 15 {
		t.Errorf("expected >= 15 iron_blocks with 2-ingot override, got %d", blocks)
	}
}

// TestDefaultRecipeCostIsUsedWhenNoOverride verifies that without overrides, the
// iron_block recipe costs 4 ingots per cycle.
func TestDefaultRecipeCostIsUsedWhenNoOverride(t *testing.T) {
	state := newState()
	past := time.Now().Add(-5 * time.Minute)
	state.Machines["builder.block_press"] = &models.Machine{
		ID:           "block_press",
		Type:         models.MachineTypeBuilder,
		Recipe:       "iron_block",
		PowerUsageMW: models.MachinePowerUsageMW[models.MachineTypeBuilder],
		BuiltAt:      &past,
		InputSlots: map[string]*models.Slot{
			"iron_ingot": {ItemType: models.IronIngot, Count: 8, Capacity: 100},
		},
		OutputSlots: map[string]*models.Slot{
			"iron_block": {ItemType: models.IronBlock, Count: 0, Capacity: 100},
		},
		Routes: []models.Route{{Target: "inventory"}},
	}

	engine.Tick(state, time.Minute)

	blocks := state.Inventory.Items[models.IronBlock]
	// 8 ingots / 4 per block = 2 blocks (rate=30/min but only 2 cycles can complete)
	if blocks == 0 {
		t.Error("expected iron_blocks produced from 8 ingots at 4 per block")
	}
	if blocks > 2 {
		t.Errorf("expected at most 2 iron_blocks from 8 ingots at 4/block, got %d", blocks)
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

// applyGenesisInstant applies the genesis factory and stores machines in state.
// Unlike buildGenesisFactory, it does NOT backdate BuiltAt – machines use the
// instant BuiltAt set by parser.Apply (i.e. time.Now()), so they are live on the
// very first tick.
func applyGenesisInstant(t *testing.T, state *models.GameState) {
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
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}
}

// TestInstantBuildPowerConsumption verifies that after applying the genesis factory
// (with instant BuiltAt), the first tick immediately reflects full power consumption
// (miner=2MW + smelter=3MW + builder=3MW = 8MW total).
func TestInstantBuildPowerConsumption(t *testing.T) {
	state := newState()
	applyGenesisInstant(t, state)

	engine.Tick(state, time.Second)

	expected := models.MachinePowerUsageMW[models.MachineTypeMiner] +
		models.MachinePowerUsageMW[models.MachineTypeSmelter] +
		models.MachinePowerUsageMW[models.MachineTypeBuilder]
	// miner=2 + smelter=3 + builder=3 = 8 MW
	if state.PowerConsumptionMW != expected {
		t.Errorf("expected power consumption %d MW after first tick, got %d MW", expected, state.PowerConsumptionMW)
	}
}

// TestGenesisStepByStepMinerStartsImmediately verifies that after applying the genesis
// factory, the iron extractor starts producing ore on the very first tick (no build delay).
func TestGenesisStepByStepMinerStartsImmediately(t *testing.T) {
	state := newState()
	applyGenesisInstant(t, state)

	// Tick 1 second – miner should produce ore and route it to the smelter's input slot.
	engine.Tick(state, time.Second)

	smelter := state.Machines["smelter.iron_processor"]
	if smelter == nil {
		t.Fatal("smelter.iron_processor not found")
	}
	oreInSmelter := smelter.InputSlots["iron_ore"]
	if oreInSmelter == nil || oreInSmelter.Count == 0 {
		t.Error("expected iron_ore in smelter input after first tick (miner should produce immediately)")
	}
}

// TestGenesisStepByStepSmelterStartsWhenOreArrives verifies that the smelter starts
// producing ingots as soon as it has at least 1 ore. Because machines are processed in
// pipeline order within the same tick (miner→smelter→builder), the ingot the smelter
// produces in tick 2 is immediately consumed by the builder in the same tick.
// The inventory should therefore contain a plate after tick 2.
func TestGenesisStepByStepSmelterStartsWhenOreArrives(t *testing.T) {
	state := newState()
	applyGenesisInstant(t, state)

	// Tick 1: miner produces ore and routes it to the smelter input.
	// The smelter's production accumulator reaches 0.5 (30/min × 1s) – no cycle yet.
	engine.Tick(state, time.Second)

	smelter := state.Machines["smelter.iron_processor"]
	if smelter == nil {
		t.Fatal("smelter.iron_processor not found")
	}
	oreAfterTick1 := smelter.InputSlots["iron_ore"].Count
	if oreAfterTick1 == 0 {
		t.Fatal("smelter should have ore after tick 1")
	}

	// Tick 2: miner adds more ore; smelter accumulator reaches 1.0 → processes 1 cycle
	// (consumes 1 ore, produces 1 ingot). Builder accumulator also reaches 1.0 →
	// immediately processes that ingot and produces 1 plate (pipeline order).
	engine.Tick(state, time.Second)

	// Verify that the smelter consumed ore (its input count should be less than after tick 1 + new ore from miner)
	// The miner added 2 more ore, so input would have been oreAfterTick1+2. After 1 smelter cycle: oreAfterTick1+2-1.
	oreAfterTick2 := smelter.InputSlots["iron_ore"].Count
	expectedOre := oreAfterTick1 + 2 - 1 // +2 from miner, -1 consumed by smelter
	if oreAfterTick2 != expectedOre {
		t.Errorf("smelter iron_ore: expected %d after tick 2 (1 consumed), got %d", expectedOre, oreAfterTick2)
	}

	// The builder consumed the ingot in the same tick → inventory should have a plate
	plates := state.Inventory.Items[models.IronPlate]
	if plates == 0 {
		t.Error("expected iron_plate in inventory by tick 2 (smelter→builder pipeline runs in same tick)")
	}
}

// TestGenesisStepByStepBuilderStartsWhenIngotsArrive verifies that the builder starts
// producing plates as soon as it receives at least 1 ingot, and that inventory fills.
func TestGenesisStepByStepBuilderStartsWhenIngotsArrive(t *testing.T) {
	state := newState()
	applyGenesisInstant(t, state)

	// Tick several seconds to let the full pipeline run:
	// tick 1: miner → smelter (ore arrives, smelter acc=0.5)
	// tick 2: miner → smelter (acc=1.0 → 1 ingot → builder acc=0.5)
	// tick 3: more ore; builder acc=1.0 → processes 1 plate → inventory
	for i := 0; i < 4; i++ {
		engine.Tick(state, time.Second)
	}

	plates := state.Inventory.Items[models.IronPlate]
	if plates == 0 {
		t.Error("expected iron_plate in inventory after 4 ticks – builder should start as soon as ingots arrive")
	}
}

// TestGenesisStepByStepFullPipelineMultipleTicks verifies the full pipeline across
// multiple 1-second ticks with two different setups (normal recipe and block recipe).
// This test confirms the power consumption, machine state, and inventory are all correct.
func TestGenesisStepByStepFullPipelineMultipleTicks(t *testing.T) {
	// --- Setup 1: standard genesis (miner→smelter→plate builder→inventory) ---
	t.Run("plate_pipeline", func(t *testing.T) {
		state := newState()
		applyGenesisInstant(t, state)

		const tickCount = 10
		for i := 1; i <= tickCount; i++ {
			engine.Tick(state, time.Second)

			// Power consumption must remain at 8 MW throughout (no trips).
			expectedPower := models.MachinePowerUsageMW[models.MachineTypeMiner] +
				models.MachinePowerUsageMW[models.MachineTypeSmelter] +
				models.MachinePowerUsageMW[models.MachineTypeBuilder]
			if state.PowerConsumptionMW != expectedPower {
				t.Errorf("tick %d: expected power %d MW, got %d MW", i, expectedPower, state.PowerConsumptionMW)
			}
			if state.PowerTripped {
				t.Errorf("tick %d: power grid should not be tripped", i)
			}
		}

		// Miner must be GREEN after running
		miner := state.Machines["miner.iron_extractor"]
		if miner == nil {
			t.Fatal("miner.iron_extractor not found")
		}
		if miner.Status != models.StatusGreen {
			t.Errorf("miner status: expected GREEN, got %s", miner.Status)
		}

		// Smelter must be GREEN (has ore from miner, output not full)
		smelter := state.Machines["smelter.iron_processor"]
		if smelter == nil {
			t.Fatal("smelter.iron_processor not found")
		}
		if smelter.Status != models.StatusGreen {
			t.Errorf("smelter status: expected GREEN, got %s", smelter.Status)
		}

		// Inventory must contain iron_plate
		plates := state.Inventory.Items[models.IronPlate]
		if plates == 0 {
			t.Error("expected iron_plate in inventory after 10 ticks")
		}
	})

	// --- Setup 2: miner→smelter→block builder (4 ingots/block)→inventory ---
	t.Run("block_pipeline", func(t *testing.T) {
		const blockYAML = `
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
        - target: "builder.block_press.inputs.iron_ingot"
  builder:
    block_press:
      recipe: "iron_block"
      outputs:
        - target: "inventory"
`
		state := newState()
		result, err := parser.Parse("test", blockYAML)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if errs := parser.Validate(result, state); len(errs) > 0 {
			t.Fatalf("validation errors: %v", errs)
		}
		parser.Apply(result, state)
		for _, m := range result.Machines {
			key := string(m.Type) + "." + m.ID
			state.Machines[key] = m
		}

		// Tick 2 minutes to accumulate enough ingots for blocks (4 ingots per block).
		engine.Tick(state, 2*time.Minute)

		// Power must still be 8 MW
		expectedPower := models.MachinePowerUsageMW[models.MachineTypeMiner] +
			models.MachinePowerUsageMW[models.MachineTypeSmelter] +
			models.MachinePowerUsageMW[models.MachineTypeBuilder]
		if state.PowerConsumptionMW != expectedPower {
			t.Errorf("expected power %d MW, got %d MW", expectedPower, state.PowerConsumptionMW)
		}

		// iron_block must be in inventory
		blocks := state.Inventory.Items[models.IronBlock]
		if blocks == 0 {
			t.Error("expected iron_block in inventory after 2-minute tick with block recipe")
		}
	})
}
