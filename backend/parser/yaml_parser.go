package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/johncave/terraform-the-game/models"
	"gopkg.in/yaml.v3"
)

// FactoryConfig represents the top-level YAML structure
type FactoryConfig struct {
	Resources map[string]map[string]MachineConfig `yaml:"resources"`
}

// MachineConfig represents a single machine definition in YAML
type MachineConfig struct {
	NodeID  string            `yaml:"node_id"`
	Recipe  string            `yaml:"recipe"`
	Outputs []OutputConfig    `yaml:"outputs"`
	Inputs  map[string]string `yaml:"inputs"`
}

type OutputConfig struct {
	Target string `yaml:"target"`
}

// ParseResult holds the result of parsing a factory YAML
type ParseResult struct {
	Machines []*models.Machine
	Errors   []string
}

// Parse parses a factory YAML string and returns machines with routes.
func Parse(factoryID string, yamlStr string) (*ParseResult, error) {
	var config FactoryConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &config); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	result := &ParseResult{}

	for machineTypeStr, machines := range config.Resources {
		machineType := models.MachineType(machineTypeStr)

		// Validate machine type
		switch machineType {
		case models.MachineTypeMiner, models.MachineTypeSmelter, models.MachineTypeBuilder, models.MachineTypeAssembler:
		default:
			result.Errors = append(result.Errors, fmt.Sprintf("unknown machine type: %s", machineTypeStr))
			continue
		}

		for machineName, machineConf := range machines {
			key := machineTypeStr + "." + machineName

			machine := &models.Machine{
				ID:          machineName,
				Type:        machineType,
				Recipe:      machineConf.Recipe,
				NodeID:      machineConf.NodeID,
				InputSlots:  make(map[string]*models.Slot),
				OutputSlots: make(map[string]*models.Slot),
				Status:      models.StatusRed,
				FactoryID:   factoryID,
				Routes:      []models.Route{},
			}

			// Validate recipe for non-miners
			if machineType != models.MachineTypeMiner {
				if machineConf.Recipe == "" {
					result.Errors = append(result.Errors, fmt.Sprintf("machine %s missing recipe", key))
				} else if _, ok := models.Recipes[machineConf.Recipe]; !ok {
					result.Errors = append(result.Errors, fmt.Sprintf("machine %s has unknown recipe: %s", key, machineConf.Recipe))
				}
			}

			// Validate miner has node_id
			if machineType == models.MachineTypeMiner && machineConf.NodeID == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("miner %s missing node_id", key))
			}

			// Set up output slots based on recipe
			if machineType == models.MachineTypeMiner {
				machine.OutputSlots["iron_ore"] = &models.Slot{
					ItemType: models.IronOre,
					Capacity: models.StackHeight,
				}
			} else if recipe, ok := models.Recipes[machineConf.Recipe]; ok {
				for itemType := range recipe.Inputs {
					machine.InputSlots[string(itemType)] = &models.Slot{
						ItemType: itemType,
						Capacity: models.StackHeight,
					}
				}
				for itemType := range recipe.Outputs {
					machine.OutputSlots[string(itemType)] = &models.Slot{
						ItemType: itemType,
						Capacity: models.StackHeight,
					}
				}
			}

			// Parse outputs/routes
			for _, out := range machineConf.Outputs {
				machine.Routes = append(machine.Routes, models.Route{Target: out.Target})
			}

			result.Machines = append(result.Machines, machine)
		}
	}

	return result, nil
}

// Validate checks the parse result against the current game state.
func Validate(result *ParseResult, state *models.GameState) []string {
	var errors []string
	errors = append(errors, result.Errors...)

	// Build a set of machine keys for route validation
	factoryMachineKeys := make(map[string]bool)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		factoryMachineKeys[key] = true
	}
	// Also include existing machines in game state
	for key := range state.Machines {
		factoryMachineKeys[key] = true
	}

	// Check build costs
	costs := make(map[models.ItemType]int)
	for _, m := range result.Machines {
		existingKey := string(m.Type) + "." + m.ID
		if _, alreadyBuilt := state.Machines[existingKey]; alreadyBuilt {
			continue
		}
		if buildCost, ok := models.BuildCosts[m.Type]; ok {
			for item, count := range buildCost {
				costs[item] += count
			}
		}
	}
	for item, required := range costs {
		if state.Inventory.Items[item] < required {
			errors = append(errors, fmt.Sprintf("insufficient %s: need %d, have %d", item, required, state.Inventory.Items[item]))
		}
	}

	// Validate node_ids for miners
	discoveredSet := make(map[string]bool)
	for _, node := range state.DiscoveredNodes {
		discoveredSet[node] = true
	}
	for _, m := range result.Machines {
		if m.Type == models.MachineTypeMiner && m.NodeID != "" {
			if !discoveredSet[m.NodeID] {
				errors = append(errors, fmt.Sprintf("node %s not discovered", m.NodeID))
			}
		}
	}

	// Validate route targets
	for _, m := range result.Machines {
		for _, route := range m.Routes {
			if route.Target == "inventory" {
				continue
			}
			parts := strings.Split(route.Target, ".")
			if len(parts) < 4 {
				errors = append(errors, fmt.Sprintf("invalid route target: %s", route.Target))
				continue
			}
			targetKey := parts[0] + "." + parts[1]
			if !factoryMachineKeys[targetKey] {
				errors = append(errors, fmt.Sprintf("route target machine not found: %s", targetKey))
			}
		}
	}

	return errors
}

// Apply sets BuiltAt on machines (60 seconds from now) and deducts build costs from inventory.
func Apply(result *ParseResult, state *models.GameState) {
	builtAt := time.Now().Add(60 * time.Second)

	for _, m := range result.Machines {
		existingKey := string(m.Type) + "." + m.ID
		if _, alreadyBuilt := state.Machines[existingKey]; alreadyBuilt {
			continue
		}

		// Deduct build costs
		if buildCost, ok := models.BuildCosts[m.Type]; ok {
			for item, count := range buildCost {
				state.Inventory.Remove(item, count)
			}
		}

		t := builtAt
		m.BuiltAt = &t
	}
}
