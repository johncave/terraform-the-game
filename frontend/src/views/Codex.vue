<template>
  <div class="codex-overlay" @click.self="$emit('close')">
    <div class="codex-panel">
      <!-- Header -->
      <div class="codex-header">
        <span class="codex-title">⬡ CODEX</span>
        <span class="codex-subtitle">— documentation</span>
        <div class="spacer"></div>
        <button class="close-btn" @click="$emit('close')">✕ close</button>
      </div>

      <!-- Body: sidebar + content -->
      <div class="codex-body">
        <nav class="codex-nav">
          <button
            v-for="doc in docs"
            :key="doc.id"
            class="nav-item"
            :class="{ active: activeDoc === doc.id }"
            @click="activeDoc = doc.id"
          >
            <span class="nav-icon">{{ doc.icon }}</span>
            {{ doc.label }}
          </button>
        </nav>

        <div class="codex-content">
          <div class="markdown-body" v-html="renderedContent"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { marked } from 'marked'

import gettingStarted from '../docs/getting-started.md?raw'
import machines from '../docs/machines.md?raw'
import recipes from '../docs/recipes.md?raw'
import nodes from '../docs/nodes.md?raw'
import yamlReference from '../docs/yaml-reference.md?raw'

defineEmits(['close'])

const docs = [
  { id: 'getting-started', label: 'Getting Started', icon: '▶', content: gettingStarted },
  { id: 'machines',        label: 'Machines',        icon: '⚙', content: machines },
  { id: 'recipes',         label: 'Recipes',         icon: '⚗', content: recipes },
  { id: 'nodes',           label: 'Nodes',           icon: '◈', content: nodes },
  { id: 'yaml-reference',  label: 'YAML Reference',  icon: '📄', content: yamlReference }
]

const activeDoc = ref('getting-started')

const renderedContent = computed(() => {
  const doc = docs.find((d) => d.id === activeDoc.value)
  return doc ? marked.parse(doc.content) : ''
})
</script>

<style scoped>
.codex-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(2px);
}

.codex-panel {
  width: 90vw;
  max-width: 1100px;
  height: 85vh;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.6);
}

/* Header */
.codex-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.codex-title {
  color: var(--color-blue);
  font-weight: 700;
  font-size: 13px;
  letter-spacing: 0.05em;
}

.codex-subtitle {
  color: var(--text-muted);
  font-size: 11px;
}

.spacer { flex: 1; }

.close-btn {
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 10px;
  padding: 3px 10px;
  cursor: pointer;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  transition: all 0.15s;
}

.close-btn:hover {
  border-color: var(--color-red);
  color: var(--color-red);
}

/* Body */
.codex-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* Sidebar */
.codex-nav {
  width: 180px;
  flex-shrink: 0;
  background: var(--bg-primary);
  border-right: 1px solid var(--border-color);
  overflow-y: auto;
  padding: 8px 0;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 14px;
  background: none;
  border: none;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 12px;
  cursor: pointer;
  text-align: left;
  transition: all 0.15s;
  border-left: 2px solid transparent;
}

.nav-item:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  background: var(--bg-tertiary);
  color: var(--color-blue);
  border-left-color: var(--color-blue);
}

.nav-icon {
  font-size: 13px;
  width: 16px;
  text-align: center;
}

/* Content area */
.codex-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px 32px;
}

/* Markdown rendering */
.markdown-body {
  max-width: 760px;
  color: var(--text-primary);
  line-height: 1.7;
  font-family: var(--font-mono);
  font-size: 13px;
}

.markdown-body :deep(h1) {
  color: var(--color-blue);
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.markdown-body :deep(h2) {
  color: var(--color-cyan);
  font-size: 15px;
  font-weight: 600;
  margin: 24px 0 10px;
}

.markdown-body :deep(h3) {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
  margin: 18px 0 8px;
}

.markdown-body :deep(p) {
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.markdown-body :deep(code) {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 3px;
  padding: 1px 5px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-green);
}

.markdown-body :deep(pre) {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 14px 16px;
  overflow-x: auto;
  margin: 12px 0;
}

.markdown-body :deep(pre code) {
  background: none;
  border: none;
  padding: 0;
  color: var(--text-primary);
  font-size: 12px;
}

.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 12px 0;
  font-size: 12px;
}

.markdown-body :deep(th) {
  background: var(--bg-tertiary);
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  text-align: left;
}

.markdown-body :deep(td) {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
}

.markdown-body :deep(tr:nth-child(even) td) {
  background: rgba(255, 255, 255, 0.02);
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
  margin-bottom: 12px;
}

.markdown-body :deep(li) {
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.markdown-body :deep(a) {
  color: var(--color-blue);
  text-decoration: none;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.markdown-body :deep(blockquote) {
  border-left: 3px solid var(--color-yellow);
  padding: 6px 14px;
  margin: 12px 0;
  background: rgba(210, 153, 34, 0.05);
  color: var(--color-yellow);
  font-size: 12px;
}

.markdown-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 20px 0;
}

.markdown-body :deep(strong) {
  color: var(--text-primary);
  font-weight: 600;
}
</style>
