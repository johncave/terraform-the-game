package api

import (
	"github.com/gorilla/mux"
	"github.com/johncave/terraform-the-game/db"
	"github.com/johncave/terraform-the-game/engine"
)

func NewRouter(database *db.DB, manager *engine.Manager) *mux.Router {
	h := NewHandlers(database, manager)
	r := mux.NewRouter()

	r.HandleFunc("/games", h.CreateGame).Methods("POST")
	r.HandleFunc("/games", h.ListGames).Methods("GET")
	r.HandleFunc("/games/{game_id}", h.GetGame).Methods("GET")

	r.HandleFunc("/games/{game_id}/factory", h.ListFactories).Methods("GET")
	r.HandleFunc("/games/{game_id}/factory/{factory_id}", h.GetFactory).Methods("GET")
	r.HandleFunc("/games/{game_id}/factory/{factory_id}/plan", h.PlanFactory).Methods("POST")
	r.HandleFunc("/games/{game_id}/factory/{factory_id}/apply", h.ApplyFactory).Methods("POST")
	r.HandleFunc("/games/{game_id}/factory/{factory_id}/destroy", h.DestroyFactory).Methods("POST")

	r.HandleFunc("/games/{game_id}/state", h.GetGameState).Methods("GET")
	r.HandleFunc("/games/{game_id}/inventory", h.GetInventory).Methods("GET")
	r.HandleFunc("/games/{game_id}/power/reset", h.ResetPowerGrid).Methods("POST")

	return r
}
