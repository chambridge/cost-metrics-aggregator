package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NodeDailySummary represents a row in the node_daily_summary table
type NodeDailySummary struct {
	Date           time.Time
	ClusterID      uuid.UUID
	ClusterName    string
	NodeName       string
	NodeIdentifier string
	NodeType       string
	CoreCount      int
	TotalHours     int
}

// PodDailySummary represents a row in the pod_daily_summary table
type PodDailySummary struct {
	Date                         time.Time
	MaxCoresUsed                 float64
	TotalPodEffectiveCoreSeconds float64
	TotalHours                   int
	ClusterID                    uuid.UUID
	ClusterName                  string
	PodName                      string
	Namespace                    string
	Component                    string
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

func (r *Repository) InsertNodeMetric(nodeID uuid.UUID, timestamp time.Time, coreCount int, clusterID uuid.UUID) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO node_metrics (node_id, timestamp, core_count, cluster_id)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING`, nodeID, timestamp, coreCount, clusterID)
	return err
}

func (r *Repository) UpdateNodeDailySummary(nodeID uuid.UUID, timestamp time.Time, coreCount int) error {
	date := timestamp.Truncate(24 * time.Hour)
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO node_daily_summary (node_id, date, core_count, total_hours)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (node_id, date, core_count)
		 DO UPDATE SET total_hours = GREATEST(daily_summary.total_hours, 1)`,
		nodeID, date, coreCount)
	return err
}

func (r *Repository) UpsertPod(clusterID, nodeID uuid.UUID, name, namespace, component string) (uuid.UUID, error) {
	var id uuid.UUID
	query := `
		INSERT INTO pods (id, cluster_id, node_id, name, namespace, component)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		ON CONFLICT (name, namespace, cluster_id) DO UPDATE
		SET node_id = EXCLUDED.node_id, component = EXCLUDED.component
		RETURNING id`
	err := r.db.QueryRow(context.Background(), query, clusterID, nodeID, name, namespace, component).Scan(&id)
	return id, err
}

func (r *Repository) InsertPodMetric(podID uuid.UUID, timestamp time.Time, podUsage, podRequest, nodeCapacityCPUCoreSeconds float64, nodeCapacityCPUCores int) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO pod_metrics (
			id, pod_id, timestamp, pod_usage_cpu_core_seconds, 
			pod_request_cpu_core_seconds, node_capacity_cpu_core_seconds, 
			node_capacity_cpu_cores
		) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO NOTHING`,
		podID, timestamp, podUsage, podRequest, nodeCapacityCPUCoreSeconds, nodeCapacityCPUCores)
	return err
}

func (r *Repository) UpdatePodDailySummary(podID uuid.UUID, timestamp time.Time, podEffectiveCoreSeconds, podEffectiveCoreUsage float64) error {
	date := timestamp.Truncate(24 * time.Hour)
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO pod_daily_summary (
			pod_id, date, max_cores_used, total_pod_effective_core_seconds, total_hours
		) VALUES ($1, $2, $3, $4, 1)
		 ON CONFLICT (pod_id, date) DO UPDATE
		 SET max_cores_used = GREATEST(pod_daily_summary.max_cores_used, EXCLUDED.max_cores_used),
		     total_pod_effective_core_seconds = pod_daily_summary.total_pod_effective_core_seconds + EXCLUDED.total_pod_effective_core_seconds,
		     total_hours = pod_daily_summary.total_hours + 1`,
		podID, date, podEffectiveCoreUsage, podEffectiveCoreSeconds)
	return err
}

func (r *Repository) QueryNodeMetrics(start, end time.Time, clusterID, clusterName, nodeType string) ([]NodeDailySummary, error) {
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
		FROM node_daily_summary ds
		JOIN nodes n ON ds.node_id = n.id
		JOIN clusters c ON n.cluster_id = c.id
		WHERE ds.date BETWEEN $1 AND $2`
	args := []interface{}{start, end}
	if clusterID != "" {
		query += " AND c.id::text = $" + fmt.Sprint(len(args)+1)
		args = append(args, clusterID)
	}
	if clusterName != "" {
		query += " AND c.name ILIKE $" + fmt.Sprint(len(args)+1)
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

	var summaries []NodeDailySummary
	for rows.Next() {
		var s NodeDailySummary
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

func (r *Repository) QueryPodMetrics(start, end time.Time, clusterID, clusterName, namespace, podName, component string) ([]PodDailySummary, error) {
	query := `
		SELECT 
			ds.date,
			ds.max_cores_used,
			ds.total_pod_effective_core_seconds,
			ds.total_hours,
			c.id AS cluster_id,
			c.name AS cluster_name,
			p.namespace,
			p.name AS pod_name,
			COALESCE(p.component, '') AS component
		FROM pod_daily_summary ds
		JOIN pods p ON ds.pod_id = p.id
		JOIN clusters c ON p.cluster_id = c.id
		WHERE ds.date BETWEEN $1 AND $2`
	args := []interface{}{start, end}
	if clusterID != "" {
		query += " AND c.id::text = $" + fmt.Sprint(len(args)+1)
		args = append(args, clusterID)
	}
	if clusterName != "" {
		query += " AND c.name ILIKE $" + fmt.Sprint(len(args)+1)
		args = append(args, "%"+clusterName+"%")
	}
	if namespace != "" {
		query += " AND p.namespace ILIKE $" + fmt.Sprint(len(args)+1)
		args = append(args, "%"+namespace+"%")
	}
	if podName != "" {
		query += " AND p.name ILIKE $" + fmt.Sprint(len(args)+1)
		args = append(args, "%"+podName+"%")
	}
	if component != "" {
		query += " AND p.component ILIKE $" + fmt.Sprint(len(args)+1)
		args = append(args, "%"+component+"%")
	}

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pod_daily_summary: %w", err)
	}
	defer rows.Close()

	var summaries []PodDailySummary
	for rows.Next() {
		var s PodDailySummary
		var component sql.NullString
		if err := rows.Scan(
			&s.Date,
			&s.MaxCoresUsed,
			&s.TotalPodEffectiveCoreSeconds,
			&s.TotalHours,
			&s.ClusterID,
			&s.ClusterName,
			&s.Namespace,
			&s.PodName,
			&component,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		s.Component = component.String
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return summaries, nil
}
