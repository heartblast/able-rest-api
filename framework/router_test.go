package framework_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/heartblast/able-rest-api/framework"
	"github.com/heartblast/able-rest-api/httpx"
)

func TestPublicRouter(t *testing.T) {
	r := framework.New()
	if err := r.SetOpenAPIJSON([]byte(`{"openapi":"3.0.3","paths":{"/api/v1/hello":{}}}`)); err != nil {
		t.Fatal(err)
	}
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Consumer-Middleware", "yes")
			next.ServeHTTP(w, req)
		})
	})
	r.Register(func(api chi.Router) {
		api.Get("/hello", func(w http.ResponseWriter, req *http.Request) {
			httpx.Success(w, req, http.StatusOK, map[string]string{"message": "hello"})
		})
	})
	h := r.Handler()
	for _, path := range []string{"/api/v1/hello", "/openapi.json"} {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusOK || res.Header().Get("X-Consumer-Middleware") != "yes" || !json.Valid(res.Body.Bytes()) {
			t.Fatalf("%s: status=%d headers=%v body=%s", path, res.Code, res.Header(), res.Body.String())
		}
		if path == "/api/v1/hello" {
			var payload struct {
				Success bool `json:"success"`
				Data    struct {
					Message string `json:"message"`
				} `json:"data"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil || !payload.Success || payload.Data.Message != "hello" {
				t.Fatalf("unexpected hello response: %s (%v)", res.Body.String(), err)
			}
		}
	}
}
