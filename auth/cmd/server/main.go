package main

import (
	"log"
	"os"

	"github.com/status-im/proxy-common/auth/config"
	"github.com/status-im/proxy-common/auth/server"
)

func main() {
	// Try to load from environment first
	cfg, err := config.LoadFromEnv()
	if err != nil {
		cfg, err = config.Load()
		if err != nil {
			log.Fatal("Failed to load config:", err)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	srv, err := server.New(
		server.WithConfig(cfg),
		server.WithMetrics(true),
		server.WithTestMode(true), // Enable test endpoints for development
	)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	if err := srv.ListenAndServe(":" + port); err != nil {
		log.Fatal("Server error:", err)
	}
}
