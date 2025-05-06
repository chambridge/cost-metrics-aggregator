package partitions

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/chambridge/cost-metrics-aggregator/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createPartition creates a daily partition for the metrics table for the given date
func createPartition(ctx context.Context, db *pgxpool.Pool, date time.Time) error {
	year, month, day := date.Year(), int(date.Month()), date.Day()
	partitionName := fmt.Sprintf("metrics_y%d_m%d_d%d", year, month, day)
	startDate := date
	endDate := startDate.AddDate(0, 0, 1)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF metrics
		FOR VALUES FROM ('%s') TO ('%s');
		CREATE INDEX IF NOT EXISTS %s_timestamp_idx ON %s (timestamp)
	`, partitionName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), partitionName, partitionName)

	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
	}
	log.Printf("Created partition %s for %s", partitionName, startDate.Format("2006-01-02"))
	return nil
}

// CreatePartitions creates daily partitions for the metrics table
func CreatePartitions() error {
	var init bool
	flag.BoolVar(&init, "init", false, "Initialize partitions for 90 days prior and current day")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	currentDate := now.Truncate(24 * time.Hour)
	nextDate := currentDate.AddDate(0, 0, 1)

	// Always create partitions for current and next day
	if err := createPartition(ctx, db, currentDate); err != nil {
		return err
	}
	if err := createPartition(ctx, db, nextDate); err != nil {
		return err
	}

	if init {
		// Create partitions for 90 days prior to now
		startDate := currentDate.AddDate(0, 0, -90)
		for d := startDate; !d.After(currentDate); d = d.AddDate(0, 0, 1) {
			if err := createPartition(ctx, db, d); err != nil {
				return err
			}
		}
	}

	return nil
}
