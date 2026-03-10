package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/mrasyidgpfl/shopfake/internal/models"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Migrate() error {
	query := `
		CREATE TABLE IF NOT EXISTS events (
			id VARCHAR(36) PRIMARY KEY,
			type VARCHAR(32) NOT NULL,
			region VARCHAR(4) NOT NULL,
			user_id VARCHAR(36) NOT NULL,
			amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency VARCHAR(8) NOT NULL DEFAULT '',
			timestamp TIMESTAMPTZ NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_events_region ON events(region);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
		CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) InsertEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (id, type, region, user_id, amount, currency, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, query, event.ID, event.Type, event.Region,
		event.UserID, event.Amount, event.Currency, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetRegionStats(ctx context.Context) ([]*models.RegionStats, error) {
	query := `
		SELECT
			region,
			COUNT(*) FILTER (WHERE type IN ('ORDER_PLACED', 'ORDER_COMPLETED')) as order_count,
			COUNT(*) FILTER (WHERE type IN ('ORDER_FAILED', 'PAYMENT_FAILED')) as failure_count,
			COALESCE(SUM(amount) FILTER (WHERE type = 'PAYMENT_SUCCESS'), 0) as revenue
		FROM events
		WHERE timestamp > NOW() - INTERVAL '1 hour'
		GROUP BY region
		ORDER BY region
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query region stats: %w", err)
	}
	defer rows.Close()

	var stats []*models.RegionStats
	for rows.Next() {
		s := &models.RegionStats{}
		if err := rows.Scan(&s.Region, &s.OrderCount, &s.FailureCount, &s.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

func (s *PostgresStore) GetEventCountByType(ctx context.Context) (map[models.EventType]int64, error) {
	query := `
		SELECT type, COUNT(*)
		FROM events
		WHERE timestamp > NOW() - INTERVAL '1 hour'
		GROUP BY type
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[models.EventType]int64)
	for rows.Next() {
		var eventType models.EventType
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		counts[eventType] = count
	}

	return counts, rows.Err()
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
