package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/deriv/arcade/internal/common"
)

// ErrorResponse represents the standard error response
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, code, message string) {
	var resp ErrorResponse
	resp.Error.Code = code
	resp.Error.Message = message

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode error response", "error", err)
	}
}

// HandleServiceError converts service errors to HTTP responses
func HandleServiceError(w http.ResponseWriter, err error) {
	// Check if it's an API error
	if apiErr, ok := err.(*common.APIError); ok {
		status := getHTTPStatusForCode(apiErr.Code)
		WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	// Check for common errors
	switch err {
	case common.ErrAccountNotFound:
		WriteError(w, http.StatusNotFound, common.ErrCodeAccountNotFound, "Account not found")
	case common.ErrInsufficientBalance:
		WriteError(w, http.StatusBadRequest, common.ErrCodeInsufficientBalance, "Insufficient balance")
	case common.ErrInvalidAmount:
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidAmount, "Invalid amount")
	case common.ErrInvalidCurrency:
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidCurrency, "Invalid currency code")
	case common.ErrInvalidSeriesType:
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidSeriesType, "Invalid series type")
	case common.ErrInvalidSentiment:
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidSentiment, "Invalid sentiment")
	case common.ErrInvalidQuote:
		WriteError(w, http.StatusBadRequest, common.ErrCodeInvalidQuote, "Invalid quote")
	default:
		slog.Error("Internal server error", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
	}
}

// getHTTPStatusForCode maps error codes to HTTP status codes
func getHTTPStatusForCode(code string) int {
	switch code {
	case common.ErrCodeAccountNotFound:
		return http.StatusNotFound
	case common.ErrCodeInvalidCurrency,
		common.ErrCodeInvalidAmount,
		common.ErrCodeInvalidStake,
		common.ErrCodeInsufficientBalance,
		common.ErrCodeInvalidSeriesType,
		common.ErrCodeInvalidSentiment,
		common.ErrCodeInvalidQuote:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
