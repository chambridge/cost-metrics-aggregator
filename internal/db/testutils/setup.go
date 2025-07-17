package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestDB creates a database connection pool and sets up the schema for testing.
// It returns the pool and a function to start a new transaction for each test case.
// The pool is closed only after all tests in the suite complete.
func SetupTestDB(t *testing.T) (*pgxpool.Pool, func() pgx.Tx) {
	dbURL := "postgres://costmetrics:costmetrics@localhost:5432/costmetrics?sslmode=disable"
	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)

	// Use a mutex to ensure pool is closed only once
	var mu sync.Mutex
	closed := false

	// Close pool after all tests, ensuring it's called only once
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		if !closed {
			pool.Close()
			closed = true
		}
	})

	// Drop and recreate schema in a transaction
	tx, err := pool.Begin(context.Background())
	require.NoError(t, err)

	_, err = tx.Exec(context.Background(), `
		DROP TABLE IF EXISTS pod_daily_summary, pod_metrics, pods, 
		node_daily_summary, node_metrics, nodes, clusters CASCADE
	`)
	require.NoError(t, err)

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not get caller information")
	}
	currentDir := filepath.Dir(currentFile)

	schemaPath := filepath.Join(currentDir, "..", "migrations", "0001_init.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(), string(schema))
	require.NoError(t, err)

	// Insert test data
	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	_, err = tx.Exec(context.Background(), `
		INSERT INTO clusters (id, name) VALUES ($1, 'test-cluster')
	`, clusterID)
	require.NoError(t, err)

	// nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	// _, err = tx.Exec(context.Background(), `
	// 	INSERT INTO nodes (id, cluster_id, name, identifier, type)
	// 	VALUES ($1, $2, 'ip-10-0-1-63.ec2.internal', 'i-09ad6102842b9a786', 'worker')
	// `, nodeID, clusterID)
	// require.NoError(t, err)
	now := time.Now().UTC()
	year, month := now.Year(), int(now.Month())

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0) // First day of next month

	partitionNameNode := fmt.Sprintf("node_metrics_%d%02d", year, int(month))
	partitionNamePod := fmt.Sprintf("pod_metrics_%d%02d", year, int(month))

	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s PARTITION OF node_metrics
		FOR VALUES FROM ('%s') TO ('%s');
	CREATE TABLE IF NOT EXISTS %s PARTITION OF pod_metrics
		FOR VALUES FROM ('%s') TO ('%s');`,
		partitionNameNode, start.Format("2006-01-02"), end.Format("2006-01-02"),
		partitionNamePod, start.Format("2006-01-02"), end.Format("2006-01-02"))

	_, err = tx.Exec(context.Background(), sql)
	require.NoError(t, err)

	// Commit initial setup
	err = tx.Commit(context.Background())
	require.NoError(t, err)

	// Truncate tables before each test
	t.Cleanup(func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("Failed to begin cleanup transaction: %v", err)
		}
		_, err = tx.Exec(context.Background(), `
			TRUNCATE TABLE pod_daily_summary, pod_metrics, pods, 
			node_daily_summary, node_metrics, nodes, clusters CASCADE
		`)
		if err != nil {
			t.Fatalf("Failed to truncate tables: %v", err)
		}
		err = tx.Commit(context.Background())
		if err != nil {
			t.Fatalf("Failed to commit cleanup transaction: %v", err)
		}
	})

	// Return a function to start a new transaction
	newTx := func() pgx.Tx {
		tx, err := pool.Begin(context.Background())
		require.NoError(t, err)
		return tx
	}

	return pool, newTx
}
