# Nodes

Resource nodes are deposits on the planet surface from which miners extract raw materials.
Each new game starts with two discovered nodes. More nodes can be found by deploying **explorers**.

## Discovered Nodes

The **Planet Overview** panel (bottom-right) lists all discovered nodes. Each node shows:

- **ID** — the unique identifier you use in your `node_id` miner field
- **Type** — the kind of resource this node produces

## Node Types

### Iron Ore Node (`iron_ore`)

The basic resource node. A miner placed on an iron ore node extracts **120 iron ore / minute**.

Iron ore is **not** stored in planet inventory — it flows directly from the miner's output slot into downstream machines via routes. This means:

- Iron ore cannot be "stockpiled" in planet inventory.
- A miner must be routed to a smelter (or it will back-fill and pause).

**Starting nodes**: `node_alpha`, `node_beta`

> More node types are planned for future updates (coal, copper, titanium...).

## Explorers

Each **explorer** you build adds to your explorer count shown in Planet Overview.
Explorers automatically scan the planet and will eventually discover new nodes.

- Build cost: 1 × iron_block + 1 × iron_wheel (assembled)
- Build with: **assembler** using recipe `explorer`

```yaml
resources:
  assembler:
    scout_1:
      recipe: "explorer"
      outputs:
        - target: "inventory"
```

> Note: In the current version, explorers accumulate as a counter. Automated node discovery via explorers is coming in a future update.

## Using Nodes in YAML

Reference a node by its ID in any miner definition:

```yaml
resources:
  miner:
    extractor_alpha:
      node_id: "node_alpha"
      outputs:
        - target: "smelter.furnace_1.inputs.iron_ore"
    extractor_beta:
      node_id: "node_beta"
      outputs:
        - target: "smelter.furnace_2.inputs.iron_ore"
```

You can place multiple miners on the same node — each runs independently.

## Validation

The game validates that:

- The `node_id` referenced in a miner **must** be in your discovered nodes list.
- Attempting to apply a miner with an undiscovered node will fail with a validation error.
