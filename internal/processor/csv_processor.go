package processor

import (
	"encoding/csv"
	"strings"
	"time"
)

type NodeMetric struct {
	NodeName string
	NodeIdentifier string
	NodeType string
	Timestamp time.Time
	CoreCount int
}

func ProcessNodeCSV(csvData string) ([]NodeMetric, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var metrics []NodeMetric
	for i, record := range records {
		if i == 0 {
			continue //Skip header
		}
		timestamp, err := time.Parse(time.RFC3339, record[3])
		if err != nil {
			return nil, err
		}
		coreCount, err := strconv.Atoi(record[4])
		if err != nil {
			return nil, err
		}

		metrics = append (metrics, NodeMetric{
			NodeName: record[0],
			NodeIdentifier: record[1],
			NodeType: record[2],
			Timestamp: timestamp,
			CoreCount: coreCount,
		})
	}

	return metrics, nil
}
