package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var (
	//go:embed migrations/*.sql
	migrationsDir embed.FS
	ErrorInit     = errors.New(`database initialization error! `)
	ErrorMigrate  = errors.New(`migrate error! `)
)

type Database struct {
	Pool *pgxpool.Pool
	DSN  string
}

func (db *Database) Init(ctx context.Context, dsn string) error {
	db.DSN = dsn
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return err
	}
	cfg.MaxConns = 16
	cfg.MinConns = 1
	cfg.HealthCheckPeriod = 1 * time.Minute
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 2 * time.Second

	db.Pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) Close() {
	db.Pool.Close()
}

func (db *Database) PrepareDB() error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance(
		`iofs`,
		d,
		db.DSN,
	)
	if err != nil {
		return fmt.Errorf(ErrorInit.Error(), err)
	}
	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf(ErrorMigrate.Error(), err)
	}

	return nil
}
