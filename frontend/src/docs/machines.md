# Machines

Machines are the building blocks of your factories. Each machine is declared in your factory YAML under `resources`.

## Machine Types

### Miner

Extracts raw resources from a discovered node.

- **YAML type**: `miner`
- **Build cost**: Free
- **Production rate**: 120 items / minute
- **Required field**: `node_id` — the node to mine from
- **Output**: Raw resource for the node type (currently `iron_ore`)

```yaml
resources:
  miner:
    my_miner:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.my_smelter.inputs.iron_ore"
```

---

### Smelter

Processes raw ore into ingots.

- **YAML type**: `smelter`
- **Build cost**: 3 × iron plate
- **Production rate**: 30 items / minute
- **Required field**: `recipe` — what to produce

```yaml
resources:
  smelter:
    my_smelter:
      recipe: "iron_ingot"
      outputs:
        - target: "builder.my_builder.inputs.iron_ingot"
```

---

### Builder

Assembles ingots into plates or simple components.

- **YAML type**: `builder`
- **Build cost**: 5 × iron plate
- **Production rate**: 30 items / minute
- **Required field**: `recipe`

```yaml
resources:
  builder:
    my_builder:
      recipe: "iron_plate"
      outputs:
        - target: "inventory"
```

---

### Assembler

Combines multiple inputs into complex components (explorers, solar panels, etc.).

- **YAML type**: `assembler`
- **Build cost**: 10 × iron plate + 5 × iron block
- **Production rate**: 30 items / minute
- **Required field**: `recipe`

```yaml
resources:
  assembler:
    my_assembler:
      recipe: "explorer"
      outputs:
        - target: "inventory"
```

---

## Machine Status

Each machine shows a status indicator in the Factory Flow panel:

| Status | Colour | Meaning |
|--------|--------|---------|
| GREEN  | 🟢     | Running normally |
| YELLOW | 🟡     | Output slot full — downstream is blocked |
| RED    | 🔴     | Missing inputs — starved or recipe error |

## Routes

The `outputs` list on each machine defines where produced items are sent.

```yaml
outputs:
  - target: "inventory"          # send to planet inventory
  - target: "smelter.my_smelter.inputs.iron_ore"  # send to another machine's input slot
```

Route target format: `<machine_type>.<machine_name>.inputs.<slot_name>`
