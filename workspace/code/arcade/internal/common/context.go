package common

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// contextKey is used for context values
type contextKey string

const (
	// TxContextKey is the key for database transaction in context
	TxContextKey contextKey = "db_transaction"
)

// GetTx extracts database transaction from context
func GetTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(TxContextKey).(pgx.Tx)
	return tx, ok
}

// WithTx adds database transaction to context
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TxContextKey, tx)
}
