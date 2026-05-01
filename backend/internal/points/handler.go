// Package points handles earning and spending of loyalty points.
package points

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cu-points/backend/internal/middleware"
	"github.com/cu-points/backend/pkg/response"
)

// Handler holds HTTP handler methods for the points domain.
type Handler struct {
	service *Service
}

// NewHandler creates a new points Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// qrResponse is the JSON body returned by GenerateQR.
type qrResponse struct {
	Token string `json:"token"`
}

// spendRequest is the expected JSON body for POST /api/v1/partner/spend.
type spendRequest struct {
	QRToken string `json:"qr_token"`
	Amount  int    `json:"amount"`
}

// GenerateQR handles GET /api/v1/me/qr.
// Returns a one-time QR JWT token with 5-minute TTL for the authenticated student.
func (h *Handler) GenerateQR(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	token, err := h.service.GenerateQRToken(r.Context(), userID)
	if err != nil {
		slog.Error("handler.GenerateQR", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, qrResponse{Token: token})
}

// Spend handles POST /api/v1/partner/spend.
// Accepts {qr_token, amount}; debits student balance atomically.
// Requires role=partner (enforced by the router's RequireRole middleware).
func (h *Handler) Spend(w http.ResponseWriter, r *http.Request) {
	var req spendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.QRToken == "" {
		response.Error(w, http.StatusBadRequest, "qr_token is required")
		return
	}
	if req.Amount <= 0 {
		response.Error(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	partnerID := middleware.UserIDFromContext(r.Context())

	err := h.service.SpendPoints(r.Context(), SpendRequest{
		QRToken:   req.QRToken,
		Amount:    req.Amount,
		PartnerID: partnerID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidQRToken):
			response.Error(w, http.StatusUnauthorized, "invalid or expired QR token")
		case errors.Is(err, ErrQRAlreadyUsed):
			response.Error(w, http.StatusConflict, "QR token has already been used")
		case errors.Is(err, ErrInsufficientBalance):
			response.Error(w, http.StatusUnprocessableEntity, "insufficient balance")
		default:
			slog.Error("handler.Spend", "err", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
