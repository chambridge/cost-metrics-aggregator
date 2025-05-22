package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Manifest represents the structure of manifest.json
type Manifest struct {
	ClusterID string   `json:"cluster_id"`
	Files     []string `json:"files"`
	CRStatus  struct {
		ClusterID string `json:"clusterID"`
		Source    struct {
			Name string `json:"name"`
		} `json:"source"`
	} `json:"cr_status"`
}

// generateCSV creates a CSV file with sample data for the given timestamp
func generateCSV(filename string, clusterID string, startTime, endTime time.Time) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	headers := []string{
		"report_period_start", "report_period_end", "interval_start", "interval_end",
		"node", "namespace", "pod", "pod_usage_cpu_core_seconds",
		"pod_request_cpu_core_seconds", "pod_limit_cpu_core_seconds",
		"pod_usage_memory_byte_seconds", "pod_request_memory_byte_seconds",
		"pod_limit_memory_byte_seconds", "node_capacity_cpu_cores",
		"node_capacity_cpu_core_seconds", "node_capacity_memory_bytes",
		"node_capacity_memory_byte_seconds", "node_role", "resource_id",
		"pod_labels",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Generate sample data for each hour in the 24-hour period
	currentTime := startTime
	for currentTime.Before(endTime) {
		intervalEnd := currentTime.Add(time.Hour)
		record := []string{
			startTime.Format("2006-01-02 15:04:05 +0000 MST"),           // report_period_start
			endTime.Format("2006-01-02 15:04:05 +0000 MST"),             // report_period_end
			currentTime.Format("2006-01-02 15:04:05 +0000 MST"),         // interval_start
			intervalEnd.Format("2006-01-02 15:04:05 +0000 MST"),         // interval_end
			fmt.Sprintf("node-%s-1", clusterID[:8]),                     // node
			"test-namespace",                                            // namespace
			fmt.Sprintf("pod-%s-%d", clusterID[:8], currentTime.Hour()), // pod
			"100.5",                               // pod_usage_cpu_core_seconds
			"200.0",                               // pod_request_cpu_core_seconds
			"300.0",                               // pod_limit_cpu_core_seconds
			"1073741824",                          // pod_usage_memory_byte_seconds
			"2147483648",                          // pod_request_memory_byte_seconds
			"4294967296",                          // pod_limit_memory_byte_seconds
			"4",                                   // node_capacity_cpu_cores
			"14400",                               // node_capacity_cpu_core_seconds (4 cores * 3600 seconds)
			"17179869184",                         // node_capacity_memory_bytes
			"61728312345600",                      // node_capacity_memory_byte_seconds
			"worker",                              // node_role
			fmt.Sprintf("resource-%s", clusterID), // resource_id
			"label_rht_comp:test-component|app:test-app", // pod_labels
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
		currentTime = intervalEnd
	}

	return nil
}

func main() {
	// Define output directory and file
	outputDir := "test_upload"
	outputFile := "test_upload.tar.gz"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate manifest
	clusterID := uuid.New().String()
	manifest := Manifest{
		ClusterID: clusterID,
		Files:     []string{"data1.csv", "data2.csv"},
	}
	manifest.CRStatus.ClusterID = clusterID
	manifest.CRStatus.Source.Name = "test-cluster"

	// Write manifest.json
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal manifest: %v\n", err)
		os.Exit(1)
	}
	manifestPath := filepath.Join(outputDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write manifest.json: %v\n", err)
		os.Exit(1)
	}

	// Generate time range for the previous 24 hours
	endTime := time.Now().UTC().Truncate(time.Hour)
	startTime := endTime.Add(-24 * time.Hour)

	// Generate CSV files
	for _, csvFile := range manifest.Files {
		csvPath := filepath.Join(outputDir, csvFile)
		if err := generateCSV(csvPath, clusterID, startTime, endTime); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate CSV %s: %v\n", csvFile, err)
			os.Exit(1)
		}
	}

	// Create tar.gz archive
	tarFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create tar.gz file: %v\n", err)
		os.Exit(1)
	}
	defer tarFile.Close()

	gzw := gzip.NewWriter(tarFile)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// Add manifest.json to tar
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stat manifest.json: %v\n", err)
		os.Exit(1)
	}
	header, err := tar.FileInfoHeader(manifestInfo, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create tar header for manifest.json: %v\n", err)
		os.Exit(1)
	}
	header.Name = "manifest.json"
	if err := tw.WriteHeader(header); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write tar header for manifest.json: %v\n", err)
		os.Exit(1)
	}
	if _, err := tw.Write(manifestData); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write manifest.json to tar: %v\n", err)
		os.Exit(1)
	}

	// Add CSV files to tar
	for _, csvFile := range manifest.Files {
		csvPath := filepath.Join(outputDir, csvFile)
		data, err := os.ReadFile(csvPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read CSV %s: %v\n", csvFile, err)
			os.Exit(1)
		}
		csvInfo, err := os.Stat(csvPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stat CSV %s: %v\n", csvFile, err)
			os.Exit(1)
		}
		header, err := tar.FileInfoHeader(csvInfo, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tar header for %s: %v\n", csvFile, err)
			os.Exit(1)
		}
		header.Name = csvFile
		if err := tw.WriteHeader(header); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write tar header for %s: %v\n", csvFile, err)
			os.Exit(1)
		}
		if _, err := tw.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s to tar: %v\n", csvFile, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Successfully created %s with manifest.json and %v\n", outputFile, manifest.Files)
}
