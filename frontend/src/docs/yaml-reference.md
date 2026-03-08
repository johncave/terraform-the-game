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
| `outputs` | No | List of route targets for produced items |

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
      outputs:
        - target: "inventory"
```

## Plan vs Apply

- **Plan**: validates the YAML, checks recipes, routes, node IDs and build costs. Does **not** change game state.
- **Apply**: validates and then deducts build costs, registers machines. Machines become active after a **60-second** warm-up.

## Multiple Factories

You can have multiple factories with different names. Applying the same factory name again is an **update** — existing machines are kept, new machines are built and charged.

Use the factory tabs in the top bar to switch between factory views.
