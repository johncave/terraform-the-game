<template>
  <div class="flow-wrap">
    <div class="panel-header">
      <span class="panel-title">FACTORY FLOW</span>
      <div class="spacer"></div>
      <span class="machine-count">{{ Object.keys(store.machines).length }} machines</span>
    </div>

    <!-- Empty state -->
    <div v-if="!hasMachines" class="empty-flow">
      <div class="empty-icon">⬡</div>
      <p>Apply your first factory to see the visualization</p>
      <p class="empty-hint">Use the editor to configure machines, then click "Apply"</p>
    </div>

    <!-- Vue Flow -->
    <div v-else class="vue-flow-container">
      <VueFlow
        :nodes="flowNodes"
        :edges="flowEdges"
        :fit-view-on-init="true"
        :nodes-draggable="true"
        :zoom-on-scroll="true"
        :pan-on-drag="true"
        :default-edge-options="defaultEdgeOptions"
        class="flow-canvas"
        @nodes-initialized="onNodesReady"
      >
        <template #node-machine="nodeProps">
          <MachineNode v-bind="nodeProps" />
        </template>

        <!-- Controls -->
        <Controls />
        <Background pattern-color="#21262d" gap="20" />
      </VueFlow>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Controls } from '@vue-flow/controls'
import { Background } from '@vue-flow/background'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'
import MachineNode from './MachineNode.vue'
import { useGameStore } from '../stores/game.js'

const store = useGameStore()
const { fitView } = useVueFlow()

const defaultEdgeOptions = {
  type: 'smoothstep',
  animated: true,
  style: { stroke: '#58a6ff', strokeWidth: 1.5, opacity: 0.6 }
}

const hasMachines = computed(() => Object.keys(store.machines).length > 0)

// Layout columns by machine type
const TYPE_COLUMNS = {
  miner: 0,
  smelter: 1,
  builder: 2,
  assembler: 3,
  factory: 2
}
const COL_WIDTH = 220
const ROW_HEIGHT = 120

function getColumn(type) {
  return TYPE_COLUMNS[type] ?? 2
}

const flowNodes = computed(() => {
  const machines = store.machines
  const colCounters = {}

  const machineNodes = Object.entries(machines).map(([key, machine]) => {
    const type = machine.type || 'miner'
    const col = getColumn(type)
    colCounters[col] = (colCounters[col] || 0)
    const row = colCounters[col]
    colCounters[col]++

    return {
      id: key,
      type: 'machine',
      position: { x: col * COL_WIDTH + 20, y: row * ROW_HEIGHT + 20 },
      data: {
        id: machine.id,
        label: machine.id,
        type: machine.type || 'unknown',
        status: machine.status || 'IDLE',
        input_slots: machine.input_slots || {},
        output_slots: machine.output_slots || {},
        routes: machine.routes || []
      }
    }
  })

  // Add inventory node at the far right
  const maxCol = Math.max(...Object.keys(colCounters).map(Number), 3)
  machineNodes.push({
    id: '__inventory__',
    type: 'machine',
    position: { x: (maxCol + 1) * COL_WIDTH + 20, y: 20 },
    data: {
      id: 'inventory',
      label: 'INVENTORY',
      type: 'inventory',
      status: 'GREEN',
      input_slots: buildInventorySlots(),
      output_slots: {}
    }
  })

  return machineNodes
})

function buildInventorySlots() {
  const items = store.inventory
  const slots = {}
  for (const [k, v] of Object.entries(items)) {
    if (v > 0) {
      slots[k] = { item_type: k, count: v, capacity: 1000 }
    }
  }
  return slots
}

const flowEdges = computed(() => {
  const machines = store.machines
  const edges = []

  for (const [key, machine] of Object.entries(machines)) {
    for (const route of machine.routes || []) {
      const target = route.target || ''
      // Format: "type.id.inputs.item" or "inventory"
      let targetId
      if (target === 'inventory') {
        targetId = '__inventory__'
      } else {
        // "smelter.iron_processor.inputs.iron_ore" → key is "smelter.iron_processor"
        const parts = target.split('.')
        if (parts.length >= 2) {
          targetId = `${parts[0]}.${parts[1]}`
        } else {
          targetId = target
        }
      }

      edges.push({
        id: `${key}->${targetId}`,
        source: key,
        target: targetId,
        label: route.item || undefined
      })
    }
  }

  return edges
})

function onNodesReady() {
  setTimeout(() => fitView({ padding: 0.1 }), 50)
}

watch(
  () => store.machines,
  () => {
    setTimeout(() => fitView({ padding: 0.1 }), 100)
  },
  { deep: false }
)
</script>

<style scoped>
.flow-wrap {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  flex-shrink: 0;
}

.panel-title { color: var(--color-cyan); }
.spacer { flex: 1; }

.machine-count {
  font-size: 10px;
  color: var(--text-muted);
}

/* Empty state */
.empty-flow {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-muted);
  font-size: 13px;
  text-align: center;
  padding: 20px;
}

.empty-icon {
  font-size: 48px;
  opacity: 0.15;
  color: var(--color-blue);
}

.empty-hint {
  font-size: 11px;
  color: var(--text-muted);
  opacity: 0.6;
}

/* Flow canvas */
.vue-flow-container {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.flow-canvas {
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
}

/* Override vue-flow styles for dark theme */
:deep(.vue-flow__controls) {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

:deep(.vue-flow__controls-button) {
  background: var(--bg-tertiary);
  border-color: var(--border-color);
  color: var(--text-secondary);
  fill: var(--text-secondary);
}

:deep(.vue-flow__controls-button:hover) {
  background: var(--bg-panel);
  color: var(--color-blue);
  fill: var(--color-blue);
}

:deep(.vue-flow__background) {
  background: var(--bg-primary);
}
</style>
