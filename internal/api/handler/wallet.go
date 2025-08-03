// internal/api/handler/wallet.go
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"finflow-wallet/internal/service"
	"finflow-wallet/internal/util" // For custom errors
)

// WalletHandler handles HTTP requests related to wallet operations.
type WalletHandler struct {
	service service.WalletService
	logger  *slog.Logger
}

// NewWalletHandler creates a new WalletHandler.
func NewWalletHandler(svc service.WalletService, logger *slog.Logger) *WalletHandler {
	return &WalletHandler{
		service: svc,
		logger:  logger,
	}
}

// Helper function to send JSON responses.
func (h *WalletHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("Failed to marshal JSON response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

// Helper function to send error responses.
func (h *WalletHandler) respondWithError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case util.IsError(err, util.ErrInvalidInput):
		statusCode = http.StatusBadRequest
		message = err.Error() // Use the error message directly for invalid input
	case util.IsError(err, util.ErrNotFound), util.IsError(err, util.ErrWalletNotFound), util.IsError(err, util.ErrUserNotFound):
		statusCode = http.StatusNotFound
		message = "Resource not found"
	case util.IsError(err, util.ErrInsufficientFunds):
		statusCode = http.StatusPaymentRequired // 402 Payment Required
		message = "Insufficient funds"
	case util.IsError(err, util.ErrSameWalletTransfer):
		statusCode = http.StatusBadRequest
		message = "Cannot transfer to the same wallet"
	// Add more specific error mappings as needed
	default:
		h.logger.Error("Unhandled service error", "error", err)
	}

	h.respondWithJSON(w, statusCode, map[string]string{"error": message})
}

// DepositRequest represents the request body for deposit.
type DepositRequest struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// Deposit handles the deposit money request.
// POST /wallets/{walletID}/deposit
func (h *WalletHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := strconv.ParseInt(walletIDStr, 10, 64)
	if err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	// Basic validation
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}
	if req.Currency == "" {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	wallet, transaction, err := h.service.Deposit(r.Context(), walletID, req.Amount, req.Currency)
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "Deposit successful",
		"wallet_id":      wallet.ID,
		"new_balance":    wallet.Balance,
		"transaction_id": transaction.ID,
	})
}

// WithdrawRequest represents the request body for withdraw.
type WithdrawRequest struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// Withdraw handles the withdraw money request.
// POST /wallets/{walletID}/withdraw
func (h *WalletHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := strconv.ParseInt(walletIDStr, 10, 64)
	if err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	// Basic validation
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}
	if req.Currency == "" {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	wallet, transaction, err := h.service.Withdraw(r.Context(), walletID, req.Amount, req.Currency)
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "Withdrawal successful",
		"wallet_id":      wallet.ID,
		"new_balance":    wallet.Balance,
		"transaction_id": transaction.ID,
	})
}

// TransferRequest represents the request body for transfer.
type TransferRequest struct {
	FromWalletID int64           `json:"from_wallet_id"`
	ToWalletID   int64           `json:"to_wallet_id"`
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
}

// Transfer handles the transfer money request.
// POST /transfers
func (h *WalletHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	// Basic validation
	if req.FromWalletID == 0 || req.ToWalletID == 0 {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}
	if req.Currency == "" {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	fromWallet, toWallet, transaction, err := h.service.Transfer(r.Context(), req.FromWalletID, req.ToWalletID, req.Amount, req.Currency)
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":                 "Transfer successful",
		"transaction_id":          transaction.ID,
		"from_wallet_new_balance": fromWallet.Balance,
		"to_wallet_new_balance":   toWallet.Balance,
	})
}

// GetWalletBalance handles the get wallet balance request.
// GET /wallets/{walletID}/balance
func (h *WalletHandler) GetWalletBalance(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := strconv.ParseInt(walletIDStr, 10, 64)
	if err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	wallet, err := h.service.GetBalance(r.Context(), walletID)
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"wallet_id": wallet.ID,
		"balance":   wallet.Balance,
		"currency":  wallet.Currency,
	})
}

// GetTransactionHistory handles the get transaction history request.
// GET /wallets/{walletID}/transactions
func (h *WalletHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := strconv.ParseInt(walletIDStr, 10, 64)
	if err != nil {
		h.respondWithError(w, util.ErrInvalidInput)
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // Default limit
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0 // Default offset
	}

	transactions, err := h.service.GetTransactionHistory(r.Context(), walletID, limit, offset)
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	// For simplicity, total count is not returned here, but can be added if needed
	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"data":   transactions,
		"limit":  limit,
		"offset": offset,
	})
}
