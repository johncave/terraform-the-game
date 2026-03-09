package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/johncave/terraform-the-game/db"
	"github.com/johncave/terraform-the-game/engine"
	"github.com/johncave/terraform-the-game/models"
	"github.com/johncave/terraform-the-game/parser"
)

type Handlers struct {
	db      *db.DB
	manager *engine.Manager
}

func NewHandlers(database *db.DB, manager *engine.Manager) *Handlers {
	return &Handlers{db: database, manager: manager}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseGameID(r *http.Request) (uuid.UUID, error) {
	vars := mux.Vars(r)
	return uuid.Parse(vars["game_id"])
}

func (h *Handlers) CreateGame(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	state, err := h.db.CreateGame(req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.manager.Start(state.GameID)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"game_id":    state.GameID,
		"created_at": time.Now(),
	})
}

func (h *Handlers) ListGames(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	ids, err := h.db.ListGames()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"games": ids})
}

func (h *Handlers) GetGame(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		if h.db == nil {
			writeError(w, http.StatusServiceUnavailable, "database not available")
			return
		}
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	h.manager.Touch(gameID)
	engine.FastForward(state)

	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) GetGameState(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		if h.db == nil {
			writeError(w, http.StatusServiceUnavailable, "database not available")
			return
		}
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	h.manager.Touch(gameID)
	engine.FastForward(state)

	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) GetInventory(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		if h.db == nil {
			writeError(w, http.StatusServiceUnavailable, "database not available")
			return
		}
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	h.manager.Touch(gameID)
	engine.FastForward(state)

	writeJSON(w, http.StatusOK, state.Inventory)
}

func (h *Handlers) ListFactories(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}

	factories, err := h.db.ListFactories(gameID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if factories == nil {
		factories = []*models.Factory{}
	}

	h.manager.Touch(gameID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"factories": factories})
}

func (h *Handlers) GetFactory(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}
	vars := mux.Vars(r)
	factoryID := vars["factory_id"]

	factory, err := h.db.GetFactory(gameID, factoryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if factory == nil {
		writeError(w, http.StatusNotFound, "factory not found")
		return
	}

	// Enrich with current machine state
	state, err := h.manager.GetState(gameID)
	if err == nil {
		engine.FastForward(state)
		for _, m := range factory.Machines {
			key := string(m.Type) + "." + m.ID
			if live, exists := state.Machines[key]; exists {
				*m = *live
			}
		}
	}

	h.manager.Touch(gameID)
	writeJSON(w, http.StatusOK, factory)
}

// PlanFactory parses and validates a factory YAML without applying it.
func (h *Handlers) PlanFactory(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}
	vars := mux.Vars(r)
	factoryID := vars["factory_id"]

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		if h.db == nil {
			writeError(w, http.StatusServiceUnavailable, "database not available")
			return
		}
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	engine.FastForward(state)

	result, err := parser.Parse(factoryID, req.YAML)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	validationErrors := parser.Validate(result, state)
	diff := parser.Diff(factoryID, result, state)

	resp := map[string]interface{}{
		"factory_id": factoryID,
		"machines":   result.Machines,
		"valid":      len(validationErrors) == 0,
		"errors":     validationErrors,
		"to_add":     diff.ToAdd,
		"to_change":  diff.ToChange,
		"to_destroy": diff.ToDestroy,
	}

	h.manager.Touch(gameID)
	writeJSON(w, http.StatusOK, resp)
}

// ApplyFactory applies a factory YAML: validates, deducts costs, registers machines.
// Machines removed from the YAML are destroyed. Machines with changed config are updated.
func (h *Handlers) ApplyFactory(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}
	vars := mux.Vars(r)
	factoryID := vars["factory_id"]

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	engine.FastForward(state)

	result, err := parser.Parse(factoryID, req.YAML)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Run plan (validation) first — apply always validates before acting.
	validationErrors := parser.Validate(result, state)
	if len(validationErrors) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"errors": validationErrors,
		})
		return
	}

	// Compute diff to know what to destroy
	diff := parser.Diff(factoryID, result, state)

	// Destroy machines that are no longer in the YAML
	for _, key := range diff.ToDestroy {
		delete(state.Machines, key)
	}

	// Apply: deduct costs and set BuiltAt on new machines; update changed machines in-place
	parser.Apply(result, state)

	// Register/update machines in game state
	for _, m := range result.Machines {
		key := string(m.Type) + "." + m.ID
		state.Machines[key] = m
	}

	// Save/update factory record
	now := time.Now()
	factory := &models.Factory{
		ID:        factoryID,
		GameID:    gameID,
		YAML:      req.YAML,
		Status:    "running",
		Machines:  result.Machines,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Check if factory already exists (ignore error — proceed with current timestamp if not found)
	existingFactory, _ := h.db.GetFactory(gameID, factoryID)
	if existingFactory != nil {
		factory.CreatedAt = existingFactory.CreatedAt
	}

	if err := h.db.SaveFactory(gameID, factory); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Persist events
	_ = h.db.AppendEvent(gameID, "InventoryUpdated", state.Inventory)
	for _, m := range result.Machines {
		_ = h.db.AppendEvent(gameID, "MachineConstructed", m)
	}

	// Save snapshot
	_ = h.db.SaveSnapshot(state)
	h.manager.UpdateState(state)
	h.manager.Touch(gameID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"factory_id": factoryID,
		"machines":   result.Machines,
		"status":     "applied",
		"success":    true,
		"to_add":     diff.ToAdd,
		"to_change":  diff.ToChange,
		"to_destroy": diff.ToDestroy,
		"inventory":  state.Inventory,
	})
}

// ResetPowerGrid clears the power trip flag so machines can resume.
func (h *Handlers) ResetPowerGrid(w http.ResponseWriter, r *http.Request) {
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		if h.db == nil {
			writeError(w, http.StatusServiceUnavailable, "database not available")
			return
		}
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	state.PowerTripped = false

	if h.db != nil {
		_ = h.db.SaveSnapshot(state)
	}
	h.manager.UpdateState(state)
	h.manager.Touch(gameID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "power_reset",
		"power_tripped": false,
	})
}

// DestroyFactory removes all machines belonging to a factory.
func (h *Handlers) DestroyFactory(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	gameID, err := parseGameID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid game_id")
		return
	}
	vars := mux.Vars(r)
	factoryID := vars["factory_id"]

	factory, err := h.db.GetFactory(gameID, factoryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if factory == nil {
		writeError(w, http.StatusNotFound, "factory not found")
		return
	}

	state, err := h.manager.GetState(gameID)
	if err != nil {
		state, err = h.db.GetGame(gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "game not found")
			return
		}
	}

	// Remove machines from game state
	for _, m := range factory.Machines {
		key := string(m.Type) + "." + m.ID
		delete(state.Machines, key)
	}

	if err := h.db.DeleteFactory(gameID, factoryID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.db.SaveSnapshot(state)
	h.manager.UpdateState(state)
	h.manager.Touch(gameID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
}
