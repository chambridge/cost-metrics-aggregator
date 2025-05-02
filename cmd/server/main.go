package main

import (
	"context"
	"github.com/chambridge/cost-metrics-aggregator/api"
	"github.com/chambridge/cost-metrics-aggregator/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	router := api.SetupRouter(dbpool, cfg)
	log.Fatal(router.Run(cfg.ServerAddress))
}
