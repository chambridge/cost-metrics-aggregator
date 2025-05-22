package processor

import (
	"context"
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/chambridge/cost-metrics-aggregator/internal/processor/testutils"
	// "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
)

// func TestProcessCSV(t *testing.T) {
// 	pool := testutils.SetupTestDB(t)
// 	repo := db.NewRepository(pool)
// 	ctx := context.Background()

// 	os.Setenv("POD_LABEL_KEYS", "label_rht_comp")
// 	defer os.Unsetenv("POD_LABEL_KEYS")

// 	clusterID := "10f5a0f9-223a-41c1-8456-9a3eb0323a99"
// 	clusterUUID, _ := uuid.Parse(clusterID)
// 	nodeName := "ip-10-0-1-63.ec2.internal"
// 	// identifier := "i-09ad6102842b9a786"
// 	// nodeRole := "worker"
// 	// podName := "zip-1"
// 	// namespace := "test"
// 	// component := "EAP"

// 	// Setup cluster
// 	err := repo.UpsertCluster(clusterUUID, "test-cluster")
// 	require.NoError(t, err)

// 	csvData := `report_period_start,report_period_end,interval_start,interval_end,node,namespace,pod,pod_usage_cpu_core_seconds,pod_request_cpu_core_seconds,pod_limit_cpu_core_seconds,pod_usage_memory_byte_seconds,pod_request_memory_byte_seconds,pod_limit_memory_byte_seconds,node_capacity_cpu_cores,node_capacity_cpu_core_seconds,node_capacity_memory_bytes,node_capacity_memory_byte_seconds,node_role,resource_id,pod_labels
// 2025-05-17 00:00:00 +0000 UTC,2025-05-17 23:59:59 +0000 UTC,2025-05-17 14:00:00 +0000 UTC,2025-05-17 15:00:00 +0000 UTC,ip-10-0-1-63.ec2.internal,test,zip-1,150,250,350,1500,2500,3500,4,14400,17179869184,61729433600,worker,i-09ad6102842b9a786,app:web|label_rht_comp:EAP
// 2025-05-17 00:00:00 +0000 UTC,2025-05-17 23:59:59 +0000 UTC,2025-05-17 15:00:00 +0000 UTC,2025-05-17 16:00:00 +0000 UTC,ip-10-0-1-63.ec2.internal,test,zip-1,200,300,400,2000,3000,4000,4,14400,17179869184,61729433600,worker,i-09ad6102842b9a786,app:web|label_rht_comp:EAP`

// 	reader := csv.NewReader(strings.NewReader(csvData))
// 	err = ProcessCSV(ctx, repo, reader, clusterID)
// 	assert.NoError(t, err)

// 	var nodeID string
// 	err = pool.QueryRow(context.Background(), "SELECT id FROM nodes WHERE name = $1 AND cluster_id = $2 ", nodeName, clusterID).Scan(&nodeID)
// 	nodeUUID, _ := uuid.Parse(nodeID)

// 	// Check node_daily_summary
// 	var totalHours int
// 	err = pool.QueryRow(context.Background(), "SELECT total_hours FROM node_daily_summary WHERE node_id = $1 AND date = $2 AND core_count = $3", nodeUUID, "2025-05-17", 4).Scan(&totalHours)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, totalHours, "node_daily_summary should have 2 hours (14:00, 15:00)")

// 	// // Check pod_metrics (aggregate usage for 14:00)
// 	// var podUsage float64
// 	// err = pool.QueryRow(context.Background(), "SELECT pod_usage_cpu_core_seconds FROM pod_metrics WHERE pod_id = $1 AND timestamp = $2", podID, "2025-05-17 14:00:00+00").Scan(&podUsage)
// 	// assert.NoError(t, err)
// 	// assert.Equal(t, 250.0, podUsage, "pod_metrics should sum usage for 14:00 (100+150)")

// 	// // Check pod_metrics for 15:00
// 	// err = pool.QueryRow(context.Background(), "SELECT pod_usage_cpu_core_seconds FROM pod_metrics WHERE pod_id = $1 AND timestamp = $2", podID, "2025-05-17 15:00:00+00").Scan(&podUsage)
// 	// assert.NoError(t, err)
// 	// assert.Equal(t, 200.0, podUsage, "pod_metrics should have usage for 15:00")
// }

func TestProcessCSVInvalidTimestamp(t *testing.T) {
	pool := testutils.SetupTestDB(t)
	repo := db.NewRepository(pool)
	clusterID := "10f5a0f9-223a-41c1-8456-9a3eb0323a99"
	ctx := context.Background()

	csvData := `report_period_start,report_period_end,interval_start,interval_end,node,namespace,pod,pod_usage_cpu_core_seconds,pod_request_cpu_core_seconds,pod_limit_cpu_core_seconds,pod_usage_memory_byte_seconds,pod_request_memory_byte_seconds,pod_limit_memory_byte_seconds,node_capacity_cpu_cores,node_capacity_cpu_core_seconds,node_capacity_memory_bytes,node_capacity_memory_byte_seconds,node_role,resource_id,pod_labels
2025-05-17 00:00:00 +0000 UTC,2025-05-17 23:59:59 +0000 UTC,invalid-timestamp,2025-05-17 15:00:00 +0000 UTC,ip-10-0-1-63.ec2.internal,test,zip-1,100,200,300,1000,2000,3000,4,14400,17179869184,61729433600,worker,i-09ad6102842b9a786,app:web|label_rht_comp:EAP`

	reader := csv.NewReader(strings.NewReader(csvData))
	reader.Comma = ','
	reader.TrimLeadingSpace = true

	err := ProcessCSV(ctx, repo, reader, clusterID)
	assert.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM pod_metrics").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "No metrics should be inserted for invalid timestamp")
}

func TestProcessCSVMissingLabel(t *testing.T) {
	pool := testutils.SetupTestDB(t)
	repo := db.NewRepository(pool)
	clusterID := "10f5a0f9-223a-41c1-8456-9a3eb0323a99"
	ctx := context.Background()

	os.Setenv("POD_LABEL_KEYS", "label_rht_comp")
	defer os.Unsetenv("POD_LABEL_KEYS")

	csvData := `report_period_start,report_period_end,interval_start,interval_end,node,namespace,pod,pod_usage_cpu_core_seconds,pod_request_cpu_core_seconds,pod_limit_cpu_core_seconds,pod_usage_memory_byte_seconds,pod_request_memory_byte_seconds,pod_limit_memory_byte_seconds,node_capacity_cpu_cores,node_capacity_cpu_core_seconds,node_capacity_memory_bytes,node_capacity_memory_byte_seconds,node_role,resource_id,pod_labels
2025-05-17 00:00:00 +0000 UTC,2025-05-17 23:59:59 +0000 UTC,2025-05-17 14:00:00 +0000 UTC,2025-05-17 15:00:00 +0000 UTC,ip-10-0-1-63.ec2.internal,test,zip-1,100,200,300,1000,2000,3000,4,14400,17179869184,61729433600,worker,i-09ad6102842b9a786,app:web`

	reader := csv.NewReader(strings.NewReader(csvData))
	reader.Comma = ','
	reader.TrimLeadingSpace = true

	err := ProcessCSV(ctx, repo, reader, clusterID)
	assert.NoError(t, err)

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM pod_metrics").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "No pod metrics should be inserted without matching label")
}
