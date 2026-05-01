package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cu-points/backend/internal/auth"
)

func newTestHandler(repo auth.UserRepository) *auth.Handler {
	jwtMgr := auth.NewJWTManager(
		"test-secret-minimum-32-characters-long",
		15*time.Minute,
		168*time.Hour,
	)
	svc := auth.NewService(repo, jwtMgr)
	return auth.NewHandler(svc)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestHandler_Login_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)
	repo := &mockRepo{
		user: &auth.UserRecord{
			ID:           "u-1",
			Email:        "a@cu.ru",
			PasswordHash: string(hash),
			Role:         "student",
		},
	}
	h := newTestHandler(repo)

	body, _ := json.Marshal(map[string]string{"email": "a@cu.ru", "password": "pass123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
		t.Error("expected both tokens to be non-empty")
	}
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader("{bad json"))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Login_MissingFields(t *testing.T) {
	h := newTestHandler(&mockRepo{})

	body, _ := json.Marshal(map[string]string{"email": "", "password": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Login_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	repo := &mockRepo{
		user: &auth.UserRecord{
			ID:           "u-1",
			Email:        "a@cu.ru",
			PasswordHash: string(hash),
			Role:         "student",
		},
	}
	h := newTestHandler(repo)

	body, _ := json.Marshal(map[string]string{"email": "a@cu.ru", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestHandler_Refresh_Success(t *testing.T) {
	jwtMgr := auth.NewJWTManager(
		"test-secret-minimum-32-characters-long",
		15*time.Minute,
		168*time.Hour,
	)
	user := &auth.UserRecord{ID: "u-2", Role: "student"}
	repo := &mockRepo{user: user}
	svc := auth.NewService(repo, jwtMgr)
	h := auth.NewHandler(svc)

	refreshToken, _ := jwtMgr.GenerateRefreshToken(user.ID)

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Refresh_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
		strings.NewReader("{bad json"))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Refresh_MissingToken(t *testing.T) {
	h := newTestHandler(&mockRepo{})

	body, _ := json.Marshal(map[string]string{"refresh_token": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Refresh_InvalidToken(t *testing.T) {
	h := newTestHandler(&mockRepo{})

	body, _ := json.Marshal(map[string]string{"refresh_token": "bad.token.here"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
