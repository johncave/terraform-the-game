# Getting Started

Welcome to **Terraform: The Game** — a factory-building game played through infrastructure-as-code.
Your goal is to terraform a planet by automating resource extraction and processing through a network of machines.

## Your First Session

When you create a new game you receive a starter inventory of **20 iron plates** — enough to build your first factory chain.

### The Genesis Factory

The simplest factory that produces iron plates indefinitely:

```yaml
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
```

**Step by step:**

1. Paste (or type) the YAML above into the Factory Editor.
2. Set the factory name to `genesis` in the name field.
3. Click **Plan** — this validates the configuration and shows what will be built.
4. Click **Apply** — this deducts build costs and registers the machines.
5. Watch the **Factory Flow** panel to see machines come online (there is a 60-second warm-up delay after apply).
6. Check the **Planet Overview** panel (bottom-right) as iron plates accumulate in inventory.

### Build Costs

| Machine  | Cost            |
|----------|-----------------|
| Miner    | Free            |
| Smelter  | 3 × iron plate  |
| Builder  | 5 × iron plate  |
| Assembler| 10 × iron plate + 5 × iron block |

The Genesis Factory costs **3 + 5 = 8 iron plates**, leaving you with 12 in reserve.

## Tips

- You can have multiple factories. Each factory is an independent YAML file identified by its name.
- Applying the same factory name again updates it (existing machines are re-used; new ones are built).
- Machines take **60 seconds** to become operational after apply.
- The game auto-polls every second while the tab is open. Pause polling with the **Pause** button to avoid interruptions while editing YAML.
