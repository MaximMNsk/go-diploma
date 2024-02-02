package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Database struct {
	pool *pgxpool.Pool
}

func (db *Database) Connect(ctx context.Context, connectString string) error {
	cfg, err := pgxpool.ParseConfig(connectString)
	if err != nil {
		return err
	}
	cfg.MaxConns = 16
	cfg.MinConns = 1
	cfg.HealthCheckPeriod = 1 * time.Minute
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 2 * time.Second

	db.pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetDB() *pgxpool.Pool {
	return db.pool
}

func (db *Database) Close() {
	db.pool.Close()
}
