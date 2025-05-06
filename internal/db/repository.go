package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DailySummary represents a row in the daily_summary table
type DailySummary struct {
	Date           time.Time
	ClusterID      uuid.UUID
	ClusterName    string
	NodeName       string
	NodeIdentifier string
	NodeType       string
	CoreCount      int
	TotalHours     int
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertCluster(id uuid.UUID, name string) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO clusters (id, name) VALUES ($1, $2)
		 ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`,
		id, name)
	return err
}

func (r *Repository) UpsertNode(clusterID uuid.UUID, name, identifier, nodeType string) (uuid.UUID, error) {
	var id uuid.UUID
	query := `
		INSERT INTO nodes (id, cluster_id, name, identifier, type)
		VALUES (gen_random_uuid(), $1, $2, $3, $4)
		ON CONFLICT (identifier) DO UPDATE
		SET name = EXCLUDED.name, cluster_id = EXCLUDED.cluster_id, type = EXCLUDED.type
		RETURNING id`
	err := r.db.QueryRow(context.Background(), query, clusterID, name, identifier, nodeType).Scan(&id)
	return id, err
}

func (r *Repository) InsertMetric(nodeID uuid.UUID, timestamp time.Time, coreCount int, clusterID uuid.UUID) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO metrics (node_id, timestamp, core_count, cluster_id)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`, nodeID, timestamp, coreCount, clusterID)
	return err
}

func (r *Repository) UpdateDailySummary(nodeID uuid.UUID, timestamp time.Time, coreCount int) error {
	date := timestamp.Truncate(24 * time.Hour)
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO daily_summary (node_id, date, core_count, total_hours)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (node_id, date, core_count)
		 DO UPDATE SET total_hours = GREATEST(daily_summary.total_hours, 1)`,
		nodeID, date, coreCount)
	return err
}

func (r *Repository) QueryMetrics(start, end time.Time, clusterID, clusterName, nodeType string) ([]DailySummary, error) {
	query := `
		SELECT 
			ds.date,
			c.id AS cluster_id,
			c.name AS cluster_name,
			n.name AS node_name,
			COALESCE(n.identifier, '') AS node_identifier,
			COALESCE(n.type, '') AS node_type,
			ds.core_count, 
			ds.total_hours
		FROM daily_summary ds
		JOIN nodes n ON ds.node_id = n.id
		JOIN clusters c ON n.cluster_id = c.id
		WHERE ds.date BETWEEN $1 AND $2`
	args := []interface{}{start, end}
	if clusterID != "" {
		query += " AND c.id::text = $" + fmt.Sprint(len(args)+1)
		args = append(args, clusterID)
	}
	if clusterName != "" {
		query += " AND c.name = $" + fmt.Sprint(len(args)+1)
		args = append(args, clusterName)
	}
	if nodeType != "" {
		query += " AND n.type = $" + fmt.Sprint(len(args)+1)
		args = append(args, nodeType)
	}

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily_summary: %w", err)
	}
	defer rows.Close()

	var summaries []DailySummary
	for rows.Next() {
		var s DailySummary
		var nodeIdentifier, nodeType sql.NullString
		if err := rows.Scan(
			&s.Date,
			&s.ClusterID,
			&s.ClusterName,
			&s.NodeName,
			&nodeIdentifier,
			&nodeType,
			&s.CoreCount,
			&s.TotalHours,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		s.NodeIdentifier = nodeIdentifier.String
		s.NodeType = nodeType.String
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return summaries, nil
}
