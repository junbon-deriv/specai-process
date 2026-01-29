package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/deriv/arcade/internal/trading"
)

// TradingHandler handles trading-related API endpoints
type TradingHandler struct {
	tradingService *trading.Service
}

// NewTradingHandler creates a new trading handler
func NewTradingHandler(tradingService *trading.Service) *TradingHandler {
	return &TradingHandler{
		tradingService: tradingService,
	}
}

// SwipeGet handles GET /swipe
func (h *TradingHandler) SwipeGet(w http.ResponseWriter, r *http.Request) {
	seriesType := r.URL.Query().Get("series_type")

	if seriesType == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "series_type is required")
		return
	}

	response, err := h.tradingService.GeneratePreview(r.Context(), seriesType)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, response)
}

// SwipeBuy handles POST /swipe/buy
func (h *TradingHandler) SwipeBuy(w http.ResponseWriter, r *http.Request) {
	var req trading.SwipeBuyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	response, err := h.tradingService.ExecuteTrade(r.Context(), req)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, response)
}

// SwipeList handles GET /swipe/contracts
func (h *TradingHandler) SwipeList(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "account_id is required")
		return
	}

	// Optional contract_id parameter for single contract lookup
	var contractID *int64
	if cidStr := r.URL.Query().Get("contract_id"); cidStr != "" {
		var cid int64
		if _, err := fmt.Sscanf(cidStr, "%d", &cid); err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid contract_id format")
			return
		}
		contractID = &cid
	}

	// Optional series_type filter for list
	var seriesType *string
	if st := r.URL.Query().Get("series_type"); st != "" {
		seriesType = &st
	}

	response, err := h.tradingService.ListContracts(r.Context(), accountID, contractID, seriesType)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, response)
}

// SwipeInstruments handles GET /swipe/instruments
func (h *TradingHandler) SwipeInstruments(w http.ResponseWriter, r *http.Request) {
	response, err := h.tradingService.ListInstruments(r.Context())
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, response)
}
