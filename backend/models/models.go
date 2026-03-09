package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const StackHeight = 100
const MaxStacksPerItem = 10

// Power constants
const BasePowerMW = 10        // Starting power available (MW)
const SolarPanelGenMW = 5     // MW generated per solar panel built
const ExplorerConsumptionMW = 2 // MW consumed per active explorer

type ItemType string

const (
	IronOre    ItemType = "iron_ore"
	IronIngot  ItemType = "iron_ingot"
	IronPlate  ItemType = "iron_plate"
	IronSheet  ItemType = "iron_sheet"
	IronBlock  ItemType = "iron_block"
	IronWheel  ItemType = "iron_wheel"
	SolarPanel ItemType = "solar_panel"
	Explorer   ItemType = "explorer"
)

type MachineType string

const (
	MachineTypeMiner     MachineType = "miner"
	MachineTypeSmelter   MachineType = "smelter"
	MachineTypeBuilder   MachineType = "builder"
	MachineTypeAssembler MachineType = "assembler"
)

type MachineStatus string

const (
	StatusGreen  MachineStatus = "GREEN"
	StatusYellow MachineStatus = "YELLOW"
	StatusRed    MachineStatus = "RED"
)

type Item struct {
	Type        ItemType `json:"type"`
	Count       int      `json:"count"`
	StackHeight int      `json:"stack_height"`
}

type Slot struct {
	ItemType ItemType `json:"item_type"`
	Count    int      `json:"count"`
	Capacity int      `json:"capacity"`
}

type Route struct {
	Target string `json:"target"`
}

type Machine struct {
	ID              string            `json:"id"`
	Type            MachineType       `json:"type"`
	Recipe          string            `json:"recipe,omitempty"`
	NodeID          string            `json:"node_id,omitempty"`
	// RecipeInputs holds per-machine input overrides defined via the YAML `inputs:` field.
	// When non-empty, these quantities are used instead of the recipe's default Inputs.
	RecipeInputs    map[ItemType]int  `json:"recipe_inputs,omitempty"`
	InputSlots      map[string]*Slot  `json:"input_slots"`
	OutputSlots     map[string]*Slot  `json:"output_slots"`
	Status          MachineStatus     `json:"status"`
	Routes          []Route           `json:"routes"`
	FactoryID       string            `json:"factory_id"`
	PowerUsageMW    int               `json:"power_usage_mw"`   // MW draw of this machine
	BuiltAt         *time.Time        `json:"built_at,omitempty"`
	ProdAccumulator float64           `json:"prod_accumulator"` // fractional production remainder carried across ticks
}

type PlanetInventory struct {
	Items map[ItemType]int `json:"items"`
}

func (pi *PlanetInventory) CanAdd(item ItemType, count int) bool {
	current := pi.Items[item]
	return current+count <= StackHeight*MaxStacksPerItem
}

func (pi *PlanetInventory) Add(item ItemType, count int) bool {
	if !pi.CanAdd(item, count) {
		return false
	}
	if pi.Items == nil {
		pi.Items = make(map[ItemType]int)
	}
	pi.Items[item] += count
	return true
}

func (pi *PlanetInventory) Remove(item ItemType, count int) bool {
	if pi.Items[item] < count {
		return false
	}
	pi.Items[item] -= count
	return true
}

type NodeKind string

const (
	NodeKindIronOre NodeKind = "iron_ore"
)

type GameState struct {
	GameID             uuid.UUID           `json:"game_id"`
	Inventory          PlanetInventory     `json:"inventory"`
	Machines           map[string]*Machine `json:"machines"`
	DiscoveredNodes    []string            `json:"discovered_nodes"`
	NodeTypes          map[string]string   `json:"node_types"` // node_id -> NodeKind
	LastTick           time.Time           `json:"last_tick"`
	PowerGeneration    int                 `json:"power_generation"`     // number of solar panels built
	PowerConsumptionMW int                 `json:"power_consumption_mw"` // last-tick total demand (informational)
	PowerTripped       bool                `json:"power_tripped"`        // true when demand > supply
	Explorers          int                 `json:"explorers_built"`
	TickCount          int64               `json:"tick_count"`
}

// TotalPowerAvailableMW returns the total MW available for the current game state.
func (gs *GameState) TotalPowerAvailableMW() int {
	return BasePowerMW + gs.PowerGeneration*SolarPanelGenMW
}

type Factory struct {
	ID        string     `json:"id"`
	GameID    uuid.UUID  `json:"game_id"`
	YAML      string     `json:"yaml"`
	Status    string     `json:"status"`
	Machines  []*Machine `json:"machines"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Event struct {
	ID          int64     `json:"id"`
	GameID      uuid.UUID `json:"game_id"`
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	PayloadJSON []byte    `json:"payload_json"`
}

type Recipe struct {
	Inputs  map[ItemType]int
	Outputs map[ItemType]int
	Rate    int
}

var Recipes = map[string]Recipe{
	"iron_ingot": {
		Inputs:  map[ItemType]int{IronOre: 1},
		Outputs: map[ItemType]int{IronIngot: 1},
		Rate:    30,
	},
	"iron_plate": {
		Inputs:  map[ItemType]int{IronIngot: 1},
		Outputs: map[ItemType]int{IronPlate: 1},
		Rate:    30,
	},
	// iron_sheet: thin pressed sheet — 1 ingot produces 1 sheet (low density)
	"iron_sheet": {
		Inputs:  map[ItemType]int{IronIngot: 1},
		Outputs: map[ItemType]int{IronSheet: 1},
		Rate:    30,
	},
	// iron_block: solid block — requires 4 ingots (high density)
	"iron_block": {
		Inputs:  map[ItemType]int{IronIngot: 4},
		Outputs: map[ItemType]int{IronBlock: 1},
		Rate:    30,
	},
	"iron_wheel": {
		Inputs:  map[ItemType]int{IronBlock: 1},
		Outputs: map[ItemType]int{IronWheel: 1},
		Rate:    30,
	},
	"solar_panel": {
		Inputs:  map[ItemType]int{IronPlate: 1, IronBlock: 1},
		Outputs: map[ItemType]int{SolarPanel: 1},
		Rate:    30,
	},
	"explorer": {
		Inputs:  map[ItemType]int{IronBlock: 1, IronWheel: 1},
		Outputs: map[ItemType]int{Explorer: 1},
		Rate:    30,
	},
}

var BuildCosts = map[MachineType]map[ItemType]int{
	MachineTypeMiner:     {},
	MachineTypeSmelter:   {IronPlate: 3},
	MachineTypeBuilder:   {IronPlate: 5},
	MachineTypeAssembler: {IronPlate: 10, IronBlock: 5},
}

// MinerRate is the number of items a miner produces per minute.
const MinerRate = 120

// ProcessingRate is the number of items a processing machine (smelter/builder/assembler) produces per minute.
const ProcessingRate = 30

// ItemEffect describes the side effects of producing an item (beyond routing it to inventory).
type ItemEffect struct {
	SolarPanels int // number of solar panels this item counts as (each generates SolarPanelGenMW)
	Explorers   int // explorer count increment when this item is produced
}

// ItemEffects maps output item types to their side effects.
// This makes adding new "special" item types generic — just add an entry here.
var ItemEffects = map[ItemType]ItemEffect{
	SolarPanel: {SolarPanels: 1},
	Explorer:   {Explorers: 1},
}

// MachinePowerUsageMW is the ongoing power draw of each machine type while active.
var MachinePowerUsageMW = map[MachineType]int{
	MachineTypeMiner:     2,
	MachineTypeSmelter:   3,
	MachineTypeBuilder:   3,
	MachineTypeAssembler: 5,
}

func NewGameState(gameID uuid.UUID) *GameState {
	return &GameState{
		GameID: gameID,
		Inventory: PlanetInventory{
			// Start with enough iron plates to bootstrap the first factory.
			// Iron ore is not a planet inventory item — it flows directly from
			// miners into their output slots and onward through routes.
			Items: map[ItemType]int{
				IronPlate: 20,
			},
		},
		Machines:        make(map[string]*Machine),
		DiscoveredNodes: []string{"node_alpha", "node_beta"},
		NodeTypes: map[string]string{
			"node_alpha": string(NodeKindIronOre),
			"node_beta":  string(NodeKindIronOre),
		},
		LastTick: time.Now(),
	}
}

func (gs *GameState) ApplyEvent(eventType string, payload []byte) {
	switch eventType {
	case "MachineConstructed":
		var m Machine
		if err := json.Unmarshal(payload, &m); err == nil {
			key := string(m.Type) + "." + m.ID
			gs.Machines[key] = &m
		}
	case "InventoryUpdated":
		var inv PlanetInventory
		if err := json.Unmarshal(payload, &inv); err == nil {
			gs.Inventory = inv
		}
	case "TickApplied":
		var tick struct {
			NewState GameState `json:"new_state"`
		}
		if err := json.Unmarshal(payload, &tick); err == nil {
			gs.Inventory = tick.NewState.Inventory
			gs.Machines = tick.NewState.Machines
			gs.LastTick = tick.NewState.LastTick
		}
	case "NodeDiscovered":
		var node struct {
			NodeID   string `json:"node_id"`
			NodeType string `json:"node_type"`
		}
		if err := json.Unmarshal(payload, &node); err == nil {
			gs.DiscoveredNodes = append(gs.DiscoveredNodes, node.NodeID)
			if gs.NodeTypes == nil {
				gs.NodeTypes = make(map[string]string)
			}
			if node.NodeType != "" {
				gs.NodeTypes[node.NodeID] = node.NodeType
			}
		}
	}
}
