package repository

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/repository/postgres"
	"github.com/deriv/arcade/internal/series"
	"github.com/deriv/arcade/internal/trading"
	"github.com/golang-migrate/migrate/v4"
	pgxmig "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed sql/migrations
var sqlFS embed.FS

// Manager manages repository instances and handles schema migrations
type Manager struct {
	pool         *pgxpool.Pool
	accountsRepo accounts.Repository
	seriesRepo   series.Repository
	tradingRepo  trading.Repository
}

// New creates a new repository manager, connects to database, and initializes schema
func New(ctx context.Context, databaseURL string, maxConns, minConns int) (*Manager, error) {
	// Parse database configuration
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// This is needed because we use connection pooler to connect to supabase.
	// We don't support IPv6, so that's our only option to connect.
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	poolConfig.MaxConns = int32(maxConns)
	poolConfig.MinConns = int32(minConns)

	// Create database pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test database connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &Manager{
		pool:         pool,
		accountsRepo: postgres.NewAccountsRepository(pool),
		seriesRepo:   postgres.NewSeriesRepository(pool),
		tradingRepo:  postgres.NewTradingRepository(pool),
	}

	// Initialize tables (run migrations if needed)
	if err := manager.initTables(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return manager, nil
}

// AccountsRepository returns the accounts repository
func (m *Manager) AccountsRepository() accounts.Repository {
	return m.accountsRepo
}

// SeriesRepository returns the series repository
func (m *Manager) SeriesRepository() series.Repository {
	return m.seriesRepo
}

// TradingRepository returns the trading repository
func (m *Manager) TradingRepository() trading.Repository {
	return m.tradingRepo
}

// MigrationNeeded checks if any migrations are needed
func (m *Manager) MigrationNeeded() (bool, error) {
	db := stdlib.OpenDBFromPool(m.pool)
	defer db.Close()

	dbDriver, err := pgxmig.WithInstance(db, &pgxmig.Config{})
	if err != nil {
		return false, fmt.Errorf("failed to create migration db instance: %w", err)
	}
	defer dbDriver.Close()

	v, d, err := dbDriver.Version()
	if err != nil {
		return false, fmt.Errorf("failed to get migration version: %w", err)
	}
	if d {
		// If dirty, migration is needed
		return true, nil
	}
	if v < 0 {
		// Never migrated, migration is needed
		return true, nil
	}

	srcDriver, err := iofs.New(sqlFS, "sql/migrations")
	if err != nil {
		return false, fmt.Errorf("failed to create migration source instance: %w", err)
	}

	_, err = srcDriver.Next(uint(v))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No next migration, no migration needed
			return false, nil
		}
		return false, fmt.Errorf("failed to get next migration: %w", err)
	}

	return true, nil
}

// buildMigrate creates a migration instance
// Note: The returned migrate.Migrate owns the db connection and will close it when Close() is called
func (m *Manager) buildMigrate() (*migrate.Migrate, error) {
	db := stdlib.OpenDBFromPool(m.pool)

	dbDriver, err := pgxmig.WithInstance(db, &pgxmig.Config{})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create migration db instance: %w", err)
	}

	srcDriver, err := iofs.New(sqlFS, "sql/migrations")
	if err != nil {
		dbDriver.Close()
		db.Close()
		return nil, fmt.Errorf("failed to create migration source instance: %w", err)
	}

	mig, err := migrate.NewWithInstance("iofs", srcDriver, "postgres", dbDriver)
	if err != nil {
		dbDriver.Close()
		db.Close()
		return nil, err
	}

	return mig, nil
}

// Migrate performs any necessary migrations
func (m *Manager) Migrate() error {
	mig, err := m.buildMigrate()
	if err != nil {
		return fmt.Errorf("failed to build migration instance: %w", err)
	}
	defer mig.Close()

	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// initTables ensures that the required tables exist in the database
func (m *Manager) initTables(ctx context.Context) error {
	// Check if migrations are needed
	needed, err := m.MigrationNeeded()
	if err != nil {
		return fmt.Errorf("failed to check if migrations are needed: %w", err)
	}

	// Apply migrations if needed
	if needed {
		if err := m.Migrate(); err != nil {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
	}

	// Verify database connection is still healthy
	if err := m.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database connection unhealthy after migrations: %w", err)
	}

	return nil
}

// Close closes all resources
func (m *Manager) Close() {
	if m.pool != nil {
		m.pool.Close()
	}
}
