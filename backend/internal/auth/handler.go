// Package auth handles user authentication: login and token refresh.
package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cu-points/backend/pkg/response"
)

// Handler holds HTTP handler methods for the auth domain.
// It only parses requests and writes responses — no business logic here.
type Handler struct {
	service *Service
}

// NewHandler creates a new auth Handler backed by the given service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// loginRequest is the expected JSON body for POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// refreshRequest is the expected JSON body for POST /api/v1/auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// accessTokenResponse is the JSON body returned by a successful refresh.
type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// Login handles POST /api/v1/auth/login.
// Accepts {"email": "...", "password": "..."}. Returns a token pair on success.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	pair, err := h.service.Login(r.Context(), LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		slog.Error("handler.Login", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, pair)
}

// Refresh handles POST /api/v1/auth/refresh.
// Accepts {"refresh_token": "..."}. Returns a new access token on success.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	accessToken, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		slog.Error("handler.Refresh", "err", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, accessTokenResponse{AccessToken: accessToken})
}
