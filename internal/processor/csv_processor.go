package processor

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chambridge/cost-metrics-aggregator/internal/db"
	"github.com/google/uuid"
)

// RequiredHeaders is the subset of CSV headers that must be present
var RequiredHeaders = []string{
	"report_period_start", "report_period_end", "interval_start", "interval_end",
	"node", "namespace", "pod", "pod_usage_cpu_core_seconds",
	"pod_request_cpu_core_seconds", "pod_limit_cpu_core_seconds",
	"pod_usage_memory_byte_seconds", "pod_request_memory_byte_seconds",
	"pod_limit_memory_byte_seconds", "node_capacity_cpu_cores",
	"node_capacity_cpu_core_seconds", "node_capacity_memory_bytes",
	"node_capacity_memory_byte_seconds", "node_role", "resource_id",
	"pod_labels",
}

// ProcessCSV processes a CSV reader, extracting distinct node data and inserting into data tables
func ProcessCSV(ctx context.Context, repo *db.Repository, reader *csv.Reader, clusterID string) error {
	// Configure CSV reader
	reader.Comma = ','
	reader.FieldsPerRecord = len(RequiredHeaders) // Expect 20 fields per record
	reader.TrimLeadingSpace = true

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV records: %w", err)
	}

	// Check if CSV is empty
	if len(records) < 1 {
		return fmt.Errorf("empty CSV file")
	}

	// Map header indices
	headers := records[0]
	headerIndices := make(map[string]int)
	for i, h := range headers {
		headerIndices[strings.TrimSpace(h)] = i
	}

	log.Printf("Headers: %v", headers)
	log.Printf("Header indices: %v", headerIndices)

	// Validate headers
	for _, required := range RequiredHeaders {
		if _, exists := headerIndices[required]; !exists {
			return fmt.Errorf("missing required header: %s", required)
		}
	}

	// Get pod label keys from environment, default to "label_rht_comp"
	podLabelKeysStr := os.Getenv("POD_LABEL_KEYS")
	if podLabelKeysStr == "" {
		podLabelKeysStr = "label_rht_comp"
	}
	podLabelKeys := strings.Split(podLabelKeysStr, ",")
	podLabelKeySet := make(map[string]struct{})
	for _, key := range podLabelKeys {
		podLabelKeySet[strings.TrimSpace(key)] = struct{}{}
	}

	// Process each record
	for i, record := range records[1:] {
		if len(record) != len(headers) {
			log.Printf("Skipping record %d: expected %d fields, got %d: %v", i+1, len(headers), len(record), record)
			continue
		}

		log.Printf("Processing record %d: %v", i+1, record)

		intervalStartStr := record[headerIndices["interval_start"]]
		nodeName := record[headerIndices["node"]]
		resourceID := record[headerIndices["resource_id"]]
		nodeRole := record[headerIndices["node_role"]]
		capacityCPUStr := record[headerIndices["node_capacity_cpu_cores"]]
		podName := record[headerIndices["pod"]]
		namespace := record[headerIndices["namespace"]]
		podLabels := record[headerIndices["pod_labels"]]
		podUsageStr := record[headerIndices["pod_usage_cpu_core_seconds"]]
		podRequestStr := record[headerIndices["pod_request_cpu_core_seconds"]]
		nodeCapacityCPUCoreSecondsStr := record[headerIndices["node_capacity_cpu_core_seconds"]]

		intervalStart, err := time.Parse("2006-01-02 15:04:05 +0000 MST", intervalStartStr)
		if err != nil {
			log.Printf("Skipping record %d: invalid interval_start %s: %v", i+1, intervalStartStr, err)
			continue
		}

		capacityCPU, err := strconv.ParseFloat(capacityCPUStr, 64)
		if err != nil {
			log.Printf("Skipping record %d: invalid node_capacity_cpu_cores %s: %v", i+1, capacityCPUStr, err)
			continue
		}

		clusterUUID, err := uuid.Parse(clusterID)
		if err != nil {
			log.Printf("Skipping record %d: invalid cluster_id %s: %v", i+1, clusterID, err)
			continue
		}

		podUsage, err := strconv.ParseFloat(podUsageStr, 64)
		if err != nil {
			log.Printf("Skipping record %d: invalid pod_usage_cpu_core_seconds %s: %v", i+1, podUsageStr, err)
			continue
		}

		podRequest, err := strconv.ParseFloat(podRequestStr, 64)
		if err != nil {
			log.Printf("Record %d: invalid pod_request_cpu_core_seconds %s: %v - setting to 0.0", i+1, podRequestStr, err)
			podRequest = 0.0
		}

		nodeCapacityCPUCoreSeconds, err := strconv.ParseFloat(nodeCapacityCPUCoreSecondsStr, 64)
		if err != nil {
			log.Printf("Skipping record %d: invalid node_capacity_cpu_core_seconds %s: %v", i+1, nodeCapacityCPUCoreSecondsStr, err)
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
			log.Printf("Skipping record %d: failed to insert/update node %s: %v", i+1, nodeName, err)
			continue
		}

		// Insert into node_metrics table using Repository
		err = repo.InsertNodeMetric(nodeID, intervalStart, int(capacityCPU), clusterUUID)
		if err != nil {
			log.Printf("Skipping record %d: failed to insert node_metrics for node %s at %s: %v", i+1, nodeName, intervalStart, err)
			continue
		}

		// Update node_daily_summary using Repository
		err = repo.UpdateNodeDailySummary(nodeID, intervalStart, int(capacityCPU))
		if err != nil {
			log.Printf("Skipping record %d: failed to update node_daily_summary for node %s on %s: %v", i+1, nodeID, intervalStart, err)
			continue
		}

		// Process pod if it has a matching label key
		labels := strings.Split(podLabels, "|")
		labelMap := make(map[string]string)
		for _, label := range labels {
			parts := strings.SplitN(label, ":", 2)
			if len(parts) == 2 {
				labelMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		var component string
		hasMatchingLabel := false
		for key := range labelMap {
			if _, exists := podLabelKeySet[key]; exists {
				hasMatchingLabel = true
				if key == "label_rht_comp" {
					component = labelMap[key]
				}
			}
		}

		if !hasMatchingLabel {
			log.Printf("Skipping pod %s in namespace %s: no matching label key in %v", podName, namespace, podLabelKeys)
			continue
		}

		// Insert into pods table
		podID, err := repo.UpsertPod(clusterUUID, nodeID, podName, namespace, component)
		if err != nil {
			log.Printf("Skipping record %d: failed to insert/update pod %s in namespace %s: %v", i+1, podName, namespace, err)
			continue
		}

		// Insert into pod_metrics table
		err = repo.InsertPodMetric(podID, intervalStart, podUsage, podRequest, nodeCapacityCPUCoreSeconds, int(capacityCPU))
		if err != nil {
			log.Printf("Skipping record %d: failed to insert pod_metrics for pod %s at %s: %v", i+1, podName, intervalStart, err)
			continue
		}

		// Update pod_daily_summary
		podEffectiveCoreSeconds := podUsage
		if podRequest > podUsage {
			podEffectiveCoreSeconds = podRequest
		}
		podEffectiveCoreUsage := 0.0
		if nodeCapacityCPUCoreSeconds > 0 {
			podEffectiveCoreUsage = podEffectiveCoreSeconds / nodeCapacityCPUCoreSeconds
		}
		err = repo.UpdatePodDailySummary(podID, intervalStart, podEffectiveCoreSeconds, podEffectiveCoreUsage)
		if err != nil {
			log.Printf("Skipping record %d: failed to update pod_daily_summary for pod %s on %s: %v", i+1, podName, intervalStart, err)
			continue
		}
	}

	return nil
}
