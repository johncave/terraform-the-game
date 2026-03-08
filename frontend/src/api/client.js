const BASE_URL = import.meta.env.VITE_API_URL || '/api'

export const api = {
  async createGame() {
    const r = await fetch(`${BASE_URL}/games`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    })
    return r.json()
  },

  async getGameState(gameId) {
    const r = await fetch(`${BASE_URL}/games/${gameId}/state`)
    return r.json()
  },

  async getInventory(gameId) {
    const r = await fetch(`${BASE_URL}/games/${gameId}/inventory`)
    return r.json()
  },

  async planFactory(gameId, factoryId, yaml) {
    const r = await fetch(`${BASE_URL}/games/${gameId}/factory/${factoryId}/plan`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ yaml })
    })
    return r.json()
  },

  async applyFactory(gameId, factoryId, yaml) {
    const r = await fetch(`${BASE_URL}/games/${gameId}/factory/${factoryId}/apply`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ yaml })
    })
    return r.json()
  },

  async getFactoryState(gameId, factoryId) {
    const r = await fetch(`${BASE_URL}/games/${gameId}/factory/${factoryId}/state`)
    return r.json()
  },

  async listFactories(gameId) {
    const r = await fetch(`${BASE_URL}/games/${gameId}/factory`)
    return r.json()
  }
}
