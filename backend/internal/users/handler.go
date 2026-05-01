// Package users handles student profile and transaction history endpoints.
package users

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cu-points/backend/internal/middleware"
	"github.com/cu-points/backend/pkg/response"
)

// Handler holds HTTP handler methods for the users domain.
type Handler struct {
	service *Service
}

// NewHandler creates a new users Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// transactionsResponse is the JSON body returned by GET /me/transactions.
type transactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
}

// Me handles GET /api/v1/me.
// Returns the authenticated student's profile and current balance.
// Requires role=student (enforced by the router's RequireRole middleware).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		slog.Error("handler.Me", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, profile)
}

// Transactions handles GET /api/v1/me/transactions.
// Returns paginated transaction history for the authenticated student.
// Query params: limit (default 20, max 100), offset (default 0).
// Response: { "transactions": [...], "total": N }
func (h *Handler) Transactions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	txs, total, err := h.service.GetTransactions(r.Context(), userID, limit, offset)
	if err != nil {
		slog.Error("handler.Transactions", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	// Return an empty array rather than null when there are no transactions.
	if txs == nil {
		txs = []Transaction{}
	}
	response.JSON(w, http.StatusOK, transactionsResponse{
		Transactions: txs,
		Total:        total,
	})
}
