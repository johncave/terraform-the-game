<template>
  <div class="dashboard">
    <!-- Top bar -->
    <header class="topbar">
      <div class="topbar-left">
        <span class="logo">⬡ TERRAFORM</span>
        <span class="sep">:</span>
        <span class="game-id" :title="store.gameId">{{ store.gameId }}</span>
      </div>
      <div class="topbar-center">
        <!-- Factory tabs -->
        <div class="factory-tabs">
          <button
            v-for="fac in visibleFactories"
            :key="fac.id"
            class="factory-tab"
            :class="{ active: store.activeFactoryId === fac.id }"
            @click="store.activeFactoryId = fac.id"
          >
            <span class="status-dot" :class="fac.status || 'IDLE'"></span>
            {{ fac.id }}
          </button>
          <button class="factory-tab" :class="{ active: store.activeFactoryId === 'main' }" @click="store.activeFactoryId = 'main'">
            <span class="status-dot IDLE"></span>
            main
          </button>
        </div>
      </div>
      <div class="topbar-right">
        <span class="poll-indicator" :class="{ active: store.polling }">
          {{ store.polling ? '● LIVE' : '○ PAUSED' }}
        </span>
        <button class="btn btn-sm" @click="store.polling ? store.stopPolling() : store.startPolling()">
          {{ store.polling ? 'Pause' : 'Resume' }}
        </button>
        <button class="btn btn-sm btn-codex" @click="showCodex = true">? Codex</button>
        <button class="btn btn-sm" @click="handleExit">Exit</button>
      </div>
    </header>

    <!-- Main 2x2 Grid -->
    <main class="grid">
      <!-- Top Left: Factory Editor -->
      <div class="cell cell-editor">
        <FactoryEditor />
      </div>

      <!-- Top Right: Factory Flow -->
      <div class="cell cell-flow">
        <FactoryFlow />
      </div>

      <!-- Bottom Left: Terminal -->
      <div class="cell cell-terminal">
        <Terminal />
      </div>

      <!-- Bottom Right: Planet Overview -->
      <div class="cell cell-inventory">
        <Inventory />
      </div>
    </main>

    <!-- Codex overlay -->
    <Codex v-if="showCodex" @close="showCodex = false" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGameStore } from '../stores/game.js'
import FactoryEditor from '../components/FactoryEditor.vue'
import FactoryFlow from '../components/FactoryFlow.vue'
import Terminal from '../components/Terminal.vue'
import Inventory from '../components/Inventory.vue'
import Codex from './Codex.vue'

const route = useRoute()
const router = useRouter()
const store = useGameStore()
const showCodex = ref(false)

const visibleFactories = computed(() => {
  return (store.factories || []).filter((f) => f.id !== 'main')
})

onMounted(async () => {
  const routeId = route.params.id
  if (routeId && routeId !== store.gameId) {
    await store.loadGame(routeId)
  } else if (store.gameId) {
    await store.refreshState()
    store.startPolling()
  } else {
    router.push('/')
  }
})

onUnmounted(() => {
  store.stopPolling()
})

function handleExit() {
  store.stopPolling()
  router.push('/')
}
</script>

<style scoped>
.dashboard {
  height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  overflow: hidden;
}

/* Top bar */
.topbar {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  gap: 12px;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.logo {
  color: var(--color-blue);
  font-weight: 700;
  font-size: 13px;
  letter-spacing: 0.05em;
}

.sep {
  color: var(--text-muted);
}

.game-id {
  font-size: 11px;
  color: var(--text-muted);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: default;
}

.topbar-center {
  flex: 1;
  display: flex;
  justify-content: center;
}

.factory-tabs {
  display: flex;
  gap: 4px;
}

.factory-tab {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 10px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s;
}

.factory-tab:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.factory-tab.active {
  background: var(--bg-tertiary);
  border-color: var(--color-blue);
  color: var(--color-blue);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.poll-indicator {
  font-size: 10px;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}

.poll-indicator.active {
  color: var(--color-green);
  animation: pulse-text 2s ease-in-out infinite;
}

@keyframes pulse-text {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.btn-sm {
  padding: 3px 10px;
  font-size: 11px;
}

.btn-codex {
  background: rgba(188, 140, 255, 0.08);
  border-color: var(--color-purple);
  color: var(--color-purple);
}

.btn-codex:hover {
  background: rgba(188, 140, 255, 0.15);
}

/* 2x2 grid */
.grid {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 4px;
  padding: 4px;
  overflow: hidden;
}

.cell {
  overflow: hidden;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  background: var(--bg-secondary);
  display: flex;
  flex-direction: column;
}
</style>
