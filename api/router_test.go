package api

import (
	"testing"

	"github.com/chambridge/cost-metrics-aggregator/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupRouter(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	db := &pgxpool.Pool{} // Mock pool, not used in routing
	cfg := &config.Config{
		ServerAddress: ":8080",
		DatabaseURL:   "postgres://user:pass@localhost:5432/dbname",
	}

	// Act
	router := SetupRouter(db, cfg)

	// Assert
	require.NotNil(t, router, "Router should not be nil")
	routes := router.Routes()

	// Expected routes
	expectedRoutes := []struct {
		method string
		path   string
	}{
		{method: "POST", path: "/api/ingress/v1/upload"},
		{method: "GET", path: "/api/metrics/v1/nodes"},
		{method: "GET", path: "/api/metrics/v1/pods"},
	}

	// Verify all expected routes exist
	for _, expected := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == expected.method && route.Path == expected.path {
				found = true
				assert.NotNil(t, route.HandlerFunc, "Handler for %s %s should be set", expected.method, expected.path)
				break
			}
		}
		assert.True(t, found, "Route %s %s should be registered", expected.method, expected.path)
	}

	// Verify no unexpected routes
	for _, route := range routes {
		isExpected := false
		for _, expected := range expectedRoutes {
			if route.Method == expected.method && route.Path == expected.path {
				isExpected = true
				break
			}
		}
		assert.True(t, isExpected, "Unexpected route found: %s %s", route.Method, route.Path)
	}

	// Verify route count
	assert.Equal(t, 3, len(routes), "Router should have exactly 3 routes")
}

func TestSetupRouter_GroupPrefix(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	db := &pgxpool.Pool{}
	cfg := &config.Config{}

	// Act
	router := SetupRouter(db, cfg)

	// Assert
	routes := router.Routes()
	for _, route := range routes {
		assert.True(t, route.Path == "/api/ingress/v1/upload" ||
			route.Path == "/api/metrics/v1/nodes" ||
			route.Path == "/api/metrics/v1/pods",
			"Route %s should be under /api group", route.Path)
	}
}
