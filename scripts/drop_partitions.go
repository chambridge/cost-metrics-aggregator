package main

import (
	"context"
	"fmt"
	"github.com/chambridge/cost-metrics-aggregator/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get the previous month
	lastMonth := time.Now().AddDate(0, -1, 0)
	year, month := lastMonth.Year(), lastMonth.Month()
	// Calculate the number of days in the previous month
	daysInMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()

	for day := 1; day <= daysInMonth; day++ {
		partitionName := fmt.Sprintf("metrics_y%d_m%d_d%d", year, month, day)
		_, err := db.Exec(context.Background(), `DROP TABLE IF EXISTS %s`, partitionName)
		if err != nil {
			log.Printf("Failed to drop partition %s: %v", partitionName, err)
			continue
		}
		log.Printf("Dropped partition %s", partitionName)
	}

	log.Printf("Successfully dropped %d partitions for %d-%02d", daysInMonth, year, month)
}
