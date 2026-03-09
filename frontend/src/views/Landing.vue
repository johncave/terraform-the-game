<template>
  <div class="landing">
    <!-- Scanline overlay -->
    <div class="scanlines"></div>

    <div class="landing-inner">
      <!-- ASCII Art Title -->
      <pre class="ascii-title">{{ asciiArt }}</pre>

      <p class="tagline">
        <span class="tag-bracket">[</span>
        Infrastructure as Code · Factory Simulation · Multi-tenant
        <span class="tag-bracket">]</span>
      </p>

      <!-- Connection Status -->
      <div class="conn-status" :class="connStatus.class">
        <span class="status-dot" :class="connStatus.dotClass"></span>
        {{ connStatus.label }}
      </div>

      <!-- Actions -->
      <div class="actions">
        <!-- New Game -->
        <div class="action-card">
          <div class="action-card-title">
            <span class="icon">▶</span> New Game
          </div>
          <p class="action-card-desc">Provision a fresh terraform workspace and begin your planetary conquest.</p>
          <button class="btn btn-primary btn-lg" :disabled="creating" @click="handleCreate">
            <span v-if="creating" class="spinner">◌</span>
            <span v-else>⬡</span>
            {{ creating ? 'Provisioning...' : '$ terraform init' }}
          </button>
        </div>

        <!-- Load Game -->
        <div class="action-card">
          <div class="action-card-title">
            <span class="icon">↺</span> Load Game
          </div>
          <p class="action-card-desc">Resume from an existing game ID. Restore your infrastructure state.</p>
          <div class="load-row">
            <input
              v-model="loadId"
              type="text"
              placeholder="Enter game UUID..."
              class="load-input"
              @keydown.enter="handleLoad"
            />
            <button class="btn btn-success" :disabled="!loadId.trim() || loading" @click="handleLoad">
              {{ loading ? '...' : '$ load' }}
            </button>
          </div>
          <div v-if="savedGameId" class="saved-id">
            <span class="label">Saved:</span>
            <span class="id" @click="loadId = savedGameId">{{ savedGameId }}</span>
          </div>
        </div>
      </div>

      <!-- Error message -->
      <div v-if="errorMsg" class="error-msg">
        <span class="icon">✗</span> {{ errorMsg }}
      </div>

      <!-- Footer -->
      <div class="footer">
        <span class="version">v0.1.0</span>
        <span class="sep">·</span>
        <span>terraform-the-game</span>
        <span class="sep">·</span>
        <span>backend: localhost:8080</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useGameStore } from '../stores/game.js'

const router = useRouter()
const store = useGameStore()

const creating = ref(false)
const loading = ref(false)
const loadId = ref('')
const errorMsg = ref('')
const backendReachable = ref(null)
const savedGameId = computed(() => localStorage.getItem('gameId'))

const asciiArt = `
 ████████╗███████╗██████╗ ██████╗  █████╗ ███████╗ ██████╗ ██████╗ ███╗   ███╗
    ██╔══╝██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔═══██╗██╔══██╗████╗ ████║
    ██║   █████╗  ██████╔╝██████╔╝███████║█████╗  ██║   ██║██████╔╝██╔████╔██║
    ██║   ██╔══╝  ██╔══██╗██╔══██╗██╔══██║██╔══╝  ██║   ██║██╔══██╗██║╚██╔╝██║
    ██║   ███████╗██║  ██║██║  ██║██║  ██║██║     ╚██████╔╝██║  ██║██║ ╚═╝ ██║
    ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝      ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝
                          ── THE  GAME ──`

const connStatus = computed(() => {
  if (backendReachable.value === null) return { class: 'checking', dotClass: 'IDLE', label: 'Checking backend...' }
  if (backendReachable.value) return { class: 'online', dotClass: 'GREEN', label: 'Backend online · localhost:8080' }
  return { class: 'offline', dotClass: 'RED', label: 'Backend offline · start the server' }
})

onMounted(async () => {
  try {
    const r = await fetch('/api/games', { method: 'HEAD' }).catch(() => null)
    backendReachable.value = r !== null
  } catch {
    backendReachable.value = false
  }
})

async function handleCreate() {
  errorMsg.value = ''
  creating.value = true
  try {
    const id = await store.createGame()
    router.push(`/game/${id}`)
  } catch (e) {
    errorMsg.value = `Failed to create game: ${e.message}`
  } finally {
    creating.value = false
  }
}

async function handleLoad() {
  const id = loadId.value.trim()
  if (!id) return
  errorMsg.value = ''
  loading.value = true
  try {
    await store.loadGame(id)
    router.push(`/game/${id}`)
  } catch (e) {
    errorMsg.value = `Failed to load game: ${e.message}`
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.landing {
  position: relative;
  min-height: 100vh;
  width: 100%;
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
}

/* Scanline effect */
.scanlines {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background: repeating-linear-gradient(
    0deg,
    transparent,
    transparent 2px,
    rgba(0, 0, 0, 0.03) 2px,
    rgba(0, 0, 0, 0.03) 4px
  );
}

.landing-inner {
  position: relative;
  z-index: 1;
  max-width: 900px;
  width: 100%;
  padding: 40px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
}

.ascii-title {
  font-family: var(--font-mono);
  font-size: clamp(5px, 1.1vw, 11px);
  line-height: 1.2;
  color: var(--color-blue);
  text-shadow: 0 0 20px rgba(88, 166, 255, 0.4);
  text-align: center;
  white-space: pre;
  overflow: hidden;
}

.tagline {
  color: var(--text-secondary);
  font-size: 12px;
  letter-spacing: 0.05em;
  text-align: center;
}

.tag-bracket {
  color: var(--color-blue);
  opacity: 0.6;
}

/* Connection status */
.conn-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: var(--radius-md);
  font-size: 11px;
  border: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.conn-status.online { border-color: rgba(63, 185, 80, 0.3); color: var(--color-green); }
.conn-status.offline { border-color: rgba(248, 81, 73, 0.3); color: var(--color-red); }
.conn-status.checking { color: var(--text-muted); }

/* Actions */
.actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  width: 100%;
}

@media (max-width: 600px) {
  .actions { grid-template-columns: 1fr; }
}

.action-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: border-color 0.2s;
}

.action-card:hover {
  border-color: rgba(88, 166, 255, 0.3);
}

.action-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-card-title .icon {
  color: var(--color-blue);
}

.action-card-desc {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.load-row {
  display: flex;
  gap: 8px;
}

.load-input {
  flex: 1;
  min-width: 0;
}

.saved-id {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--text-muted);
}

.saved-id .id {
  color: var(--color-blue);
  cursor: pointer;
  text-decoration: underline;
  font-size: 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}

.saved-id .id:hover {
  color: var(--text-primary);
}

/* Error */
.error-msg {
  color: var(--color-red);
  font-size: 12px;
  background: rgba(248, 81, 73, 0.1);
  border: 1px solid rgba(248, 81, 73, 0.2);
  border-radius: var(--radius-sm);
  padding: 8px 14px;
  width: 100%;
}

/* Footer */
.footer {
  font-size: 10px;
  color: var(--text-muted);
  display: flex;
  gap: 8px;
  align-items: center;
}

.footer .sep { opacity: 0.4; }
.footer .version { color: var(--color-purple); }

/* Spinner animation */
.spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
