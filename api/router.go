package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/chambridge/cost-metrics-aggregator/api/handlers"
	"github.com/chambridge/cost-metrics-aggregator/internal/config"
)

func SetupRouter(db *pgxpool.Pool, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/ingress/v1/upload", handlers.UploadHander(db))
		api.GET("/metrics/v1/node", handlers.QueryMetricsHandler(db))
	}

	return r
}
