package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const StackHeight = 100
const MaxStacksPerItem = 10

type ItemType string

const (
	IronOre    ItemType = "iron_ore"
	IronIngot  ItemType = "iron_ingot"
	IronPlate  ItemType = "iron_plate"
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
	ID          string           `json:"id"`
	Type        MachineType      `json:"type"`
	Recipe      string           `json:"recipe,omitempty"`
	NodeID      string           `json:"node_id,omitempty"`
	InputSlots  map[string]*Slot `json:"input_slots"`
	OutputSlots map[string]*Slot `json:"output_slots"`
	Status      MachineStatus    `json:"status"`
	Routes      []Route          `json:"routes"`
	FactoryID   string           `json:"factory_id"`
	BuiltAt          *time.Time `json:"built_at,omitempty"`
	ProdAccumulator  float64    `json:"prod_accumulator"` // fractional production remainder carried across ticks
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

type GameState struct {
	GameID          uuid.UUID           `json:"game_id"`
	Inventory       PlanetInventory     `json:"inventory"`
	Machines        map[string]*Machine `json:"machines"`
	DiscoveredNodes []string            `json:"discovered_nodes"`
	LastTick        time.Time           `json:"last_tick"`
	PowerGeneration int                 `json:"power_generation"`
	Explorers       int                 `json:"explorers_built"`
	TickCount       int64               `json:"tick_count"`
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
	"iron_block": {
		Inputs:  map[ItemType]int{IronIngot: 1},
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

func NewGameState(gameID uuid.UUID) *GameState {
	return &GameState{
		GameID: gameID,
		Inventory: PlanetInventory{
			Items: map[ItemType]int{
				IronOre: 500,
			},
		},
		Machines:        make(map[string]*Machine),
		DiscoveredNodes: []string{"node_alpha", "node_beta"},
		LastTick:        time.Now(),
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
			NodeID string `json:"node_id"`
		}
		if err := json.Unmarshal(payload, &node); err == nil {
			gs.DiscoveredNodes = append(gs.DiscoveredNodes, node.NodeID)
		}
	}
}
