package api

import (
	"encoding/json"
	"net/http"

	"github.com/deriv/arcade/internal/accounts"
	"github.com/deriv/arcade/internal/common"
	"github.com/go-chi/chi/v5"
)

// AccountsHandler handles account-related API endpoints
type AccountsHandler struct {
	accountService *accounts.Service
}

// NewAccountsHandler creates a new accounts handler
func NewAccountsHandler(accountService *accounts.Service) *AccountsHandler {
	return &AccountsHandler{
		accountService: accountService,
	}
}

// CreateAccount handles POST /accounts
func (h *AccountsHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req accounts.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), req)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	response := map[string]string{
		"account_id": account.AccountID,
	}
	WriteJSON(w, http.StatusCreated, response)
}

// GetAccount handles GET /accounts/{account_id}
func (h *AccountsHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")

	account, err := h.accountService.GetAccount(r.Context(), accountID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	response := map[string]interface{}{
		"account_id":  account.AccountID,
		"external_id": account.ExternalID,
		"balance":     common.FormatAmount(account.Balance),
		"currency":    account.Currency,
	}
	WriteJSON(w, http.StatusOK, response)
}

// Deposit handles POST /accounts/{account_id}/deposits
func (h *AccountsHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")

	var req accounts.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Parse amount
	amount, err := common.ParseAmount(req.Amount)
	if err != nil {
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidAmount, "Invalid amount format")
		return
	}

	// Process deposit (returns transaction and new balance from stored procedure)
	result, err := h.accountService.Deposit(r.Context(), accountID, amount, req.DepositID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	response := accounts.TransactionResponse{
		Balance:         common.FormatAmount(result.NewBalance),
		TransactionID:   result.Transaction.TransactionID,
		TransactionTime: result.Transaction.TransactionTime,
	}
	WriteJSON(w, http.StatusOK, response)
}

// Withdraw handles POST /accounts/{account_id}/withdrawals
func (h *AccountsHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")

	var req accounts.WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Parse amount
	amount, err := common.ParseAmount(req.Amount)
	if err != nil {
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidAmount, "Invalid amount format")
		return
	}

	// Process withdrawal (returns transaction and new balance from stored procedure)
	result, err := h.accountService.Withdraw(r.Context(), accountID, amount, req.WithdrawalID)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	response := accounts.TransactionResponse{
		Balance:         common.FormatAmount(result.NewBalance),
		TransactionID:   result.Transaction.TransactionID,
		TransactionTime: result.Transaction.TransactionTime,
	}
	WriteJSON(w, http.StatusOK, response)
}
