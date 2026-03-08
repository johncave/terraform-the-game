import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../api/client.js'

export const useGameStore = defineStore('game', () => {
  const gameId = ref(localStorage.getItem('gameId') || null)
  const gameState = ref(null)
  const factories = ref([])
  const activeFactoryId = ref('main')
  const terminalLines = ref([])
  const polling = ref(false)
  let pollInterval = null

  function log(line, type = 'info') {
    const timestamp = new Date().toLocaleTimeString()
    terminalLines.value.push({ timestamp, line, type })
    if (terminalLines.value.length > 200) terminalLines.value.shift()
  }

  async function createGame() {
    const data = await api.createGame()
    gameId.value = data.game_id
    localStorage.setItem('gameId', data.game_id)
    log(`✓ Game created: ${data.game_id}`, 'success')
    await refreshState()
    startPolling()
    return data.game_id
  }

  async function loadGame(id) {
    gameId.value = id
    localStorage.setItem('gameId', id)
    await refreshState()
    startPolling()
    log(`✓ Game loaded: ${id}`, 'success')
  }

  async function refreshState() {
    if (!gameId.value) return
    try {
      gameState.value = await api.getGameState(gameId.value)
      factories.value = (await api.listFactories(gameId.value)) || []
    } catch (e) {
      log(`Error fetching state: ${e.message}`, 'error')
    }
  }

  function startPolling() {
    if (pollInterval) clearInterval(pollInterval)
    polling.value = true
    pollInterval = setInterval(refreshState, 1000)
  }

  function stopPolling() {
    if (pollInterval) clearInterval(pollInterval)
    polling.value = false
  }

  async function plan(factoryId, yaml) {
    log(`\n$ terraform plan (factory: ${factoryId})`, 'command')
    try {
      const result = await api.planFactory(gameId.value, factoryId, yaml)
      if (result.plan_output) {
        result.plan_output.split('\n').forEach((l) => log(l, result.valid ? 'info' : 'error'))
      }
      if (result.errors && result.errors.length > 0) {
        result.errors.forEach((e) => log(`  ✗ ${e}`, 'error'))
      }
      if (result.valid) {
        log('Plan: 1 to add, 0 to change, 0 to destroy.', 'success')
      }
      return result
    } catch (e) {
      log(`Error: ${e.message}`, 'error')
      return { valid: false }
    }
  }

  async function apply(factoryId, yaml) {
    log(`\n$ terraform apply (factory: ${factoryId})`, 'command')
    try {
      const result = await api.applyFactory(gameId.value, factoryId, yaml)
      log(result.message || 'Apply complete.', result.success ? 'success' : 'error')
      if (result.success) {
        log('Apply complete! Resources: 1 added.', 'success')
        await refreshState()
      }
      return result
    } catch (e) {
      log(`Error: ${e.message}`, 'error')
      return { success: false }
    }
  }

  const machines = computed(() => {
    if (!gameState.value) return {}
    return gameState.value.machines || {}
  })

  const inventory = computed(() => {
    if (!gameState.value) return {}
    return gameState.value.inventory?.items || {}
  })

  const discoveredNodes = computed(() => {
    if (!gameState.value) return []
    return gameState.value.discovered_nodes || []
  })

  return {
    gameId,
    gameState,
    factories,
    activeFactoryId,
    terminalLines,
    polling,
    createGame,
    loadGame,
    refreshState,
    startPolling,
    stopPolling,
    plan,
    apply,
    log,
    machines,
    inventory,
    discoveredNodes
  }
})
