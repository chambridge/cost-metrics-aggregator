package handlers

import (
	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"time"
)

type NodeMetricsQueryParams struct {
	StartDate   string `form:"start_date"`
	EndDate     string `form:"end_date"`
	ClusterID   string `form:"cluster_id"`
	ClusterName string `form:"cluster_name"`
	NodeType    string `form:"node_type"`
}

type PodMetricsQueryParams struct {
	StartDate   string `form:"start_date"`
	EndDate     string `form:"end_date"`
	ClusterID   string `form:"cluster_id"`
	ClusterName string `form:"cluster_name"`
	Namespace   string `form:"namespace"`
	PodName     string `form:"pod_name"`
	Component   string `form:"component"`
}

// QueryPodMetricsHandler handles the /api/metrics/v1/nodes endpoint, querying node_daily_summary
func QueryNodeMetricsHandler(database *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params NodeMetricsQueryParams
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
		node_metrics, err := repo.QueryNodeMetrics(start, end, params.ClusterID, params.ClusterName, params.NodeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query node metrics: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, node_metrics)
	}
}

// QueryPodMetricsHandler handles the /api/metrics/v1/pods endpoint, querying pod_daily_summary
func QueryPodMetricsHandler(database *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params PodMetricsQueryParams
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
		pod_metrics, err := repo.QueryPodMetrics(start, end, params.ClusterID, params.ClusterName, params.Namespace, params.PodName, params.Component)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pod metrics: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, pod_metrics)
	}
}
