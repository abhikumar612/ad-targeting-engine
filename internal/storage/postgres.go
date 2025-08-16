package storage

import (
	"ad-targeting-engine/internal/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type CampaignRow struct {
	ID       string
	Name     string
	ImageURL string
	CTA      string
	Status   string
	Rules    []RuleRow
}

type RuleRow struct {
	Dimension   string
	IsInclusion bool
	Values      []string
}

func New(ctx context.Context, cfg config.Config) (*Store, error) {
	dsn := cfg.DSN()
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres DSN: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.Postgres.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.Postgres.MaxIdleConns)
	poolCfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// LoadActiveCampaigns loads all active campaigns + their rules
func (s *Store) LoadActiveCampaigns(ctx context.Context) ([]CampaignRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name, c.image_url, c.cta, c.status,
		       r.dimension, r.type, r.value
		FROM campaigns c
		LEFT JOIN targeting_rules r ON r.campaign_id = c.id
		WHERE c.status = 'ACTIVE'
		ORDER BY c.id
	`)
	if err != nil {
		return nil, fmt.Errorf("query campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := map[string]*CampaignRow{}

	for rows.Next() {
		var (
			id, name, image, cta, status string
			dim, typ, val sql.NullString
		)
		if err := rows.Scan(&id, &name, &image, &cta, &status, &dim, &typ, &val); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		c, ok := campaigns[id]
		if !ok {
			c = &CampaignRow{
				ID:       id,
				Name:     name,
				ImageURL: image,
				CTA:      cta,
				Status:   status,
			}
			campaigns[id] = c
		}

		if dim.Valid && typ.Valid && val.Valid {
			c.Rules = append(c.Rules, RuleRow{
				Dimension:   dim.String,
				IsInclusion: typ.String == "INCLUDE",
				Values:      []string{val.String},
			})
		}
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// flatten map → slice
	out := make([]CampaignRow, 0, len(campaigns))
	for _, c := range campaigns {
		out = append(out, *c)
	}

	return out, nil
}

func (s *Store) ListenChannel() string {
	return "tg_data_change"
}

func (s *Store) PgxPool() *pgxpool.Pool {
	if s.pool == nil {
		panic(errors.New("pgx pool is nil"))
	}
	return s.pool
}

func (s *Store) DSNRedacted() string {
	return fmt.Sprintf("postgres://***:***@host:port/db")
}