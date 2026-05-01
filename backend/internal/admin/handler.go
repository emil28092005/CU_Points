// Package admin handles administration endpoints: granting points, viewing stats.
package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cu-points/backend/pkg/response"
)

// Handler holds HTTP handler methods for the admin domain.
type Handler struct {
	service *Service
}

// NewHandler creates a new admin Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// grantRequest is the expected JSON body for POST /api/v1/admin/points/grant.
type grantRequest struct {
	UserID      string `json:"user_id"`
	Amount      int    `json:"amount"`
	Description string `json:"description"`
}

// transactionsResponse is the JSON body returned by GET /api/v1/admin/transactions.
type transactionsResponse struct {
	Transactions []AdminTransaction `json:"transactions"`
	Total        int                `json:"total"`
}

// GrantPoints handles POST /api/v1/admin/points/grant.
// Accepts {user_id, amount, description}; credits the student's balance.
// Requires role=admin.
func (h *Handler) GrantPoints(w http.ResponseWriter, r *http.Request) {
	var req grantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UserID == "" {
		response.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Amount <= 0 {
		response.Error(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	if err := h.service.GrantPoints(r.Context(), req.UserID, req.Amount, req.Description); err != nil {
		slog.Error("handler.GrantPoints", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListTransactions handles GET /api/v1/admin/transactions.
// Returns all transactions in the system (paginated), newest first.
// Query params: limit (default 50, max 200), offset (default 0).
// Response: { "transactions": [...], "total": N }
func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	txs, total, err := h.service.ListTransactions(r.Context(), limit, offset)
	if err != nil {
		slog.Error("handler.ListTransactions", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if txs == nil {
		txs = []AdminTransaction{}
	}
	response.JSON(w, http.StatusOK, transactionsResponse{
		Transactions: txs,
		Total:        total,
	})
}

// ListUsers handles GET /api/v1/admin/users.
// Returns all students with their current balances.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	students, err := h.service.ListStudents(r.Context())
	if err != nil {
		slog.Error("handler.ListUsers", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if students == nil {
		students = []Student{}
	}
	response.JSON(w, http.StatusOK, students)
}

// Stats handles GET /api/v1/admin/stats.
// Returns aggregated statistics: total students, points issued/spent, active partners.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		slog.Error("handler.Stats", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, stats)
}
