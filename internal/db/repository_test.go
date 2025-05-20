package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Connect to PostgreSQL from podman-compose
	dbURL := "postgres://costmetrics:costmetrics@localhost:5432/costmetrics?sslmode=disable"
	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)

	// Start a transaction
	tx, err := pool.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		tx.Rollback(context.Background())
	})

	// Drop existing tables
	_, err = tx.Exec(context.Background(), `
		DROP TABLE IF EXISTS pod_daily_summary, pod_metrics, pods, node_daily_summary, node_metrics, nodes, clusters CASCADE
	`)
	require.NoError(t, err)

	// Apply schema
	schemaPath := filepath.Join("..", "..", "internal", "db", "migrations", "0001_init.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(), string(schema))
	require.NoError(t, err)

	// Insert a cluster
	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	_, err = tx.Exec(context.Background(), `
		INSERT INTO clusters (id, name) VALUES ($1, 'test-cluster')
	`, clusterID)
	require.NoError(t, err)

	// Insert a node for tests requiring node_id
	nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	_, err = tx.Exec(context.Background(), `
		INSERT INTO nodes (id, cluster_id, name, identifier, type)
		VALUES ($1, $2, 'ip-10-0-1-63.ec2.internal', 'i-09ad6102842b9a786', 'worker')
	`, nodeID, clusterID)
	require.NoError(t, err)

	// Insert a pod for tests requiring pod_id
	podID, _ := uuid.Parse("d6e9ea32-4b00-4262-a297-a79da36a590a")
	_, err = tx.Exec(context.Background(), `
		INSERT INTO pods (id, cluster_id, node_id, name, namespace, component)
		VALUES ($1, $2, $3, 'zip-1', 'test', 'EAP')
	`, podID, clusterID, nodeID)
	require.NoError(t, err)

	// Create partitions for May 2025
	_, err = tx.Exec(context.Background(), `
		CREATE TABLE node_metrics_202505 PARTITION OF node_metrics
		FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
		CREATE TABLE pod_metrics_202505 PARTITION OF pod_metrics
		FOR VALUES FROM ('2025-05-01') TO ('2025-06-01')
	`)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestUpsertNode(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"

	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, nodeID)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM nodes WHERE id = $1 AND name = $2 AND type = $3", nodeID, nodeName, nodeRole).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInsertNodeMetric(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	timestamp, _ := time.Parse("2006-01-02 15:04:05 +0000 MST", "2025-05-17 14:00:00 +0000 UTC")
	coreCount := 4

	err := repo.InsertNodeMetric(nodeID, timestamp, coreCount, clusterID)
	assert.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM node_metrics WHERE node_id = $1 AND timestamp = $2", nodeID, timestamp).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUpdateNodeDailySummary(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	timestamp, _ := time.Parse("2006-01-02 15:04:05 +0000 MST", "2025-05-17 14:00:00 +0000 UTC")
	coreCount := 4

	err := repo.UpdateNodeDailySummary(nodeID, timestamp, coreCount)
	assert.NoError(t, err)

	var totalHours int
	err = pool.QueryRow(context.Background(), "SELECT total_hours FROM node_daily_summary WHERE node_id = $1 AND date = $2", nodeID, "2025-05-17").Scan(&totalHours)
	assert.NoError(t, err)
	assert.Equal(t, 1, totalHours)

	// Update again for same date
	err = repo.UpdateNodeDailySummary(nodeID, timestamp, coreCount)
	assert.NoError(t, err)
	err = pool.QueryRow(context.Background(), "SELECT total_hours FROM node_daily_summary WHERE node_id = $1 AND date = $2", nodeID, "2025-05-17").Scan(&totalHours)
	assert.NoError(t, err)
	assert.Equal(t, 1, totalHours) // Adjust based on ON CONFLICT logic
}

func TestUpsertPod(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	podName := "zip-1"
	namespace := "test"
	component := "EAP"

	podID, err := repo.UpsertPod(clusterID, nodeID, podName, namespace, component)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, podID)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM pods WHERE id = $1 AND name = $2 AND namespace = $3", podID, podName, namespace).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInsertPodMetric(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	podID, _ := uuid.Parse("d6e9ea32-4b00-4262-a297-a79da36a590a")
	timestamp, _ := time.Parse("2006-01-02 15:04:05 +0000 MST", "2025-05-17 14:00:00 +0000 UTC")
	usage := 100.0
	request := 200.0
	nodeCap := 14400.0
	coreCount := 4

	err := repo.InsertPodMetric(podID, timestamp, usage, request, nodeCap, coreCount)
	assert.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM pod_metrics WHERE pod_id = $1 AND timestamp = $2", podID, timestamp).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUpdatePodDailySummary(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewRepository(pool)

	podID, _ := uuid.Parse("d6e9ea32-4b00-4262-a297-a79da36a590a")
	timestamp, _ := time.Parse("2006-01-02 15:04:05 +0000 MST", "2025-05-17 14:00:00 +0000 UTC")
	effectiveCoreSeconds := 200.0
	coreUsage := 0.013888 // 200 / 14400

	err := repo.UpdatePodDailySummary(podID, timestamp, effectiveCoreSeconds, coreUsage)
	assert.NoError(t, err)

	var totalHours int
	var maxCoresUsed float64
	err = pool.QueryRow(context.Background(), "SELECT total_hours FROM pod_daily_summary WHERE pod_id = $1 AND date = $2", podID, "2025-05-17").Scan(&totalHours)
	assert.NoError(t, err)
	err = pool.QueryRow(context.Background(), "SELECT max_cores_used FROM pod_daily_summary WHERE pod_id = $1 AND date = $2", podID, "2025-05-17").Scan(&maxCoresUsed)
	assert.NoError(t, err)
	assert.Equal(t, 1, totalHours)
	assert.InDelta(t, 0.013888, maxCoresUsed, 0.000001)
}
