package handlers

import (
	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

type QueryParams struct {
	StartDate   string `form:"start_date"`
	EndDate     string `form:"end_date"`
	ClusterID   string `form:"cluster_id"`
	ClusterName string `form:"cluster_name"`
	NodeType    string `form:"node_type"`
}

func QueryMetricsHandler(database *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params QueryParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
			return
		}

		// Set default dates: start_date = beginning of current month, end_date = current day
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := now.Truncate(24 * time.Hour)

		// Parse start_date if provided
		if params.StartDate != "" {
			var err error
			start, err = time.Parse("2006-01-02", params.StartDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date: " + err.Error()})
				return
			}
		}

		// Parse end_date if provided
		if params.EndDate != "" {
			var err error
			end, err = time.Parse("2006-01-02", params.EndDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date: " + err.Error()})
				return
			}
		}

		repo := db.NewRepository(database)
		metrics, err := repo.QueryMetrics(start, end, params.ClusterID, params.ClusterName, params.NodeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query metrics: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, metrics)
	}
}
