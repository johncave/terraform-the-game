package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/johncave/terraform-the-game/api"
	"github.com/johncave/terraform-the-game/db"
	"github.com/johncave/terraform-the-game/engine"
	"github.com/rs/cors"
)

func main() {
	database, err := db.New()
	if err != nil {
		log.Printf("Warning: could not connect to database: %v", err)
		log.Printf("Starting without database connection (some features will be unavailable)")
	} else {
		if err := database.RunMigrations(); err != nil {
			log.Printf("Warning: migration error: %v", err)
		}
	}

	manager := engine.NewManager(database)

	router := api.NewRouter(database, manager)

	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigins},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting Terraform: The Game backend on %s", addr)

	handler := c.Handler(router)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
