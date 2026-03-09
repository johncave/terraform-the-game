package engine

import (
	"strings"
	"time"

	"github.com/johncave/terraform-the-game/models"
)

// machineProcessOrder defines the order in which machine types are ticked each game tick.
// Processing in pipeline order (miners before smelters before builders/assemblers) ensures
// that items flow through the full chain in a single tick.
var machineProcessOrder = []models.MachineType{
	models.MachineTypeMiner,
	models.MachineTypeSmelter,
	models.MachineTypeBuilder,
	models.MachineTypeAssembler,
}

// Tick processes one second of game time for the given state.
// It mutates state in-place and returns whether anything changed.
func Tick(state *models.GameState, elapsed time.Duration) bool {
	now := time.Now()
	elapsedSecs := elapsed.Seconds()
	if elapsedSecs <= 0 {
		return false
	}

	// --- Power budget ---
	// Calculate total demand from all active (built) machines + explorers.
	totalDemandMW := 0
	for _, machine := range state.Machines {
		if machine.BuiltAt != nil && !machine.BuiltAt.After(now) {
			totalDemandMW += machine.PowerUsageMW
		}
	}
	totalDemandMW += state.Explorers * models.ExplorerConsumptionMW
	state.PowerConsumptionMW = totalDemandMW

	availableMW := state.TotalPowerAvailableMW()

	// If demand exceeds supply, trip the grid. All machines stop.
	if totalDemandMW > availableMW && !state.PowerTripped {
		state.PowerTripped = true
	}
	if state.PowerTripped {
		// Mark all active machines RED while power is tripped.
		for _, machine := range state.Machines {
			if machine.BuiltAt != nil && !machine.BuiltAt.After(now) {
				machine.Status = models.StatusRed
			}
		}
		state.LastTick = now
		state.TickCount++
		return true
	}

	changed := false

	// Process each machine TYPE in pipeline order so items flow through the
	// full chain (miners → smelters → builders/assemblers) in a single tick.
	for _, machineType := range machineProcessOrder {
		for _, machine := range state.Machines {
			if machine.Type != machineType {
				continue
			}
			if machine.BuiltAt == nil || machine.BuiltAt.After(now) {
				continue // not yet built
			}

			switch machineType {
			case models.MachineTypeMiner:
				changed = tickMiner(state, machine, elapsedSecs) || changed
			case models.MachineTypeSmelter, models.MachineTypeBuilder, models.MachineTypeAssembler:
				changed = tickProcessor(state, machine, elapsedSecs) || changed
			}
		}
	}

	state.LastTick = now
	state.TickCount++
	return changed
}

// FastForward calculates elapsed time since last tick and applies many ticks at once.
func FastForward(state *models.GameState) bool {
	now := time.Now()
	elapsed := now.Sub(state.LastTick)
	if elapsed < time.Second {
		return false
	}
	return Tick(state, elapsed)
}

func tickMiner(state *models.GameState, machine *models.Machine, elapsedSecs float64) bool {
	if len(machine.OutputSlots) == 0 {
		updateMinerStatus(machine)
		return false
	}

	// Accumulate fractional production so no items are lost between ticks.
	machine.ProdAccumulator += float64(models.MinerRate) / 60.0 * elapsedSecs
	itemsToProduce := int(machine.ProdAccumulator)
	machine.ProdAccumulator -= float64(itemsToProduce)

	if itemsToProduce <= 0 {
		updateMinerStatus(machine)
		return false
	}

	// Use the first configured output slot; miners carry their output item type
	// in their OutputSlots map (set at parse/apply time), not hard-coded here.
	var outputItem models.ItemType
	var outSlot *models.Slot
	for k, s := range machine.OutputSlots {
		outputItem = models.ItemType(k)
		outSlot = s
		break
	}
	if outSlot == nil {
		updateMinerStatus(machine)
		return false
	}

	available := outSlot.Capacity - outSlot.Count
	if available <= 0 {
		machine.Status = models.StatusYellow
		return false
	}

	canProduce := itemsToProduce
	if canProduce > available {
		canProduce = available
	}

	outSlot.Count += canProduce
	routeItems(state, machine, outputItem, canProduce)
	updateMinerStatus(machine)
	return canProduce > 0
}

func tickProcessor(state *models.GameState, machine *models.Machine, elapsedSecs float64) bool {
	recipe, ok := models.Recipes[machine.Recipe]
	if !ok {
		machine.Status = models.StatusRed
		return false
	}

	// Accumulate fractional production so no craft cycles are lost between ticks.
	machine.ProdAccumulator += float64(models.ProcessingRate) / 60.0 * elapsedSecs
	itemsToProduce := int(machine.ProdAccumulator)
	machine.ProdAccumulator -= float64(itemsToProduce)

	if itemsToProduce <= 0 {
		updateProcessorStatus(machine, recipe)
		return false
	}

	changed := false
	for i := 0; i < itemsToProduce; i++ {
		if !processOnce(state, machine, recipe) {
			break
		}
		changed = true
	}

	updateProcessorStatus(machine, recipe)
	return changed
}

func processOnce(state *models.GameState, machine *models.Machine, recipe models.Recipe) bool {
	// Check all input slots have required items
	for itemType, required := range recipe.Inputs {
		slot, exists := machine.InputSlots[string(itemType)]
		if !exists || slot.Count < required {
			machine.Status = models.StatusRed
			return false
		}
	}

	// Check output slot has capacity
	for itemType := range recipe.Outputs {
		slot, exists := machine.OutputSlots[string(itemType)]
		if !exists {
			slot = &models.Slot{
				ItemType: itemType,
				Capacity: models.StackHeight,
			}
			machine.OutputSlots[string(itemType)] = slot
		}
		if slot.Count >= slot.Capacity {
			machine.Status = models.StatusYellow
			return false
		}
	}

	// Consume inputs
	for itemType, required := range recipe.Inputs {
		machine.InputSlots[string(itemType)].Count -= required
	}

	// Produce outputs
	for itemType, amount := range recipe.Outputs {
		slot := machine.OutputSlots[string(itemType)]
		slot.Count += amount

		// Route immediately
		routeItems(state, machine, itemType, amount)

		// Apply generic item effects (solar panel count, explorer count, etc.)
		if effect, hasEffect := models.ItemEffects[itemType]; hasEffect {
			state.PowerGeneration += effect.SolarPanels * amount
			state.Explorers += effect.Explorers * amount
		}
	}

	return true
}

// routeItems routes items from machine output to their destination.
func routeItems(state *models.GameState, machine *models.Machine, itemType models.ItemType, count int) {
	if len(machine.Routes) == 0 {
		return
	}

	for _, route := range machine.Routes {
		if route.Target == "inventory" {
			outSlot := machine.OutputSlots[string(itemType)]
			if outSlot == nil {
				continue
			}
			toMove := outSlot.Count
			if toMove > count {
				toMove = count
			}
			if toMove <= 0 {
				continue
			}
			added := state.Inventory.Add(itemType, toMove)
			if added {
				outSlot.Count -= toMove
			}
		} else {
			// Format: "machinetype.machinename.inputs.slotname"
			parts := strings.Split(route.Target, ".")
			if len(parts) < 4 {
				continue
			}
			targetKey := parts[0] + "." + parts[1]
			slotName := strings.Join(parts[3:], ".")

			targetMachine, exists := state.Machines[targetKey]
			if !exists {
				continue
			}

			outSlot := machine.OutputSlots[string(itemType)]
			if outSlot == nil {
				continue
			}
			toMove := outSlot.Count
			if toMove > count {
				toMove = count
			}
			if toMove <= 0 {
				continue
			}

			inSlot, slotExists := targetMachine.InputSlots[slotName]
			if !slotExists {
				inSlot = &models.Slot{
					ItemType: itemType,
					Capacity: models.StackHeight,
				}
				targetMachine.InputSlots[slotName] = inSlot
			}

			available := inSlot.Capacity - inSlot.Count
			if available <= 0 {
				continue
			}
			if toMove > available {
				toMove = available
			}

			inSlot.Count += toMove
			outSlot.Count -= toMove
		}
	}
}

func updateMinerStatus(machine *models.Machine) {
	for _, slot := range machine.OutputSlots {
		if slot.Count >= slot.Capacity {
			machine.Status = models.StatusYellow
			return
		}
	}
	machine.Status = models.StatusGreen
}

func updateProcessorStatus(machine *models.Machine, recipe models.Recipe) {
	// Check if output is full
	for itemType := range recipe.Outputs {
		slot, exists := machine.OutputSlots[string(itemType)]
		if exists && slot.Count >= slot.Capacity {
			machine.Status = models.StatusYellow
			return
		}
	}

	// Check if any input is empty
	for itemType, required := range recipe.Inputs {
		slot, exists := machine.InputSlots[string(itemType)]
		if !exists || slot.Count < required {
			machine.Status = models.StatusRed
			return
		}
	}

	machine.Status = models.StatusGreen
}

