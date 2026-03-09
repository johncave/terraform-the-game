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
	Count   int               `yaml:"count"` // number of identical machines to create (default 1)
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

// PlanDiff describes what changes will be applied.
type PlanDiff struct {
	ToAdd     []string // machine keys to be newly created
	ToChange  []string // machine keys that already exist but have changed config
	ToDestroy []string // machine keys in current state (for this factory) not in new YAML
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
			// Determine how many copies to create (count field, defaults to 1)
			count := machineConf.Count
			if count < 1 {
				count = 1
			}

			for i := 0; i < count; i++ {
				// For count > 1, suffix the name with _1, _2, ...
				instanceName := machineName
				if count > 1 {
					instanceName = fmt.Sprintf("%s_%d", machineName, i+1)
				}

				key := machineTypeStr + "." + instanceName

				machine := &models.Machine{
					ID:           instanceName,
					Type:         machineType,
					Recipe:       machineConf.Recipe,
					NodeID:       machineConf.NodeID,
					InputSlots:   make(map[string]*models.Slot),
					OutputSlots:  make(map[string]*models.Slot),
					Status:       models.StatusRed,
					FactoryID:    factoryID,
					Routes:       []models.Route{},
					PowerUsageMW: models.MachinePowerUsageMW[machineType],
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
	}

	return result, nil
}

// Diff computes what would change relative to the current game state for a given factory.
func Diff(factoryID string, result *ParseResult, state *models.GameState) PlanDiff {
	var diff PlanDiff

	// Build set of incoming machine keys
	incomingKeys := make(map[string]bool)
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		incomingKeys[key] = true
	}

	// Find existing machines in this factory
	for key, existing := range state.Machines {
		if existing.FactoryID != factoryID {
			continue
		}
		if !incomingKeys[key] {
			diff.ToDestroy = append(diff.ToDestroy, key)
		}
	}

	// Classify new vs changed machines
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		existing, alreadyBuilt := state.Machines[key]
		if !alreadyBuilt {
			diff.ToAdd = append(diff.ToAdd, key)
		} else {
			// Check if anything changed (recipe, routes, node_id)
			if existing.Recipe != m.Recipe || existing.NodeID != m.NodeID ||
				!routesEqual(existing.Routes, m.Routes) {
				diff.ToChange = append(diff.ToChange, key)
			}
		}
	}

	return diff
}

func routesEqual(a, b []models.Route) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Target != b[i].Target {
			return false
		}
	}
	return true
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

	// Check build costs (only for new machines not yet built)
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
// For machines that already exist, it updates their recipe/routes in-place and carries over BuiltAt.
func Apply(result *ParseResult, state *models.GameState) {
	builtAt := time.Now().Add(60 * time.Second)

	for _, m := range result.Machines {
		existingKey := string(m.Type) + "." + m.ID
		existing, alreadyBuilt := state.Machines[existingKey]
		if alreadyBuilt {
			// Update recipe/routes/node_id in place but preserve BuiltAt and accumulators.
			existing.Recipe = m.Recipe
			existing.NodeID = m.NodeID
			existing.Routes = m.Routes
			// Rebuild slots if recipe changed (reset counts to 0 for new recipe).
			existing.InputSlots = m.InputSlots
			existing.OutputSlots = m.OutputSlots
			// Carry BuiltAt and accumulator back to m so the caller can safely use m.
			m.BuiltAt = existing.BuiltAt
			m.ProdAccumulator = existing.ProdAccumulator
			m.Status = existing.Status
			continue
		}

		// New machine: deduct build costs and schedule for construction.
		if buildCost, ok := models.BuildCosts[m.Type]; ok {
			for item, count := range buildCost {
				state.Inventory.Remove(item, count)
			}
		}

		t := builtAt
		m.BuiltAt = &t
	}
}

