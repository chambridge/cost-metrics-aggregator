package handlers

import (
	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

type QueryParams struct {
	StartDate   string `form:"start_date" bining:"required"`
	EndDate     string `form:"end_date" bining:"required"`
	ClusterID   string `form:"cluster_id"`
	ClusterName string `form:"cluster_name"`
	NodeType    string `form:"node_type"`
}

func QueryMetricsHandler(database *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params QueryParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
			return
		}

		start, err := time.Parse("2006-01-02", params.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date"})
			return
		}

		end, err := time.Parse("2006-01-02", params.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date"})
			return
		}

		if start.IsZero() {
			start = time.Now().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().Day()+1)
		}
		if end.IsZero() {
			end = time.Now().Truncate(24 * time.Hour)
		}

		repo := db.NewRepository(database)
		metrics, err := repo.QueryMetrics(start, end, params.ClusterID, params.ClusterName, params.NodeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query metrics"})
			return
		}

		c.JSON(http.StatusOK, metrics)
	}
}
