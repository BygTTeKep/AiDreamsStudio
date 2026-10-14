package database

import (
	"context"
	"time"

	"github.com/BygTTeKep/AiDreamsStudio/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepository struct {
	DB *pgxpool.Pool
}

func NewDB(ctx context.Context, cfg config.DatabaseConfig) (*PGRepository, error) {
	conf, err := pgxpool.ParseConfig(cfg.Dsn)
	if err != nil {
		return nil, err
	}
	conf.MaxConns = 25
	conf.MinConns = 5
	conf.MaxConnLifetime = 1 * time.Hour
	conf.MaxConnIdleTime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return nil, err
	}
	return &PGRepository{
		DB: pool,
	}, nil
}

func (pg *PGRepository) Close() {
	pg.DB.Close()
}
