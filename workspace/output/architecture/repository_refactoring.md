# Repository Refactoring - Dependency Injection Pattern

## Overview

This document outlines the refactoring plan to implement proper dependency injection for repositories, improving testability and separation of concerns.

## Current Architecture Issues

1. **Tight Coupling**: Each service (accounts, trading, series) creates its own repository instance
2. **Testing Difficulty**: Hard to mock repositories for unit tests
3. **Schema Management**: Migrations scattered, no clear ownership
4. **Shared Connection**: Multiple repository instances share the same pool without clear lifecycle

## Proposed Architecture

### Directory Structure

```
internal/
├── repository/
│   ├── sql/                    # SQL schema files
│   │   ├── schema/
│   │   │   ├── 000001_create_accounts.up.sql
│   │   │   ├── 000002_create_series_types.up.sql
│   │   │   └── ...
│   │   └── procedures/
│   │       ├── 000010_create_deposit_funds.up.sql
│   │       └── ...
│   ├── postgres/              # PostgreSQL implementations
│   │   ├── accounts.go        # Implements accounts.Repository interface
│   │   ├── series.go          # Implements series.Repository interface
│   │   ├── trading.go         # Implements trading.Repository interface
│   │   └── migrations.go      # Migration runner
│   └── manager.go             # Repository manager/factory
├── accounts/
│   ├── model.go               # Models (Account, Transaction, etc.)
│   ├── repository.go          # Repository interface definition
│   └── service.go             # Service (depends on Repository interface)
├── series/
│   ├── model.go               # Models (SeriesType, SeriesConfig, OHLC, etc.)
│   ├── repository.go          # Repository interface definition
│   ├── service.go             # Service (depends on Repository interface)
│   ├── generator.go           # Generator interface
│   └── gbm.go                 # GBM implementation
└── trading/
    ├── model.go               # Models (Contract, PriceSeries, etc.)
    ├── repository.go          # Repository interface definition
    └── service.go             # Service (depends on Repository interface)
```

**Key Principle**: Each service package owns its models and repository interface. The central repository package only contains concrete PostgreSQL implementations.

### Dependency Injection Flow

```go
// main.go
func main() {
    pool := createDatabasePool()
    
    // Create repository manager
    repoManager := repository.NewManager(pool)
    
    // Run migrations
    if err := repoManager.Migrate(); err != nil {
        log.Fatal(err)
    }
    
    // Create repositories
    accountRepo := repoManager.AccountsRepository()
    seriesRepo := repoManager.SeriesRepository()
    tradingRepo := repoManager.TradingRepository()
    
    // Inject into services
    accountService := accounts.NewService(accountRepo)
    seriesService := series.NewService(seriesRepo)
    tradingService := trading.NewService(tradingRepo, accountService, seriesService)
}
```

### Interface Definitions

```go
// accounts/repository.go
package accounts

type AccountRepository interface {
    GetAccount(ctx context.Context, accountID string) (*Account, error)
    GetAccountBalance(ctx context.Context, accountID string) (decimal.Decimal, error)
    // Stored procedure wrappers
    DepositFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*DepositResult, error)
    WithdrawFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*WithdrawalResult, error)
}

// series/repository.go
package series

type SeriesRepository interface {
    GetSeriesType(ctx context.Context, seriesType string) (*SeriesType, error)
    GetSeriesConfig(ctx context.Context, seriesType string) (*SeriesConfig, error)
    ListActiveSeries(ctx context.Context) ([]SeriesType, error)
}

// trading/repository.go
package trading

type TradingRepository interface {
    CreatePriceSeries(ctx context.Context, accountID, seriesType string, candles []series.OHLC, quoteValue string) (*PriceSeries, error)
    FindPriceSeries(ctx context.Context, accountID, seriesType, quoteValue string) (*PriceSeries, error)
    ListContracts(ctx context.Context, accountID string, seriesType *string) ([]Contract, error)
    // Stored procedure wrappers
    OpenTrade(ctx context.Context, accountID string, seriesID int64, sentiment string, buyPrice decimal.Decimal) (*OpenTradeResult, error)
    CloseTrade(ctx context.Context, accountID string, contractID int64, sellPrice decimal.Decimal, sellOHLCs []series.OHLC) (*CloseTradeResult, error)
}
```

### Repository Implementation

```go
// repository/postgres/accounts.go
package postgres

type AccountsRepository struct {
    pool *pgxpool.Pool
}

func NewAccountsRepository(pool *pgxpool.Pool) *AccountsRepository {
    return &AccountsRepository{pool: pool}
}

func (r *AccountsRepository) DepositFunds(ctx context.Context, accountID string, amount decimal.Decimal, idempotencyID uuid.UUID) (*accounts.DepositResult, error) {
    var txnID int64
    var newBalance decimal.Decimal
    var isDuplicate bool
    var txnTime time.Time

    err := r.pool.QueryRow(ctx, `
        SELECT transaction_id, new_balance, is_duplicate, transaction_time
        FROM deposit_funds($1, $2, $3)
    `, accountID, amount, idempotencyID).Scan(&txnID, &newBalance, &isDuplicate, &txnTime)
    
    if err != nil {
        return nil, mapPgError(err)
    }

    // Build result...
}
```

### Migration Management

```go
// repository/postgres/migrations.go
package postgres

type MigrationManager struct {
    pool *pgxpool.Pool
}

func NewMigrationManager(pool *pgxpool.Pool) *MigrationManager {
    return &MigrationManager{pool: pool}
}

func (m *MigrationManager) Migrate() error {
    // Use golang-migrate or custom migration logic
    // Read from internal/repository/sql/schema/
    // Apply in order
}
```

### Repository Manager

```go
// repository/manager.go
package repository

type Manager struct {
    pool            *pgxpool.Pool
    accountsRepo    accounts.AccountRepository
    seriesRepo      series.SeriesRepository
    tradingRepo     trading.TradingRepository
}

func NewManager(pool *pgxpool.Pool) *Manager {
    return &Manager{
        pool:         pool,
        accountsRepo: postgres.NewAccountsRepository(pool),
        seriesRepo:   postgres.NewSeriesRepository(pool),
        tradingRepo:  postgres.NewTradingRepository(pool),
    }
}

func (m *Manager) AccountsRepository() accounts.AccountRepository {
    return m.accountsRepo
}

func (m *Manager) SeriesRepository() series.SeriesRepository {
    return m.seriesRepo
}

func (m *Manager) TradingRepository() trading.TradingRepository {
    return m.tradingRepo
}

func (m *Manager) Migrate() error {
    migrator := postgres.NewMigrationManager(m.pool)
    return migrator.Migrate()
}
```

## Benefits

1. **Testability**: Services can be tested with mock repositories
2. **Single Source of Truth**: One repository instance per type
3. **Clear Ownership**: Repository package owns all data access
4. **Migration Management**: Centralized schema migration
5. **Flexibility**: Easy to swap implementations (PostgreSQL → MySQL, etc.)

## Migration Steps

1. Create `internal/repository` package structure
2. Move SQL files to `internal/repository/sql/`
3. Define repository interfaces in each service package
4. Implement PostgreSQL repositories in `repository/postgres/`
5. Create repository manager
6. Add migration runner
7. Update service constructors to accept repository interfaces
8. Update main.go to use dependency injection
9. Create mock repositories for tests

## Implementation Priority

| Priority | Task |
|----------|------|
| High | Create repository interfaces |
| High | Implement PostgreSQL repositories |
| High | Create repository manager |
| Medium | Move SQL files |
| Medium | Add migration runner |
| Low | Create mock repositories for tests |
