package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chambridge/cost-metrics-aggregator/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// Create daily partitions for the next 90 days
	for i := 0; i < 90; i++ {
		day := time.Now().AddDate(0, 0, i)
		tableName := fmt.Sprintf("metrics_y%d_m%d_d%d", day.Year(), day.Month(), day.Day())
		startDate := day.Format("2006-01-02")
		endDate := day.AddDate(0, 0, 1).Format("2006-01-02")

		query := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				LIKE metrics INCLUDING ALL,
				CHECK (timestamp >= '%s' AND timestamp < '%s')
			) INHERITS (metrics);
			CREATE INDEX IF NOT EXISTS %s_timestamp_idx ON %s (timestamp);
		`, tableName, startDate, endDate, tableName, tableName)

		_, err = db.Exec(context.Background(), query)
		if err != nil {
			log.Fatalf("Failed to create partition %s: %v", tableName, err)
		}
		fmt.Printf("Created partition %s\n", tableName)
	}
}
