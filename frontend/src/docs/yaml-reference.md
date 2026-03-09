# Factory YAML Reference

Factories are defined using a YAML configuration. Each factory has a **name** (set in the editor header) and a `resources` block.

## Structure

```yaml
resources:
  <machine_type>:
    <machine_name>:
      # machine-specific fields
```

- `<machine_type>`: one of `miner`, `smelter`, `builder`, `assembler`
- `<machine_name>`: a unique name for this machine within the factory

## Fields

### All machines

| Field | Required | Description |
|-------|----------|-------------|
| `count` | No | Number of identical machines to create (default 1); names become `<name>_1`, `<name>_2`, etc. |
| `outputs` | No | List of route targets for produced items |
| `inputs` | No | Override recipe input quantities for this machine (see Recipe Cost Overrides) |

### Miner only

| Field | Required | Description |
|-------|----------|-------------|
| `node_id` | **Yes** | ID of the node to mine (must be discovered) |

### Smelter / Builder / Assembler

| Field | Required | Description |
|-------|----------|-------------|
| `recipe` | **Yes** | Recipe name to execute (see Recipes) |

## Outputs / Routes

```yaml
outputs:
  - target: "inventory"
  - target: "<type>.<name>.inputs.<slot>"
```

- `"inventory"` — sends produced items to planet inventory
- `"<type>.<name>.inputs.<slot>"` — sends items to another machine's input slot

### Examples

```yaml
- target: "smelter.furnace_1.inputs.iron_ore"
- target: "builder.press_1.inputs.iron_ingot"
- target: "inventory"
```

## Recipe Cost Overrides

The `inputs:` field lets you override how many of each resource a machine consumes per craft cycle. Without it, the recipe's default quantities apply.

```yaml
builder:
  block_press:
    recipe: "iron_block"
    inputs:
      iron_ingot: 2    # uses 2 ingots instead of the default 4
    outputs:
      - target: "inventory"
```

- All keys must be positive integers (≥ 1).
- The override applies **only** to that specific machine instance.
- Other machines using the same recipe are unaffected.
- Changing an override after apply shows up as `~` (change) in `terraform plan`.

## Recipes

| Recipe | Machine | Default Inputs | Output |
|--------|---------|----------------|--------|
| `iron_ingot` | smelter | 1 × iron_ore | 1 × iron_ingot |
| `iron_sheet` | builder | 1 × iron_ingot | 1 × iron_sheet |
| `iron_plate` | builder | 1 × iron_ingot | 1 × iron_plate |
| `iron_block` | builder | **4 × iron_ingot** | 1 × iron_block |
| `iron_wheel` | builder | 1 × iron_block | 1 × iron_wheel |
| `solar_panel` | assembler | 1 × iron_plate + 1 × iron_block | 1 × solar_panel |
| `explorer` | assembler | 1 × iron_block + 1 × iron_wheel | 1 × explorer |

## Full Example

```yaml
resources:
  miner:
    extractor_1:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.furnace_1.inputs.iron_ore"
    extractor_2:
      node_id: "node_beta"
      outputs:
        - target: "smelter.furnace_2.inputs.iron_ore"

  smelter:
    furnace_1:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.press_1.inputs.iron_ingot"
    furnace_2:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.block_caster.inputs.iron_ingot"

  builder:
    press_1:
      recipe: "iron_plate"
      outputs:
        - target: "inventory"
    block_caster:
      recipe: "iron_block"
      # Override: 2 ingots per block instead of the default 4
      inputs:
        iron_ingot: 2
      outputs:
        - target: "inventory"
```

## Plan vs Apply

- **Plan**: validates the YAML, checks recipes, routes, node IDs and build costs. Does **not** change game state.
- **Apply**: runs plan first, then deducts build costs, registers machines. Machines removed from the YAML are destroyed. Machines become active after a **60-second** warm-up.

## Multiple Factories

You can have multiple factories with different names. Applying the same factory name again is an **update** — existing machines are kept, new machines are built, removed machines are destroyed.

Use the factory tabs in the top bar to switch between factory views.
