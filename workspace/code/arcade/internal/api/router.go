package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/trading"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the HTTP router
func NewRouter(accountService *accounts.Service, tradingService *trading.Service) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Create handlers
	accountsHandler := NewAccountsHandler(accountService)
	tradingHandler := NewTradingHandler(tradingService)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Accounts endpoints
	r.Post("/accounts", accountsHandler.CreateAccount)
	r.Get("/accounts/{account_id}", accountsHandler.GetAccount)
	r.Post("/accounts/{account_id}/deposits", accountsHandler.Deposit)
	r.Post("/accounts/{account_id}/withdrawals", accountsHandler.Withdraw)

	// Trading endpoints
	r.Get("/swipe", tradingHandler.SwipeGet)
	r.Post("/swipe/buy", tradingHandler.SwipeBuy)
	r.Get("/swipe/contracts", tradingHandler.SwipeList)
	r.Get("/swipe/instruments", tradingHandler.SwipeInstruments)

	slog.Info("Router initialized with all endpoints")
	return r
}
