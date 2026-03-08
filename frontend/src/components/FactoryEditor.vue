<template>
  <div class="editor-wrap">
    <!-- Header bar -->
    <div class="editor-header">
      <span class="panel-title">FACTORY EDITOR</span>
      <div class="factory-name-wrap">
        <span class="name-prefix">factory:</span>
        <input
          v-model="factoryId"
          type="text"
          class="factory-name-input"
          placeholder="main"
        />
      </div>
      <div class="spacer"></div>
      <button
        class="btn btn-primary"
        :disabled="planLoading || applyLoading"
        @click="handlePlan"
      >
        <span v-if="planLoading" class="spinner">◌</span>
        <span v-else>▷</span>
        Plan
      </button>
      <button
        class="btn btn-success"
        :disabled="planLoading || applyLoading || lastPlanValid === false"
        @click="handleApply"
      >
        <span v-if="applyLoading" class="spinner">◌</span>
        <span v-else>✓</span>
        Apply
      </button>
    </div>

    <!-- Plan status bar -->
    <div v-if="planStatus" class="plan-status" :class="planStatus.class">
      <span class="status-icon">{{ planStatus.icon }}</span>
      {{ planStatus.message }}
    </div>

    <!-- Monaco Editor container -->
    <div ref="editorContainer" class="editor-container"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import loader from '@monaco-editor/loader'
import { useGameStore } from '../stores/game.js'

const store = useGameStore()

const editorContainer = ref(null)
const factoryId = ref(store.activeFactoryId || 'main')
const planLoading = ref(false)
const applyLoading = ref(false)
const lastPlanValid = ref(null)
const planStatus = ref(null)

let monacoEditor = null

const GENESIS_YAML = `# Terraform: The Game - Genesis Configuration
# This is your first factory. Modify and apply it!
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
        - target: "inventory"
`

onMounted(async () => {
  loader.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' } })

  const monaco = await loader.init()

  monacoEditor = monaco.editor.create(editorContainer.value, {
    value: GENESIS_YAML,
    language: 'yaml',
    theme: 'vs-dark',
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    lineNumbers: 'on',
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    automaticLayout: true,
    renderWhitespace: 'boundary',
    smoothScrolling: true,
    cursorBlinking: 'smooth',
    padding: { top: 8, bottom: 8 },
    scrollbar: {
      verticalScrollbarSize: 6,
      horizontalScrollbarSize: 6
    }
  })
})

onUnmounted(() => {
  if (monacoEditor) {
    monacoEditor.dispose()
    monacoEditor = null
  }
})

watch(
  () => store.activeFactoryId,
  (id) => {
    factoryId.value = id
  }
)

function getEditorValue() {
  return monacoEditor ? monacoEditor.getValue() : ''
}

async function handlePlan() {
  planLoading.value = true
  planStatus.value = null
  lastPlanValid.value = null
  try {
    const yaml = getEditorValue()
    const result = await store.plan(factoryId.value, yaml)
    lastPlanValid.value = result.valid
    planStatus.value = result.valid
      ? { class: 'status-ok', icon: '✓', message: 'Plan succeeded — ready to apply' }
      : { class: 'status-err', icon: '✗', message: 'Plan failed — check terminal for errors' }
  } finally {
    planLoading.value = false
  }
}

async function handleApply() {
  applyLoading.value = true
  planStatus.value = null
  try {
    const yaml = getEditorValue()
    const result = await store.apply(factoryId.value, yaml)
    planStatus.value = result.success
      ? { class: 'status-ok', icon: '✓', message: 'Apply complete! Factory updated.' }
      : { class: 'status-err', icon: '✗', message: result.message || 'Apply failed' }
    if (result.success) lastPlanValid.value = null
  } finally {
    applyLoading.value = false
  }
}
</script>

<style scoped>
.editor-wrap {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

/* Header */
.editor-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  flex-wrap: wrap;
}

.panel-title {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-blue);
  white-space: nowrap;
}

.factory-name-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
}

.name-prefix {
  color: var(--text-muted);
  white-space: nowrap;
}

.factory-name-input {
  width: 120px;
  padding: 3px 6px;
  font-size: 11px;
  height: 24px;
}

.spacer { flex: 1; }

/* Buttons */
.btn {
  height: 26px;
  padding: 0 12px;
  font-size: 11px;
}

/* Plan status bar */
.plan-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  font-size: 11px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-color);
}

.plan-status.status-ok {
  background: rgba(63, 185, 80, 0.08);
  color: var(--color-green);
}

.plan-status.status-err {
  background: rgba(248, 81, 73, 0.08);
  color: var(--color-red);
}

.status-icon { font-size: 12px; }

/* Monaco editor container */
.editor-container {
  flex: 1;
  overflow: hidden;
}

/* Spinner */
.spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
