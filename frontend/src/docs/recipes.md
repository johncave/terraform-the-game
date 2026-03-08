# Recipes

Recipes define what a processing machine (smelter, builder, assembler) produces.
Specify the recipe name via the `recipe` field in your YAML.

## Reference

### iron_ingot

Smelt raw iron ore into ingots.

| | Item | Qty |
|---|---|---|
| **Input** | iron_ore | 1 |
| **Output** | iron_ingot | 1 |

- Compatible machines: **smelter**
- Rate: 30 / min

---

### iron_plate

Press ingots into flat plates. The most fundamental construction material.

| | Item | Qty |
|---|---|---|
| **Input** | iron_ingot | 1 |
| **Output** | iron_plate | 1 |

- Compatible machines: **builder**
- Rate: 30 / min

---

### iron_block

Cast ingots into solid blocks.

| | Item | Qty |
|---|---|---|
| **Input** | iron_ingot | 1 |
| **Output** | iron_block | 1 |

- Compatible machines: **builder**
- Rate: 30 / min

---

### iron_wheel

Forge blocks into wheels (used for explorer chassis).

| | Item | Qty |
|---|---|---|
| **Input** | iron_block | 1 |
| **Output** | iron_wheel | 1 |

- Compatible machines: **builder**
- Rate: 30 / min

---

### solar_panel

Assemble a solar panel for power generation.

| | Item | Qty |
|---|---|---|
| **Input** | iron_plate | 1 |
| **Input** | iron_block | 1 |
| **Output** | solar_panel | 1 |

- Compatible machines: **assembler**
- Rate: 30 / min
- Effect: +1 MW power generation per panel built

---

### explorer

Build an autonomous explorer rover.

| | Item | Qty |
|---|---|---|
| **Input** | iron_block | 1 |
| **Input** | iron_wheel | 1 |
| **Output** | explorer | 1 |

- Compatible machines: **assembler**
- Rate: 30 / min
- Effect: Explorers discover new resource nodes over time

---

## Production Chain

```
iron_ore  ──[smelter]──▶  iron_ingot
                              │
              ┌───────────────┤
              │               │
          [builder]       [builder]
          iron_plate       iron_block
              │               │
              │        ┌──────┴──────┐
              │    [builder]     [assembler]
              │   iron_wheel    solar_panel
              │        │
              └─── [assembler] ──▶ explorer
```
