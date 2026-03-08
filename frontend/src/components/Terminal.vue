<template>
  <div class="terminal-wrap">
    <div class="panel-header">
      <span class="panel-title">TERMINAL</span>
      <span class="sep">—</span>
      <span>{{ store.terminalLines.length }} lines</span>
      <div class="spacer"></div>
      <button class="btn-icon" title="Clear terminal" @click="store.terminalLines = []">✕ clear</button>
    </div>
    <div ref="terminalBody" class="terminal-body">
      <div
        v-for="(entry, i) in store.terminalLines"
        :key="i"
        class="terminal-line"
        :class="`type-${entry.type}`"
      >
        <span class="ts">{{ entry.timestamp }}</span>
        <span class="line-text">{{ entry.line }}</span>
      </div>

      <!-- Empty state -->
      <div v-if="store.terminalLines.length === 0" class="empty-state">
        <span class="prompt">$</span>
        <span class="cursor-blink">█</span>
        <span class="hint"> waiting for output...</span>
      </div>
    </div>

    <!-- Prompt bar -->
    <div class="prompt-bar">
      <span class="prompt-sym">▸</span>
      <span class="prompt-path">terraform-the-game</span>
      <span class="prompt-sep">$</span>
      <span class="cursor-blink">█</span>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useGameStore } from '../stores/game.js'

const store = useGameStore()
const terminalBody = ref(null)

watch(
  () => store.terminalLines.length,
  async () => {
    await nextTick()
    if (terminalBody.value) {
      terminalBody.value.scrollTop = terminalBody.value.scrollHeight
    }
  }
)
</script>

<style scoped>
.terminal-wrap {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: #0a0e13;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: #111820;
  border-bottom: 1px solid var(--border-color);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  flex-shrink: 0;
}

.panel-title { color: var(--color-green); }
.sep { opacity: 0.3; }
.spacer { flex: 1; }

.btn-icon {
  background: none;
  border: none;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 10px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  transition: all 0.15s;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.btn-icon:hover {
  background: var(--bg-tertiary);
  color: var(--color-red);
}

/* Terminal body */
.terminal-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  font-size: 12px;
  line-height: 1.7;
}

.terminal-line {
  display: flex;
  gap: 10px;
  font-family: var(--font-mono);
  white-space: pre-wrap;
  word-break: break-all;
}

.ts {
  color: var(--text-muted);
  font-size: 10px;
  flex-shrink: 0;
  padding-top: 2px;
  min-width: 72px;
}

.line-text {
  flex: 1;
}

/* Color coding by type */
.type-command .line-text { color: var(--color-cyan); font-weight: 600; }
.type-success .line-text { color: var(--color-green); }
.type-error .line-text { color: var(--color-red); }
.type-info .line-text { color: var(--text-secondary); }
.type-warn .line-text { color: var(--color-yellow); }

/* Empty state */
.empty-state {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-muted);
  font-size: 12px;
  padding: 4px 0;
}

.prompt { color: var(--color-green); }
.hint { color: var(--text-muted); }

/* Prompt bar */
.prompt-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #111820;
  border-top: 1px solid var(--border-color);
  font-size: 12px;
  flex-shrink: 0;
}

.prompt-sym { color: var(--color-green); }
.prompt-path { color: var(--color-blue); }
.prompt-sep { color: var(--color-yellow); }

/* Blinking cursor */
.cursor-blink {
  color: var(--color-green);
  animation: blink 1.1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}
</style>
