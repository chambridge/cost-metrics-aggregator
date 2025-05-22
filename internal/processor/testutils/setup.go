package testutils

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	dbURL := "postgres://costmetrics:costmetrics@localhost:5432/costmetrics?sslmode=disable"
	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)

	tx, err := pool.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		tx.Rollback(context.Background())
	})

	_, err = tx.Exec(context.Background(), `
		DROP TABLE IF EXISTS pod_daily_summary, pod_metrics, pods, node_daily_summary, node_metrics, nodes, clusters CASCADE
	`)
	require.NoError(t, err)

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not get caller information")
	}
	currentDir := filepath.Dir(currentFile)

	schemaPath := filepath.Join(currentDir, "..", "..", "db", "migrations", "0001_init.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(), string(schema))
	require.NoError(t, err)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	_, err = tx.Exec(context.Background(), `
		INSERT INTO clusters (id, name) VALUES ($1, 'test-cluster')
	`, clusterID)
	require.NoError(t, err)

	nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	_, err = tx.Exec(context.Background(), `
		INSERT INTO nodes (id, cluster_id, name, identifier, type)
		VALUES ($1, $2, 'ip-10-0-1-63.ec2.internal', 'i-09ad6102842b9a786', 'worker')
	`, nodeID, clusterID)
	require.NoError(t, err)

	_, err = tx.Exec(context.Background(), `
		CREATE TABLE node_metrics_202505 PARTITION OF node_metrics
		FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
		CREATE TABLE pod_metrics_202505 PARTITION OF pod_metrics
		FOR VALUES FROM ('2025-05-01') TO ('2025-06-01')
	`)
	require.NoError(t, err)

	err = tx.Commit(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}
