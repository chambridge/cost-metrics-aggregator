package processor

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/google/uuid"
)

// NodeData represents distinct node information for an hourly interval
type NodeData struct {
	NodeName         string
	ResourceID       string
	NodeRole         string
	CapacityCPUCores float64
	IntervalStart    time.Time
	ClusterID        string
}

// ProcessCSV processes a CSV reader, extracting distinct node data and inserting into nodes, metrics, and daily_summary
func ProcessCSV(ctx context.Context, repo *db.Repository, reader *csv.Reader, clusterID string) error {
	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV records: %w", err)
	}

	// Map header indices
	if len(records) < 1 {
		return fmt.Errorf("empty CSV file")
	}
	headers := records[0]
	headerIndices := make(map[string]int)
	for i, h := range headers {
		headerIndices[h] = i
	}

	// Process each record
	for _, record := range records[1:] {
		if len(record) < len(headers) {
			log.Printf("Skipping invalid record: %v", record)
			continue
		}

		intervalStartStr := record[headerIndices["interval_start"]]
		nodeName := record[headerIndices["node"]]
		resourceID := record[headerIndices["resource_id"]]
		nodeRole := record[headerIndices["node_role"]]
		capacityCPUStr := record[headerIndices["node_capacity_cpu_cores"]]

		intervalStart, err := time.Parse(time.RFC3339, intervalStartStr)
		if err != nil {
			log.Printf("Skipping record: invalid interval_start %s: %v", intervalStartStr, err)
			continue
		}

		capacityCPU, err := strconv.ParseFloat(capacityCPUStr, 64)
		if err != nil {
			log.Printf("Skipping record: invalid node_capacity_cpu_cores %s: %v", capacityCPUStr, err)
			continue
		}

		clusterUUID, err := uuid.Parse(clusterID)
		if err != nil {
			log.Printf("Skipping node %s: invalid cluster_id %s: %v", nodeName, clusterID, err)
			continue
		}

		// Prepare identifier (NULL if resource_id is empty)
		var identifier string
		if resourceID != "" {
			identifier = resourceID
		}

		// Insert into nodes table using Repository
		nodeType := nodeRole
		if nodeType == "" {
			nodeType = "worker" // Default
		}
		nodeID, err := repo.UpsertNode(clusterUUID, nodeName, identifier, nodeType)
		if err != nil {
			log.Printf("Failed to insert/update node %s: %v", nodeName, err)
			continue
		}

		// Insert into metrics table using Repository
		err = repo.InsertMetric(nodeID, intervalStart, int(capacityCPU), clusterUUID)
		if err != nil {
			log.Printf("Failed to insert metrics for node %s at %s: %v", nodeName, intervalStart, err)
			continue
		}

		// Update daily_summary using Repository
		err = repo.UpdateDailySummary(nodeID, intervalStart, int(capacityCPU))
		if err != nil {
			log.Printf("Failed to update daily_summary for node %s on %s: %v", nodeID, intervalStart, err)
			continue
		}
	}

	return nil
}
