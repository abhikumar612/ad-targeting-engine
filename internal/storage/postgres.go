package storage

import (
	"ad-targeting-engine/internal/config"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct { pool *pgxpool.Pool }

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
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = int32(cfg.Postgres.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.Postgres.MaxIdleConns)
	poolCfg.HealthCheckPeriod = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { if s.pool != nil { s.pool.Close() } }

func (s *Store) LoadActiveCampaigns(ctx context.Context) ([]CampaignRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Dummy: replace with actual SQL logic
	return []CampaignRow{}, nil
}

func (s *Store) ListenChannel() string { return "tg_data_change" }

func (s *Store) PgxPool() *pgxpool.Pool {
	if s.pool == nil {
		panic(errors.New("pgx pool is nil"))
	}
	return s.pool
}

func (s *Store) DSNRedacted() string { return fmt.Sprintf("postgres://***:***@host:port/db") }