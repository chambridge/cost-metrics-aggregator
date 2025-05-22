package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
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
	Limit       int    `form:"limit,default=100"`
	Offset      int    `form:"offset,default=0"`
}

type PodMetricsQueryParams struct {
	StartDate   string `form:"start_date"`
	EndDate     string `form:"end_date"`
	ClusterID   string `form:"cluster_id"`
	ClusterName string `form:"cluster_name"`
	Namespace   string `form:"namespace"`
	PodName     string `form:"pod_name"`
	Component   string `form:"component"`
	Limit       int    `form:"limit,default=100"`
	Offset      int    `form:"offset,default=0"`
}

// QueryNodeMetricsHandler handles the /api/metrics/v1/nodes endpoint, querying node_daily_summary
func QueryNodeMetricsHandler(database *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params NodeMetricsQueryParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
			return
		}

		// Validate limit
		if params.Limit <= 0 || params.Limit > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Limit must be between 1 and 1000"})
			return
		}
		if params.Offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Offset must be non-negative"})
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
		nodeMetrics, total, err := repo.QueryNodeMetrics(start, end, params.ClusterID, params.ClusterName, params.NodeType, params.Limit, params.Offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query node metrics: " + err.Error()})
			return
		}

		// Check Accept header
		accept := c.GetHeader("Accept")
		if accept == "text/csv" {
			var buf bytes.Buffer
			writer := csv.NewWriter(&buf)

			// Write CSV header
			header := []string{"Date", "ClusterID", "ClusterName", "NodeName", "NodeIdentifier", "NodeType", "CoreCount", "TotalHours"}
			if err := writer.Write(header); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV header: " + err.Error()})
				return
			}

			// Write CSV rows
			for _, metric := range nodeMetrics {
				row := []string{
					metric.Date.Format("2006-01-02"),
					metric.ClusterID.String(),
					metric.ClusterName,
					metric.NodeName,
					metric.NodeIdentifier,
					metric.NodeType,
					fmt.Sprintf("%d", metric.CoreCount),
					fmt.Sprintf("%d", metric.TotalHours),
				}
				if err := writer.Write(row); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV row: " + err.Error()})
					return
				}
			}

			writer.Flush()
			if err := writer.Error(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flush CSV: " + err.Error()})
				return
			}

			c.Header("Content-Type", "text/csv")
			c.Header("Content-Disposition", "attachment;filename=node_metrics.csv")
			c.String(http.StatusOK, buf.String())
			return
		}

		// JSON response with metadata
		c.JSON(http.StatusOK, gin.H{
			"metadata": gin.H{
				"total":  total,
				"limit":  params.Limit,
				"offset": params.Offset,
			},
			"data": nodeMetrics,
		})
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

		// Validate limit
		if params.Limit <= 0 || params.Limit > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Limit must be between 1 and 1000"})
			return
		}
		if params.Offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Offset must be non-negative"})
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
		podMetrics, total, err := repo.QueryPodMetrics(start, end, params.ClusterID, params.ClusterName, params.Namespace, params.PodName, params.Component, params.Limit, params.Offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pod metrics: " + err.Error()})
			return
		}

		// Check Accept header
		accept := c.GetHeader("Accept")
		if accept == "text/csv" {
			var buf bytes.Buffer
			writer := csv.NewWriter(&buf)

			// Write CSV header
			header := []string{"Date", "MaxCoresUsed", "TotalPodEffectiveCoreSeconds", "TotalHours", "ClusterID", "ClusterName", "Namespace", "PodName", "Component"}
			if err := writer.Write(header); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV header: " + err.Error()})
				return
			}

			// Write CSV rows
			for _, metric := range podMetrics {
				row := []string{
					metric.Date.Format("2006-01-02"),
					fmt.Sprintf("%.2f", metric.MaxCoresUsed),
					fmt.Sprintf("%.2f", metric.TotalPodEffectiveCoreSeconds),
					fmt.Sprintf("%d", metric.TotalHours),
					metric.ClusterID.String(),
					metric.ClusterName,
					metric.Namespace,
					metric.PodName,
					metric.Component,
				}
				if err := writer.Write(row); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV row: " + err.Error()})
					return
				}
			}

			writer.Flush()
			if err := writer.Error(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flush CSV: " + err.Error()})
				return
			}

			c.Header("Content-Type", "text/csv")
			c.Header("Content-Disposition", "attachment;filename=pod_metrics.csv")
			c.String(http.StatusOK, buf.String())
			return
		}

		// JSON response with metadata
		c.JSON(http.StatusOK, gin.H{
			"metadata": gin.H{
				"total":  total,
				"limit":  params.Limit,
				"offset": params.Offset,
			},
			"data": podMetrics,
		})
	}
}
