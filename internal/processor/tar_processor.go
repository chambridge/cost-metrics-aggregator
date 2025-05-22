package processor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/chambridge/cost-metrics-aggregator/internal/db"
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

// ProcessTar processes a tar.gz archive, extracting manifest.json and valid CSVs
func ProcessTar(ctx context.Context, tarPath string, repo *db.Repository) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var manifest Manifest
	manifestFound := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		filename := header.Name
		if strings.HasSuffix(filename, "manifest.json") {
			data, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("failed to read manifest.json: %w", err)
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("failed to parse manifest.json: %w", err)
			}
			manifestFound = true
			log.Printf("Processed manifest.json: cluster_id=%s", manifest.ClusterID)

			// Insert or update clusters using Repository
			clusterID, err := uuid.Parse(manifest.ClusterID)
			if err != nil {
				return fmt.Errorf("invalid cluster_id %s: %w", manifest.ClusterID, err)
			}
			clusterName := manifest.ClusterID // Default to cluster_id
			if manifest.CRStatus.Source.Name != "" {
				clusterName = manifest.CRStatus.Source.Name
			}
			err = repo.UpsertCluster(clusterID, clusterName)
			if err != nil {
				return fmt.Errorf("failed to insert/update cluster %s: %w", clusterID, err)
			}
			log.Printf("Inserted/updated cluster: id=%s, name=%s", clusterID, clusterName)
		}
	}

	if !manifestFound {
		return fmt.Errorf("no manifest.json found in tar archive")
	}

	// Reset tar reader to process CSVs
	file.Seek(0, io.SeekStart)
	gzr.Reset(file)
	tr = tar.NewReader(gzr)

	// Process only CSVs listed in manifest.files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		filename := header.Name
		if !strings.HasSuffix(filename, ".csv") {
			continue
		}

		// Check if filename is in manifest.files
		isValidFile := false
		for _, f := range manifest.Files {
			if f == filename {
				isValidFile = true
				break
			}
		}
		if !isValidFile {
			log.Printf("Skipping %s: not listed in manifest.files", filename)
			continue
		}

		// Read CSV content
		data, err := io.ReadAll(tr)
		if err != nil {
			log.Printf("Failed to read %s: %v", filename, err)
			continue
		}
		reader := csv.NewReader(strings.NewReader(string(data)))
		log.Printf("Processing CSV file: %s", filename)
		if err := ProcessCSV(ctx, repo, reader, manifest.ClusterID); err != nil {
			log.Printf("Failed to process %s: %v", filename, err)
			continue
		}
		log.Printf("Successfully processed %s", filename)
	}

	return nil
}
