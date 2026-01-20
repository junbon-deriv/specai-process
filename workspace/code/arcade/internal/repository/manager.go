package repository

import (
	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/repository/postgres"
	"github.com/deriv/arcade/internal/series"
	"github.com/deriv/arcade/internal/trading"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Manager manages repository instances
type Manager struct {
	pool         *pgxpool.Pool
	accountsRepo accounts.Repository
	seriesRepo   series.Repository
	tradingRepo  trading.Repository
}

// NewManager creates a new repository manager
func NewManager(pool *pgxpool.Pool) *Manager {
	return &Manager{
		pool:         pool,
		accountsRepo: postgres.NewAccountsRepository(pool),
		seriesRepo:   postgres.NewSeriesRepository(pool),
		tradingRepo:  postgres.NewTradingRepository(pool),
	}
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
