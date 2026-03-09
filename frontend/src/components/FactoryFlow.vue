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
          <MachineNode v-bind="nodeProps" @node-click="openMachineDetail" />
        </template>

        <!-- Controls -->
        <Controls />
        <Background pattern-color="#21262d" gap="20" />
      </VueFlow>
    </div>

    <!-- Machine detail drawer -->
    <div v-if="selectedMachine" class="machine-drawer" @click.self="selectedMachine = null">
      <div class="drawer-panel">
        <div class="drawer-header">
          <span class="status-dot" :class="selectedMachine.status || 'IDLE'"></span>
          <span class="drawer-title">{{ selectedMachine.label || selectedMachine.id }}</span>
          <span class="badge" :class="typeBadgeClass(selectedMachine.type)">{{ selectedMachine.type }}</span>
          <div class="spacer"></div>
          <button class="close-btn" @click="selectedMachine = null">✕</button>
        </div>
        <div class="drawer-body">
          <!-- Info rows -->
          <div v-if="selectedMachine.recipe" class="info-row">
            <span class="info-label">Recipe</span>
            <span class="info-val recipe-val">{{ selectedMachine.recipe }}</span>
          </div>
          <div v-if="selectedMachine.node_id" class="info-row">
            <span class="info-label">Node</span>
            <span class="info-val">{{ selectedMachine.node_id }}</span>
          </div>
          <div v-if="selectedMachine.factory_id" class="info-row">
            <span class="info-label">Factory</span>
            <span class="info-val">{{ selectedMachine.factory_id }}</span>
          </div>
          <div v-if="selectedMachine.power_usage_mw > 0" class="info-row">
            <span class="info-label">Power draw</span>
            <span class="info-val power-val">⚡ {{ selectedMachine.power_usage_mw }} MW</span>
          </div>
          <div class="info-row">
            <span class="info-label">Status</span>
            <span class="info-val" :class="`status-val-${(selectedMachine.status || 'IDLE').toLowerCase()}`">
              {{ selectedMachine.status || 'IDLE' }}
            </span>
          </div>
          <div v-if="selectedMachine.built_at" class="info-row">
            <span class="info-label">Built at</span>
            <span class="info-val">{{ formatTime(selectedMachine.built_at) }}</span>
          </div>

          <!-- Input slots -->
          <div v-if="hasInputSlots" class="slot-section">
            <div class="slot-section-title">INPUT SLOTS</div>
            <div v-for="(slot, key) in selectedMachine.input_slots" :key="`in-${key}`" class="slot-detail slot-in">
              <span class="slot-name">{{ key }}</span>
              <div class="slot-bar-wrap">
                <div class="slot-bar" :style="{ width: slotPct(slot) + '%' }"></div>
              </div>
              <span class="slot-count">{{ slot.count }}/{{ slot.capacity }}</span>
            </div>
          </div>

          <!-- Output slots -->
          <div v-if="hasOutputSlots" class="slot-section">
            <div class="slot-section-title">OUTPUT SLOTS</div>
            <div v-for="(slot, key) in selectedMachine.output_slots" :key="`out-${key}`" class="slot-detail slot-out">
              <span class="slot-name">{{ key }}</span>
              <div class="slot-bar-wrap">
                <div class="slot-bar slot-bar-out" :style="{ width: slotPct(slot) + '%' }"></div>
              </div>
              <span class="slot-count">{{ slot.count }}/{{ slot.capacity }}</span>
            </div>
          </div>

          <!-- Routes -->
          <div v-if="selectedMachine.routes && selectedMachine.routes.length" class="slot-section">
            <div class="slot-section-title">ROUTES</div>
            <div v-for="(route, i) in selectedMachine.routes" :key="i" class="route-row">
              <span class="route-arrow">→</span>
              <span class="route-target">{{ route.target }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Controls } from '@vue-flow/controls'
import { Background } from '@vue-flow/background'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'
import MachineNode from './MachineNode.vue'
import { useGameStore } from '../stores/game.js'

const store = useGameStore()
const { fitView } = useVueFlow()
const selectedMachine = ref(null)

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
        machineKey: key,
        id: machine.id,
        label: machine.id,
        type: machine.type || 'unknown',
        status: machine.status || 'IDLE',
        recipe: machine.recipe || '',
        node_id: machine.node_id || '',
        factory_id: machine.factory_id || '',
        power_usage_mw: machine.power_usage_mw || 0,
        built_at: machine.built_at || null,
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
      power_usage_mw: 0,
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
      let targetId
      if (target === 'inventory') {
        targetId = '__inventory__'
      } else {
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

function openMachineDetail(data) {
  selectedMachine.value = data
}

function onNodesReady() {
  setTimeout(() => fitView({ padding: 0.1 }), 50)
}

watch(
  () => store.machines,
  () => {
    setTimeout(() => fitView({ padding: 0.1 }), 100)
    // Keep selected machine data up-to-date
    if (selectedMachine.value) {
      const key = selectedMachine.value.machineKey
      const updated = store.machines[key]
      if (updated) {
        selectedMachine.value = {
          ...selectedMachine.value,
          status: updated.status,
          input_slots: updated.input_slots || {},
          output_slots: updated.output_slots || {}
        }
      }
    }
  },
  { deep: false }
)

// Computed helpers for the drawer
const hasInputSlots = computed(() =>
  selectedMachine.value && Object.keys(selectedMachine.value.input_slots || {}).length > 0
)
const hasOutputSlots = computed(() =>
  selectedMachine.value && Object.keys(selectedMachine.value.output_slots || {}).length > 0
)

function slotPct(slot) {
  if (!slot || slot.capacity === 0) return 0
  return Math.min(Math.round((slot.count / slot.capacity) * 100), 100)
}

function typeBadgeClass(type) {
  const map = {
    miner: 'badge-blue',
    smelter: 'badge-purple',
    builder: 'badge-green',
    assembler: 'badge-yellow',
    inventory: 'badge-green'
  }
  return map[type] || 'badge-blue'
}

function formatTime(ts) {
  try {
    return new Date(ts).toLocaleTimeString()
  } catch {
    return ts
  }
}
</script>

<style scoped>
.flow-wrap {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  position: relative;
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

/* Machine detail drawer */
.machine-drawer {
  position: absolute;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  padding: 8px;
  pointer-events: none;
}

.drawer-panel {
  width: 240px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  pointer-events: all;
  max-height: calc(100% - 16px);
  display: flex;
  flex-direction: column;
}

.drawer-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.drawer-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
  cursor: pointer;
  padding: 0 2px;
  transition: color 0.15s;
  flex-shrink: 0;
}

.close-btn:hover { color: var(--color-red); }

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* Info rows */
.info-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
  font-size: 11px;
}

.info-label {
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-size: 9px;
  flex-shrink: 0;
}

.info-val { color: var(--text-primary); }
.recipe-val { color: var(--color-cyan); }
.power-val { color: var(--color-yellow); }

.status-val-green { color: var(--color-green); }
.status-val-yellow { color: var(--color-yellow); }
.status-val-red { color: var(--color-red); }
.status-val-idle { color: var(--text-muted); }

/* Slot sections */
.slot-section {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.slot-section-title {
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  padding-bottom: 3px;
  border-bottom: 1px solid var(--border-color);
}

.slot-detail {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
}

.slot-in .slot-name { color: var(--color-blue); }
.slot-out .slot-name { color: var(--color-green); }

.slot-bar-wrap {
  flex: 1;
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.slot-bar {
  height: 100%;
  background: var(--color-blue);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.slot-bar-out {
  background: var(--color-green);
}

.slot-count {
  font-size: 10px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* Routes */
.route-row {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
}

.route-arrow { color: var(--color-cyan); }
.route-target {
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
