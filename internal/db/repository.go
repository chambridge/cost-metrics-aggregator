package db

import (
	"context"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool)  *Repository{
	return &Repository{db: db}
}

func (r *Repository) UpsertCluster(name string) (int, error) {
	var id int
	err := r.db.QueryRow(context.Background(),
		`INSERT INTO clusters (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDE.name
		 RETURNING id`, name).Scan(&id)
	return id, err
}

func (r *Repository) UpsertNode(clusterID int, name, identifier, nodeType string) (int, error) {
	var id int
	err := r.db.QueryRow(context.Background(),
		`INSERT INTO nodes (cluster_id, name, identifier, type)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (identifier) DO UPDATE SET name = EXCLUDED.name, type = EXCLUDED.type
		 RETURNING id`, clusterID, name identifier, nodeType).Scan(&id)
	return id, err
}

func (r *Repository) InsertMetric(nodeID int, timestamp time.Time, coreCount int) error {
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO metrics (node_id, timestamp, core_count)
		 VALUES ($1, $2, $3)`, nodeID, timestamp, coreCount)
	return err
}

func (r*Repository) UpdateDailySummary(nodeID int, timestamp time.Time, coreCount int) error {
	date := timestamp.Truncate(24 * time.Hour)
	_, err := r.db.Exec(context.Background(),
		`INSERT INTO daily_summaries (node_id, date, core_count, total_hours)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (node_id, date, core_count)
		 DO UPDATE SET total_hours = dailysummaries.total_hours + 1`,
		nodeID, date, coreCount)
	return err
}

func (r *Repository) QueryMetrics(start, end time.Time, clusterID *int, clusterName, nodeType string) ([]DailySummary, error) {
	query := `
		SELECT ds.node_id, ds.date, ds.core_count, ds.total_hours
		FROM daily_summaries ds
		JOIN nodes n ON ds.node_id = n.id
		JOIN clusters c ON n.cluster_id = c.id
		WHERE ds.date BETWEEN $1 AND $2`
	args := []interface{}{start,end}
	if clusterID != nil {
		query += " AND c.id = $" + string(len(args)+1)
		args = append(args, *clusterID)
	}
	if clusterName != "" {
		query += " AND c.name = $" + string(len(args)+1)
		args = append(args, clusterName)
	}
	if nodeType != "" {
		query += " AND n.type = $" + string(len(args)+1)
		args = append(args, nodeType)
	}

	rows, err := r.db.Query(context.Background(), query, args ...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []DailySummary
	for rows.Next() {
		var s DailySummary
		if err := rows.Scan(&s.NodeID, &s.Date, &s.CoreCount, &s,TotalHours); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
