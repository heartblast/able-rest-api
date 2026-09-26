package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	mailmodule "github.com/heartblast/able-rest-api/internal/modules/mail"
)

func TestSecurityRequests(t *testing.T) {
	key := strings.Repeat("k", 40)
	t.Setenv("TEST_APP_API_KEY", key)
	repo := &contractRepo{}
	sender := &contractSender{}
	h := newTestRouter(&config.Config{App: config.AppConfig{Env: "production"}, Security: config.SecurityConfig{APIKeyEnv: "TEST_APP_API_KEY", MaxRequestBodyBytes: 128}}, quietLogger{}, nil, repo, mailmodule.NewMailService(true, sender))
	cases := []struct {
		name, path, body, mediaType, token string
		want                               int
	}{
		{"unauthorized user list", "/api/v1/users", "", "", "", 401},
		{"unauthorized openapi", "/openapi.json", "", "", "", 401},
		{"wrong key", "/api/v1/users", "", "", "wrong", 401},
		{"valid key", "/api/v1/users", "", "", key, 200},
		{"oversized body", "/api/v1/users", strings.Repeat("x", 129), "application/json", key, 413},
		{"media type suffix", "/api/v1/users", `{}`, "text/plain; note=application/json", key, 415},
		{"missing media type", "/api/v1/users", `{}`, "", key, 415},
		{"trailing JSON", "/api/v1/users", `{"name":"A","email":"a@b.co"} {}`, "application/json", key, 400},
		{"unknown field", "/api/v1/users", `{"name":"A","email":"a@b.co","admin":true}`, "application/json", key, 400},
		{"header injection", "/api/v1/mail/send", `{"to":["a@b.co"],"subject":"hi\r\nBcc: victim@example.com","body":"hello"}`, "application/json", key, 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method := http.MethodGet
			if tc.body != "" {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, tc.path, strings.NewReader(tc.body))
			if tc.mediaType != "" {
				req.Header.Set("Content-Type", tc.mediaType)
			}
			if tc.token != "" {
				req.Header.Set("X-API-Key", tc.token)
			}
			response := httptest.NewRecorder()
			h.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", response.Code, tc.want, response.Body.String())
			}
		})
	}
	if repo.createCalls != 0 || len(sender.calls) != 0 {
		t.Fatal("rejected requests caused side effects")
	}
	t.Run("chunked oversized body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"name":"`+strings.Repeat("x", 129)+`","email":"a@example.com"}`))
		req.ContentLength = -1
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", key)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, req)
		if response.Code != 413 {
			t.Fatalf("status = %d, want 413", response.Code)
		}
	})
	t.Run("duplicate content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{}`))
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Content-Type", "text/plain")
		req.Header.Set("X-API-Key", key)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, req)
		if response.Code != 415 {
			t.Fatalf("status = %d, want 415", response.Code)
		}
	})
	t.Run("duplicate API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Add("X-API-Key", key)
		req.Header.Add("X-API-Key", key)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, req)
		if response.Code != 401 {
			t.Fatalf("status = %d, want 401", response.Code)
		}
	})
}

func TestSecurityMissingKeyFailsClosed(t *testing.T) {
	t.Setenv("TEST_MISSING_KEY", "")
	h := newTestRouter(&config.Config{App: config.AppConfig{Env: "production"}, Security: config.SecurityConfig{APIKeyEnv: "TEST_MISSING_KEY"}}, quietLogger{}, nil, nil, nil)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/users", nil))
	if response.Code != 401 {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}
