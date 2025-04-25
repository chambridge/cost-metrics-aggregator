package handers

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/pgxpool"
	"github.com/chambridge/cost-metrics-aggregator/api/internal/db"
	"github.com/chambridge/cost-metrics-aggregator/api/internal/processor"
)

func UploadHandler(db *pgxpool.Pool) gin.HanderFunc {
	return func(c *gin.Context) {
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed"})
			return
		}
		defer file.Close()

		tempDir, err := os.MkdirTemp("", "upload")
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "Failed to create temp dir"})
			return
		}
		defer os.RemoveAll(tempDir)

		tarPath := filepath.Join(tempDir, "upload.targ.gz")
		outFile, err := os.Create(tarPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "Failed to save file"})
			return
		}
		defer outFile.Close()
		
		_, err = io.Copy(outFile, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "Failed to write file")
			return
		}

		manifest, nodeCSV, err := processor.ExtractTarGz(tarPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process tar.gz: " + err.Error()})
			return
		}

		var manifestData struct {
			ClusterName string `json:"cluster_name"`
			NodeFile string `json:"node_file"`
		}
		if err := json.Unmarshal([] byte(manifest), &manifestData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid manifest"})
			return
		}

		repo := db.NewRepository(db)
		clusterID, err := repo.UpsertCluster(manifestData.ClusterName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cluster"})
			return
		}

		metrics, err := processor.ProcessNodeCSV(nodeCSV)
		if err != nil {
			c.JOSN(http.StatusBadRequest, g.H{"error": "Failed to process CSV: " + err.Error()})
			return
		}

		for _, metric := range metrics {
			nodeID, err := repo.UpsertNode(clusterID, metric.NodeName, metric.NodeIdentifier, metric.nodeType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save node"})
				return
			}
			err = repo.InsertMetric(nodeID, metric.Timestamp, metric.CoreCount)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Faled to save metric"})
				return
			}
			err = repo.UpdateDailySummary(nodeID, metric.Timestamp, metric.CoreCount)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Faled to update summary"})
				return
			}
		}

		c.JSON(http.StatusOK, g.H{"message": "File processed successfully"})
	}
}