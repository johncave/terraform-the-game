# Terraform: The Game

A multi-tenant, DevOps-inspired factory simulation game. Players manage interplanetary infrastructure by writing YAML configurations that mimic Terraform's **Plan** and **Apply** workflow.

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22 · REST API · Tick-based simulation engine |
| Frontend | Vue 3 · Vite · Monaco Editor · Vue Flow (DAG) |
| Database | PostgreSQL 16 · Event-sourced state + snapshots |
| Infra | Docker Compose |

---

## Quick Start

### Prerequisites
- Docker & Docker Compose v2

### 1. Start Everything

```bash
docker-compose up --build
```

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| PostgreSQL | localhost:5432 |

### 2. Development Mode (without Docker)

**Backend:**
```bash
cd backend
export POSTGRES_HOST=localhost POSTGRES_USER=terraform POSTGRES_PASSWORD=terraform POSTGRES_DB=terraform_game
go run .
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev   # http://localhost:5173
```

---

## Game Mechanics

### Core Rules
- Every game instance has a unique `game_id` (UUID). State, inventory, and events are isolated per game.
- Items have a `stack_height` of **100**. The Planet Inventory holds a maximum of **10 stacks per item type** (1,000 items total per type).

### Machine Types

| Machine | Inputs | Output | Rate |
|---|---|---|---|
| Miner | 0 (assigned to a node) | 1 (iron_ore) | 120 items/min |
| Smelter | 1 input slot | 1 output slot | 30 items/min |
| Builder | 1 input slot | 1 output slot | 30 items/min |
| Assembler | **2 input slots** | 1 output slot | 30 items/min |

### Machine Statuses
- 🟢 **GREEN** – All input slots have items; output slot has free capacity
- 🟡 **YELLOW** – Output slot is full (throttled)
- 🔴 **RED** – One or more input slots are empty (starved)

### Recipes
```
iron_ore       → (smelter, recipe: iron_ingot) → iron_ingot
iron_ingot     → (builder, recipe: iron_plate) → iron_plate
iron_ingot     → (builder, recipe: iron_block) → iron_block
iron_block     → (builder, recipe: iron_wheel) → iron_wheel
iron_plate + iron_block → (assembler, recipe: solar_panel) → solar_panel
iron_block + iron_wheel → (assembler, recipe: explorer)    → explorer
```

---

## YAML DSL

Split your setup into multiple named **factories**, each with its own YAML spec.

### The Genesis YAML

Copy-paste this into the editor, then click **Plan** then **Apply**:

```yaml
# Terraform: The Game - Genesis Configuration
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
        - target: "inventory"   # routes to planet storage
```

### Multi-slot Assembler Example

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
  assembler:
    solar_panel_factory:
      recipe: "solar_panel"
      inputs:
        iron_plate: "inventory"   # pulls from planet storage
        iron_block: "inventory"
      outputs:
        - target: "inventory"
```

---

## API Reference

| Method | Path | Description |
|---|---|---|
| `POST` | `/games` | Create a new game → `{ game_id }` |
| `GET` | `/games` | List all game IDs |
| `GET` | `/games/:id/state` | Full game state |
| `GET` | `/games/:id/inventory` | Planet inventory |
| `GET` | `/games/:id/factory` | List factories |
| `GET` | `/games/:id/factory/:fid` | Get factory details |
| `POST` | `/games/:id/factory/:fid/plan` | Validate YAML → plan output |
| `POST` | `/games/:id/factory/:fid/apply` | Apply YAML → deduct costs, build machines |
| `DELETE` | `/games/:id/factory/:fid` | Destroy a factory |

---

## Validation Walkthrough

### Step 1: Observe RED State (Assembler starved)

Apply the multi-slot assembler YAML above. The `solar_panel_factory` assembler will immediately show **RED** status because:
- It pulls `iron_plate` and `iron_block` from inventory
- On a fresh game, only `iron_ore: 500` exists — no plates or blocks yet

Watch the GET state endpoint:
```bash
curl http://localhost:8080/games/<game_id>/state | jq '.machines["assembler.solar_panel_factory"].status'
# "RED"
```

### Step 2: Watch the Pipeline Fill

As the miner + smelter + builder pipeline processes iron ore into plates, the inventory fills up. Once `iron_plate > 0` and `iron_block > 0`, the assembler transitions to **GREEN**.

### Step 3: Verify Inventory Cap

The planet inventory cap is **1,000 items per type** (10 stacks × 100 items). When `iron_ingot` hits 1,000, the smelter output slot fills (100 items), status switches to **YELLOW** (throttled), and the miner downstream also backs up.

```bash
curl http://localhost:8080/games/<game_id>/inventory | jq
# { "items": { "iron_ore": 500, "iron_ingot": 1000 } }
```

### Step 4: Build Costs

Machines are not free. Build costs are deducted on `apply`:

| Machine | Cost |
|---|---|
| Miner | Free |
| Smelter | 3× iron_plate |
| Builder | 5× iron_plate |
| Assembler | 10× iron_plate + 5× iron_block |

---

## Project Structure

```
terraform-the-game/
├── docker-compose.yml
├── backend/
│   ├── Dockerfile
│   ├── main.go
│   ├── go.mod
│   ├── api/           # HTTP handlers & router
│   ├── db/            # PostgreSQL layer + migrations
│   ├── engine/        # Tick engine + Active Game Manager
│   ├── models/        # Game structs & recipes
│   └── parser/        # YAML DSL parser + DAG validator
└── frontend/
    ├── Dockerfile
    ├── nginx.conf
    ├── src/
    │   ├── views/     # Landing, Dashboard
    │   ├── components/ # Terminal, Inventory, FactoryEditor, FactoryFlow
    │   ├── stores/    # Pinia game store
    │   └── api/       # API client
    └── vite.config.js
```
