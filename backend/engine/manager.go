package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/johncave/terraform-the-game/db"
	"github.com/johncave/terraform-the-game/models"
)

const idleTimeout = 5 * time.Minute

// snapshotInterval is the number of ticks between automatic state snapshots saved to the database.
const snapshotInterval = 60

type gameSession struct {
	gameID       uuid.UUID
	lastActivity time.Time
	stopCh       chan struct{}
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*gameSession
	db       *db.DB
	states   map[uuid.UUID]*models.GameState
	stateMu  sync.RWMutex
}

func NewManager(database *db.DB) *Manager {
	return &Manager{
		sessions: make(map[uuid.UUID]*gameSession),
		db:       database,
		states:   make(map[uuid.UUID]*models.GameState),
	}
}

// Start starts a game session goroutine if not already running.
func (m *Manager) Start(gameID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[gameID]; exists {
		return
	}

	session := &gameSession{
		gameID:       gameID,
		lastActivity: time.Now(),
		stopCh:       make(chan struct{}),
	}
	m.sessions[gameID] = session

	go m.runSession(session)
}

// Stop stops a game session goroutine.
func (m *Manager) Stop(gameID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[gameID]
	if !exists {
		return
	}
	close(session.stopCh)
	delete(m.sessions, gameID)
}

// Touch updates the last activity time for a game, starting it if needed.
func (m *Manager) Touch(gameID uuid.UUID) {
	m.mu.Lock()
	session, exists := m.sessions[gameID]
	if exists {
		session.lastActivity = time.Now()
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	m.Start(gameID)
}

// GetState returns the current in-memory state for a game, loading from DB if needed.
func (m *Manager) GetState(gameID uuid.UUID) (*models.GameState, error) {
	m.stateMu.RLock()
	state, exists := m.states[gameID]
	m.stateMu.RUnlock()
	if exists {
		return state, nil
	}

	if m.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	state, err := m.db.LoadState(gameID)
	if err != nil {
		return nil, err
	}
	m.stateMu.Lock()
	m.states[gameID] = state
	m.stateMu.Unlock()
	return state, nil
}

// UpdateState updates the in-memory state for a game.
func (m *Manager) UpdateState(state *models.GameState) {
	m.stateMu.Lock()
	m.states[state.GameID] = state
	m.stateMu.Unlock()
}

func (m *Manager) runSession(session *gameSession) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	ticksSinceSnapshot := 0

	for {
		select {
		case <-session.stopCh:
			return
		case <-ticker.C:
			m.mu.RLock()
			_, exists := m.sessions[session.gameID]
			lastActivity := session.lastActivity
			m.mu.RUnlock()

			if !exists {
				return
			}

			// Check idle timeout
			if time.Since(lastActivity) > idleTimeout {
				m.Stop(session.gameID)
				return
			}

			// Load and tick state
			state, err := m.GetState(session.gameID)
			if err != nil {
				continue
			}

			elapsed := time.Since(state.LastTick)
			if Tick(state, elapsed) || elapsed >= time.Second {
				m.UpdateState(state)

				if m.db != nil {
					ticksSinceSnapshot++
					if ticksSinceSnapshot >= snapshotInterval {
						_ = m.db.SaveSnapshot(state)
						ticksSinceSnapshot = 0
					}
				}
			}
		}
	}
}
