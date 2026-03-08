<template>
  <div class="machine-node" :class="statusClass">
    <!-- Status dot -->
    <div class="node-status">
      <span class="status-dot" :class="data.status || 'IDLE'"></span>
      <span class="node-id">{{ data.label || data.id }}</span>
    </div>

    <!-- Type badge -->
    <div class="node-meta">
      <span class="badge" :class="typeBadgeClass">{{ data.type }}</span>
    </div>

    <!-- Slots -->
    <div v-if="hasSlots" class="slots">
      <div v-for="(slot, key) in data.output_slots" :key="`out-${key}`" class="slot slot-out">
        <span class="slot-key">{{ key }}</span>
        <span class="slot-count">{{ slot.count }}/{{ slot.capacity }}</span>
      </div>
      <div v-for="(slot, key) in data.input_slots" :key="`in-${key}`" class="slot slot-in">
        <span class="slot-key">▶ {{ key }}</span>
        <span class="slot-count">{{ slot.count }}/{{ slot.capacity }}</span>
      </div>
    </div>

    <!-- Vue Flow handles -->
    <Handle type="target" :position="Position.Left" />
    <Handle type="source" :position="Position.Right" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'

const props = defineProps({
  data: {
    type: Object,
    default: () => ({})
  }
})

const statusClass = computed(() => {
  const s = props.data.status || 'IDLE'
  return `status-${s.toLowerCase()}`
})

const typeBadgeClass = computed(() => {
  const map = {
    miner: 'badge-blue',
    smelter: 'badge-purple',
    builder: 'badge-green',
    assembler: 'badge-yellow',
    inventory: 'badge-green'
  }
  return map[props.data.type] || 'badge-blue'
})

const hasSlots = computed(() => {
  return (
    Object.keys(props.data.output_slots || {}).length > 0 ||
    Object.keys(props.data.input_slots || {}).length > 0
  )
})
</script>

<style scoped>
.machine-node {
  background: var(--bg-panel);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 8px 10px;
  min-width: 140px;
  max-width: 180px;
  font-family: var(--font-mono);
  font-size: 11px;
  cursor: default;
  transition: box-shadow 0.2s;
}

.machine-node:hover {
  box-shadow: 0 0 12px rgba(88, 166, 255, 0.15);
}

/* Status border colors */
.status-green { border-color: rgba(63, 185, 80, 0.4); }
.status-yellow { border-color: rgba(210, 153, 34, 0.4); }
.status-red { border-color: rgba(248, 81, 73, 0.4); }
.status-idle { border-color: var(--border-color); }

/* Node status row */
.node-status {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.node-id {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Meta row */
.node-meta {
  margin-bottom: 6px;
}

/* Slots */
.slots {
  display: flex;
  flex-direction: column;
  gap: 3px;
  border-top: 1px solid var(--border-color);
  padding-top: 5px;
  margin-top: 4px;
}

.slot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 10px;
  gap: 6px;
}

.slot-out { color: var(--color-green); }
.slot-in { color: var(--color-blue); }

.slot-key {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.slot-count {
  font-variant-numeric: tabular-nums;
  color: var(--text-muted);
  flex-shrink: 0;
}
</style>
