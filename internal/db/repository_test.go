package db

import (
	"context"
	"testing"
	"time"

	"github.com/chambridge/cost-metrics-aggregator/internal/db/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertNode(t *testing.T) {
	pool, newTx := testutils.SetupTestDB(t)
	tx := newTx()
	defer tx.Rollback(context.Background())

	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"

	time.Sleep(500 * time.Millisecond)
	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, nodeID)
	time.Sleep(500 * time.Millisecond)

	var count int
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM nodes WHERE id = $1", nodeID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInsertNodeMetric(t *testing.T) {
	pool, newTx := testutils.SetupTestDB(t)
	tx := newTx()
	defer tx.Rollback(context.Background())

	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"

	// Ensure node exists
	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	now := time.Now().UTC()
	year, month := now.Year(), now.Month()

	timestamp := time.Date(year, month, 15, 14, 0, 0, 0, time.UTC)
	coreCount := 4

	err = repo.InsertNodeMetric(nodeID, timestamp, coreCount, clusterID)
	assert.NoError(t, err)

	var count int
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM node_metrics WHERE node_id = $1", nodeID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected one row in node_metrics")
}

func TestUpdateNodeDailySummary(t *testing.T) {
	pool, newTx := testutils.SetupTestDB(t)
	tx := newTx()
	defer tx.Rollback(context.Background())

	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"

	// Ensure node exists
	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	require.NoError(t, err)

	timestamp, _ := time.Parse("2006-01-02 15:04:05 +0000 MST", "2025-05-17 14:00:00 +0000 UTC")
	coreCount := 4

	err = repo.UpdateNodeDailySummary(nodeID, timestamp, coreCount)
	assert.NoError(t, err)

	var totalHours int
	err = tx.QueryRow(context.Background(), "SELECT total_hours FROM node_daily_summary WHERE node_id = $1 AND date = $2", nodeID, "2025-05-17").Scan(&totalHours)
	assert.NoError(t, err)
	assert.Equal(t, 1, totalHours)
}

func TestUpsertPod(t *testing.T) {
	pool, newTx := testutils.SetupTestDB(t)
	tx := newTx()
	defer tx.Rollback(context.Background())

	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"
	podName := "zip-1"
	namespace := "test"
	component := "EAP"

	// Ensure node exists
	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	require.NoError(t, err)

	podID, err := repo.UpsertPod(clusterID, nodeID, podName, namespace, component)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, podID)

	var count int
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM pods WHERE id = $1 AND name = $2 AND namespace = $3", podID, podName, namespace).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInsertPodMetric(t *testing.T) {
	pool, newTx := testutils.SetupTestDB(t)
	tx := newTx()
	defer tx.Rollback(context.Background())

	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"
	podName := "zip-1"
	namespace := "test"
	component := "EAP"

	// Ensure node exists
	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	require.NoError(t, err)

	// Insert pod to satisfy foreign key constraint
	podID, err := repo.UpsertPod(clusterID, nodeID, podName, namespace, component)
	require.NoError(t, err)
	now := time.Now().UTC()
	year, month := now.Year(), now.Month()

	// Pick a day in this month
	timestamp := time.Date(year, month, 15, 14, 0, 0, 0, time.UTC)

	usage := 100.0
	request := 200.0
	nodeCap := 14400.0
	coreCount := 4

	err = repo.InsertPodMetric(podID, timestamp, usage, request, nodeCap, coreCount)
	assert.NoError(t, err)

	var count int
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM pod_metrics WHERE pod_id = $1 AND timestamp = $2", podID, timestamp).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected one row in pod_metrics")
}

func TestUpdatePodDailySummary(t *testing.T) {
	pool, newTx := testutils.SetupTestDB(t)
	tx := newTx()
	defer tx.Rollback(context.Background())

	repo := NewRepository(pool)

	clusterID, _ := uuid.Parse("10f5a0f9-223a-41c1-8456-9a3eb0323a99")
	nodeName := "ip-10-0-1-63.ec2.internal"
	identifier := "i-09ad6102842b9a786"
	nodeRole := "worker"
	podName := "zip-1"
	namespace := "test"
	component := "EAP"

	// Ensure node exists
	nodeID, err := repo.UpsertNode(clusterID, nodeName, identifier, nodeRole)
	require.NoError(t, err)

	// Insert pod to satisfy foreign key constraint
	podID, err := repo.UpsertPod(clusterID, nodeID, podName, namespace, component)
	require.NoError(t, err)

	timestamp, _ := time.Parse("2006-01-02 15:04:05 +0000 MST", "2025-05-17 14:00:00 +0000 UTC")
	effectiveCoreSeconds := 200.0
	coreUsage := 0.013888 // 200 / 14400

	err = repo.UpdatePodDailySummary(podID, timestamp, effectiveCoreSeconds, coreUsage)
	assert.NoError(t, err)

	var totalHours int
	var maxCoresUsed float64
	err = tx.QueryRow(context.Background(), "SELECT total_hours FROM pod_daily_summary WHERE pod_id = $1 AND date = $2", podID, "2025-05-17").Scan(&totalHours)
	assert.NoError(t, err)
	err = tx.QueryRow(context.Background(), "SELECT max_cores_used FROM pod_daily_summary WHERE pod_id = $1 AND date = $2", podID, "2025-05-17").Scan(&maxCoresUsed)
	assert.NoError(t, err)
	assert.Equal(t, 1, totalHours)
	assert.InDelta(t, 0.013888, maxCoresUsed, 0.000001)
}
