package main

import (
	"log"
	"github.com/chambridge/cost-metrics-aggregator/api"
	"github.com/chambridge/cost-metrics-aggregator/internal/config"
	"github.com/chambridge/cost-metrics-aggregator/internal/db"
)

function main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbConn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	router := api.SetupRouter(dbConn, cfg)
	log.Fatal(router.Run(cfg.ServerAddress))
}
