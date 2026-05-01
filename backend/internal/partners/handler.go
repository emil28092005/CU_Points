// Package partners handles the public partner listing endpoint.
package partners

import (
	"log/slog"
	"net/http"

	"github.com/cu-points/backend/pkg/response"
)

// Handler holds HTTP handler methods for the partners domain.
type Handler struct {
	service *Service
}

// NewHandler creates a new partners Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/partners.
// Returns all active partners. This endpoint is publicly accessible (no auth required).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	partners, err := h.service.ListActive(r.Context())
	if err != nil {
		slog.Error("handler.List", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	// Return an empty array rather than null when there are no partners.
	if partners == nil {
		partners = []Partner{}
	}
	response.JSON(w, http.StatusOK, partners)
}
