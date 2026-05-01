package points_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cu-points/backend/internal/middleware"
	"github.com/cu-points/backend/internal/points"
)

// injectUserID puts a user_id into the request context the same way middleware.Auth does.
func injectUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(middleware.ContextWithUserID(r.Context(), userID))
}

func newHandlerWithService(repo points.Repository, cache points.CacheClient) *points.Handler {
	svc := points.NewService(repo, cache, testSecret)
	return points.NewHandler(svc)
}

// ─── GenerateQR ───────────────────────────────────────────────────────────────

func TestGenerateQR_Success(t *testing.T) {
	h := newHandlerWithService(&mockRepo{}, &mockCache{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/qr", nil)
	req = injectUserID(req, "user-1")
	w := httptest.NewRecorder()

	h.GenerateQR(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Data.Token == "" {
		t.Error("expected non-empty token in response")
	}
}

// ─── Spend ───────────────────────────────────────────────────────────────────

func TestSpend_Success(t *testing.T) {
	svc := points.NewService(
		&mockRepo{balance: 500, newBalance: 400},
		&mockCache{},
		testSecret,
	)
	// Generate a valid token first.
	token, _ := svc.GenerateQRToken(context.Background(), "student-1")

	h := points.NewHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"qr_token": token,
		"amount":   100,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSpend_InvalidJSON(t *testing.T) {
	h := newHandlerWithService(&mockRepo{balance: 500}, &mockCache{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend",
		strings.NewReader("{bad json"))
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSpend_MissingQRToken(t *testing.T) {
	h := newHandlerWithService(&mockRepo{balance: 500}, &mockCache{})

	body, _ := json.Marshal(map[string]interface{}{"amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSpend_ZeroAmount(t *testing.T) {
	h := newHandlerWithService(&mockRepo{balance: 500}, &mockCache{})

	body, _ := json.Marshal(map[string]interface{}{"qr_token": "sometoken", "amount": 0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSpend_InvalidToken_Returns401(t *testing.T) {
	h := newHandlerWithService(&mockRepo{balance: 500}, &mockCache{})

	body, _ := json.Marshal(map[string]interface{}{"qr_token": "bad.token.here", "amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSpend_AlreadyUsedToken_Returns409(t *testing.T) {
	svc := points.NewService(
		&mockRepo{balance: 500},
		&mockCache{used: true},
		testSecret,
	)
	token, _ := svc.GenerateQRToken(context.Background(), "student-1")
	h := points.NewHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{"qr_token": token, "amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestSpend_InternalError_Returns500(t *testing.T) {
	// Trigger the default error path by making the cache return a generic error.
	svc := points.NewService(
		&mockRepo{balance: 500},
		&mockCache{isErr: errors.New("redis timeout")},
		testSecret,
	)
	token, _ := svc.GenerateQRToken(context.Background(), "student-1")
	// Rebuild the service with the broken cache for the actual spend call.
	svc2 := points.NewService(
		&mockRepo{balance: 500},
		&mockCache{isErr: errors.New("redis timeout")},
		testSecret,
	)
	h := points.NewHandler(svc2)

	body, _ := json.Marshal(map[string]interface{}{"qr_token": token, "amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSpend_InsufficientBalance_Returns422(t *testing.T) {
	svc := points.NewService(
		&mockRepo{balance: 10},
		&mockCache{},
		testSecret,
	)
	token, _ := svc.GenerateQRToken(context.Background(), "student-1")
	h := points.NewHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{"qr_token": token, "amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/partner/spend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, "partner-1")
	w := httptest.NewRecorder()

	h.Spend(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}
