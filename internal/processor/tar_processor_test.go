package processor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/chambridge/cost-metrics-aggregator/internal/processor/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTarGz creates a tar.gz file with the given files (map of filename to content)
func createTarGz(t *testing.T, files map[string]string) string {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")

	file, err := os.Create(tarPath)
	require.NoError(t, err)
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		err = tw.WriteHeader(hdr)
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	return tarPath
}

func TestProcessTar(t *testing.T) {
	pool := testutils.SetupTestDB(t)
	repo := db.NewRepository(pool)
	ctx := context.Background()

	os.Setenv("POD_LABEL_KEYS", "label_rht_comp")
	defer os.Unsetenv("POD_LABEL_KEYS")

	clusterID := "10f5a0f9-223a-41c1-8456-9a3eb0323a99"
	manifest := Manifest{
		ClusterID: clusterID,
		Files:     []string{"data.csv"},
		CRStatus: struct {
			ClusterID string `json:"clusterID"`
			Source    struct {
				Name string `json:"name"`
			} `json:"source"`
		}{
			ClusterID: clusterID,
			Source: struct {
				Name string `json:"name"`
			}{
				Name: "test-cluster",
			},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)

	csvData := `report_period_start,report_period_end,interval_start,interval_end,node,namespace,pod,pod_usage_cpu_core_seconds,pod_request_cpu_core_seconds,pod_limit_cpu_core_seconds,pod_usage_memory_byte_seconds,pod_request_memory_byte_seconds,pod_limit_memory_byte_seconds,node_capacity_cpu_cores,node_capacity_cpu_core_seconds,node_capacity_memory_bytes,node_capacity_memory_byte_seconds,node_role,resource_id,pod_labels
2025-05-17 00:00:00 +0000 UTC,2025-05-17 23:59:59 +0000 UTC,2025-05-17 14:00:00 +0000 UTC,2025-05-17 15:00:00 +0000 UTC,ip-10-0-1-63.ec2.internal,test,zip-1,100,200,300,1000,2000,3000,4,14400,17179869184,61729433600,worker,i-09ad6102842b9a786,app:web|label_rht_comp:EAP`

	tarPath := createTarGz(t, map[string]string{
		"manifest.json": string(manifestJSON),
		"data.csv":      csvData,
	})

	clusterUUID, _ := uuid.Parse(clusterID)
	nodeID, _ := uuid.Parse("fba4e7cd-4ee2-4f24-880d-082eb2b41128")
	podName := "zip-1"
	namespace := "test"
	component := "EAP"
	podID, err := repo.UpsertPod(clusterUUID, nodeID, podName, namespace, component)
	require.NoError(t, err)

	err = ProcessTar(ctx, tarPath, repo)
	assert.NoError(t, err)

	var clusterName string
	err = pool.QueryRow(context.Background(), "SELECT name FROM clusters WHERE id = $1", clusterUUID).Scan(&clusterName)
	assert.NoError(t, err)
	assert.Equal(t, "test-cluster", clusterName)

	var podUsage float64
	err = pool.QueryRow(context.Background(), "SELECT pod_usage_cpu_core_seconds FROM pod_metrics WHERE pod_id = $1 AND timestamp = '2025-05-17 14:00:00+00'", podID).Scan(&podUsage)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, podUsage, "pod_metrics should have usage from CSV")
}

func TestProcessTarMissingManifest(t *testing.T) {
	pool := testutils.SetupTestDB(t)
	repo := db.NewRepository(pool)
	ctx := context.Background()

	csvData := `report_period_start,report_period_end,interval_start,interval_end,node,namespace,pod,pod_usage_cpu_core_seconds,pod_request_cpu_core_seconds,pod_limit_cpu_core_seconds,pod_usage_memory_byte_seconds,pod_request_memory_byte_seconds,pod_limit_memory_byte_seconds,node_capacity_cpu_cores,node_capacity_cpu_core_seconds,node_capacity_memory_bytes,node_capacity_memory_byte_seconds,node_role,resource_id,pod_labels
2025-05-17 00:00:00 +0000 UTC,2025-05-17 23:59:59 +0000 UTC,2025-05-17 14:00:00 +0000 UTC,2025-05-17 15:00:00 +0000 UTC,ip-10-0-1-63.ec2.internal,test,zip-1,100,200,300,1000,2000,3000,4,14400,17179869184,61729433600,worker,i-09ad6102842b9a786,app:web|label_rht_comp:EAP`

	tarPath := createTarGz(t, map[string]string{
		"data.csv": csvData,
	})

	err := ProcessTar(ctx, tarPath, repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest.json")
}

func TestProcessTarInvalidCSV(t *testing.T) {
	pool := testutils.SetupTestDB(t)
	repo := db.NewRepository(pool)
	ctx := context.Background()

	clusterID := "10f5a0f9-223a-41c1-8456-9a3eb0323a99"
	manifest := Manifest{
		ClusterID: clusterID,
		Files:     []string{"data.csv"},
	}
	manifestJSON, _ := json.Marshal(manifest)

	invalidCSV := `invalid_header
bad,data`

	tarPath := createTarGz(t, map[string]string{
		"manifest.json": string(manifestJSON),
		"data.csv":      invalidCSV,
	})

	err := ProcessTar(ctx, tarPath, repo)
	assert.NoError(t, err) // ProcessTar logs errors but continues

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM pod_metrics").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "No metrics should be inserted for invalid CSV")
}
