package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Create partitions for the next 3 months
	for i := 0; i < 3; i++ {
		month := time.Now().AddDate(0, i, 0)
		tableName := fmt.Sprintf("metrics_%s", month.Format("2006_01"))
		startDate := month.Format("2006-01-01")
		endDate := month.AddDate(0, 1, 0).Format("2006-01-01")

		query := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				LIKE metrics INCLUDING ALL,
				CHECK (timestamp >= '%s' AND timestamp < '%s')
			) INHERITS (metrics);
			CREATE INDEX IF NOT EXISTS %s_timestamp_idx ON %s (timestamp);
		`, tableName, startDate, endDate, tableName, tableName)

		_, err = conn.Exec(query)
		if err != nil {
			log.Fatalf("Failed to create partition %s: %v", tableName, err)
		}
		fmt.Printf("Created partition %s\n", tableName)
	}
}
