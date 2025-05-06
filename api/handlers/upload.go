package handlers

import (
	"github.com/chambridge/cost-metrics-aggregator/internal/processor"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func UploadHandler(database *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed: " + err.Error()})
			return
		}
		defer file.Close()

		tempDir, err := os.MkdirTemp("", "upload")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp dir: " + err.Error()})
			return
		}
		defer os.RemoveAll(tempDir)

		tarPath := filepath.Join(tempDir, "upload.tar.gz")
		outFile, err := os.Create(tarPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
			return
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write file: " + err.Error()})
			return
		}

		err = processor.ProcessTar(c, tarPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process tar.gz: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "File processed successfully"})
	}
}
