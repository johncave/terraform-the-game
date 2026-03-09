<template>
  <div class="inventory-wrap">
    <div class="panel-header">
      <span class="panel-title">PLANET OVERVIEW</span>
      <div class="spacer"></div>
      <span class="power-stat" title="Total power available">
        ⚡ {{ powerAvailableMW }} MW
      </span>
      <span v-if="store.gameState?.power_generation > 0" class="solar-stat" title="Solar panels built">
        ☀ {{ store.gameState.power_generation }}
      </span>
      <span class="explorer-stat" title="Explorers deployed">
        🤖 {{ store.gameState?.explorers_built ?? 0 }}
      </span>
    </div>

    <div class="inventory-body">
      <!-- Items -->
      <section class="section">
        <div class="section-header">
          <span>RESOURCES</span>
          <span class="item-count">{{ itemCount }} types</span>
        </div>

        <div v-if="hasItems" class="items-list">
          <div v-for="(count, item) in nonZeroItems" :key="item" class="item-row">
            <div class="item-header">
              <span class="item-name">{{ formatItemName(item) }}</span>
              <span class="item-count-val" :class="countClass(count)">
                {{ count.toLocaleString() }} / 1000
              </span>
            </div>
            <div class="progress-track">
              <div
                class="progress-fill"
                :class="barClass(count)"
                :style="{ width: barWidth(count) }"
              ></div>
            </div>
          </div>
        </div>

        <div v-else class="empty-section">
          <span class="empty-icon">□</span>
          No resources collected yet
        </div>
      </section>

      <!-- All items (zero counts) -->
      <section v-if="zeroItems.length > 0" class="section">
        <div class="section-header">
          <span>AWAITING PRODUCTION</span>
        </div>
        <div class="zero-items">
          <span v-for="item in zeroItems" :key="item" class="zero-chip">
            {{ formatItemName(item) }}
          </span>
        </div>
      </section>

      <!-- Power Grid -->
      <section class="section">
        <div class="section-header">
          <span>POWER GRID</span>
          <span
            v-if="store.gameState?.power_tripped"
            class="trip-badge"
          >⚠ TRIPPED</span>
        </div>
        <div class="power-stats">
          <div class="power-row">
            <span class="power-label">Available</span>
            <span class="power-val power-green">⚡ {{ powerAvailableMW }} MW</span>
          </div>
          <div class="power-row">
            <span class="power-label">Consumption</span>
            <span
              class="power-val"
              :class="powerOverload ? 'power-red' : 'power-yellow'"
            >
              ⚡ {{ powerConsumptionMW }} MW
            </span>
          </div>
          <div class="power-progress-track">
            <div
              class="power-progress-fill"
              :class="powerOverload ? 'bar-red' : 'bar-green'"
              :style="{ width: powerUsagePct + '%' }"
            ></div>
          </div>
          <div class="power-row power-breakdown">
            <span class="power-label">Base</span>
            <span class="power-val">10 MW</span>
          </div>
          <div v-if="store.gameState?.power_generation > 0" class="power-row power-breakdown">
            <span class="power-label">Solar panels</span>
            <span class="power-val power-green">+{{ store.gameState.power_generation * 5 }} MW ({{ store.gameState.power_generation }}×)</span>
          </div>
        </div>
        <button
          v-if="store.gameState?.power_tripped"
          class="btn btn-danger btn-sm reset-btn"
          @click="handlePowerReset"
          :disabled="resetting"
        >
          <span v-if="resetting" class="spinner">◌</span>
          <span v-else>⚡</span>
          Reset Power Grid
        </button>
      </section>
      <section class="section">
        <div class="section-header">
          <span>DISCOVERED NODES</span>
          <span class="item-count">{{ store.discoveredNodes.length }}</span>
        </div>
        <div v-if="store.discoveredNodes.length > 0" class="nodes-list">
          <div v-for="node in store.discoveredNodes" :key="node" class="node-row">
            <span class="node-icon">◈</span>
            <span class="node-id">{{ node }}</span>
            <span class="node-type-badge" :class="`node-type-${store.nodeTypes[node] || 'unknown'}`">
              {{ formatNodeType(store.nodeTypes[node]) }}
            </span>
          </div>
        </div>
        <div v-else class="empty-section">
          <span class="empty-icon">◎</span>
          No nodes discovered
        </div>
      </section>

      <!-- Last tick -->
      <div v-if="store.gameState?.last_tick" class="last-tick">
        <span class="label">last tick:</span>
        <span class="val">{{ formatTick(store.gameState.last_tick) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useGameStore } from '../stores/game.js'

const store = useGameStore()
const resetting = ref(false)

const MAX_CAPACITY = 1000
const BASE_POWER_MW = 10
const SOLAR_GEN_MW = 5

const hasItems = computed(() => Object.values(store.inventory).some((v) => v > 0))

const nonZeroItems = computed(() => {
  const result = {}
  for (const [k, v] of Object.entries(store.inventory)) {
    if (v > 0) result[k] = v
  }
  return result
})

const zeroItems = computed(() => {
  return Object.entries(store.inventory)
    .filter(([, v]) => v === 0)
    .map(([k]) => k)
})

const itemCount = computed(() => Object.keys(store.inventory).length)

const powerAvailableMW = computed(() => {
  const solar = store.gameState?.power_generation || 0
  return BASE_POWER_MW + solar * SOLAR_GEN_MW
})

const powerConsumptionMW = computed(() => {
  return store.gameState?.power_consumption_mw || 0
})

const powerOverload = computed(() => powerConsumptionMW.value > powerAvailableMW.value)

const powerUsagePct = computed(() => {
  if (powerAvailableMW.value === 0) return 100
  return Math.min(Math.round((powerConsumptionMW.value / powerAvailableMW.value) * 100), 100)
})

function formatItemName(key) {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

function formatNodeType(type) {
  if (!type) return 'unknown'
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

function barWidth(count) {
  return `${Math.min((count / MAX_CAPACITY) * 100, 100)}%`
}

function barClass(count) {
  const pct = count / MAX_CAPACITY
  if (pct >= 0.9) return 'bar-red'
  if (pct >= 0.7) return 'bar-yellow'
  return 'bar-green'
}

function countClass(count) {
  const pct = count / MAX_CAPACITY
  if (pct >= 0.9) return 'count-red'
  if (pct >= 0.7) return 'count-yellow'
  return 'count-green'
}

function formatTick(tick) {
  try {
    return new Date(tick).toLocaleTimeString()
  } catch {
    return tick
  }
}

async function handlePowerReset() {
  resetting.value = true
  try {
    await store.resetPowerGrid()
  } finally {
    resetting.value = false
  }
}
</script>

<style scoped>
.inventory-wrap {
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

.panel-title { color: var(--color-purple); }
.spacer { flex: 1; }

.power-stat {
  color: var(--color-yellow);
  font-size: 11px;
}

.solar-stat {
  color: var(--color-yellow);
  font-size: 11px;
}

.explorer-stat {
  color: var(--color-cyan);
  font-size: 11px;
}

.inventory-body {
  flex: 1;
  overflow-y: auto;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Section */
.section { display: flex; flex-direction: column; gap: 8px; }

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  padding-bottom: 4px;
  border-bottom: 1px solid var(--border-color);
}

.item-count { font-size: 9px; color: var(--text-muted); }

/* Items */
.items-list { display: flex; flex-direction: column; gap: 8px; }

.item-row { display: flex; flex-direction: column; gap: 3px; }

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.item-name {
  font-size: 12px;
  color: var(--text-primary);
}

.item-count-val {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.count-green { color: var(--color-green); }
.count-yellow { color: var(--color-yellow); }
.count-red { color: var(--color-red); }

/* Progress bar */
.progress-track {
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.4s ease;
}

.bar-green { background: var(--color-green); }
.bar-yellow { background: var(--color-yellow); }
.bar-red { background: var(--color-red); }

/* Zero items */
.zero-items {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.zero-chip {
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 3px;
  padding: 2px 6px;
}

/* Nodes */
.nodes-list { display: flex; flex-direction: column; gap: 4px; }

.node-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  padding: 4px 6px;
  background: rgba(57, 197, 207, 0.05);
  border: 1px solid rgba(57, 197, 207, 0.15);
  border-radius: var(--radius-sm);
}

.node-icon {
  color: var(--color-cyan);
  opacity: 0.6;
  flex-shrink: 0;
}

.node-id {
  color: var(--color-cyan);
  flex: 1;
}

.node-type-badge {
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 1px 5px;
  border-radius: 3px;
  white-space: nowrap;
}

.node-type-iron_ore {
  background: rgba(248, 140, 0, 0.12);
  color: #f8a030;
  border: 1px solid rgba(248, 140, 0, 0.25);
}

.node-type-unknown {
  background: var(--bg-tertiary);
  color: var(--text-muted);
  border: 1px solid var(--border-color);
}

/* Empty state */
.empty-section {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: var(--text-muted);
  padding: 8px 0;
}

.empty-icon { opacity: 0.4; }

/* Power grid */
.power-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.power-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
}

.power-breakdown {
  font-size: 10px;
  opacity: 0.7;
}

.power-label {
  color: var(--text-muted);
  text-transform: uppercase;
  font-size: 9px;
  letter-spacing: 0.05em;
}

.power-val { font-variant-numeric: tabular-nums; }
.power-green { color: var(--color-green); }
.power-yellow { color: var(--color-yellow); }
.power-red { color: var(--color-red); font-weight: 600; }

.power-progress-track {
  height: 5px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
  margin: 2px 0;
}

.power-progress-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.4s ease;
}

.trip-badge {
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--color-red);
  background: rgba(248, 81, 73, 0.1);
  border: 1px solid rgba(248, 81, 73, 0.3);
  border-radius: 3px;
  padding: 1px 5px;
  animation: pulse-text 1.5s ease-in-out infinite;
}

@keyframes pulse-text {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.reset-btn {
  margin-top: 6px;
  width: 100%;
  justify-content: center;
  font-size: 11px;
  padding: 5px 10px;
}

.btn-sm { padding: 4px 10px; font-size: 11px; }

.spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* Last tick */
.last-tick {
  display: flex;
  gap: 8px;
  font-size: 10px;
  color: var(--text-muted);
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid var(--border-color);
}

.last-tick .label { text-transform: uppercase; letter-spacing: 0.05em; }
.last-tick .val { color: var(--text-secondary); }
</style>
