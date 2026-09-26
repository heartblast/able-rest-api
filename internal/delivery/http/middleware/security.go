package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
)

// APIKey protects API endpoints when a key is configured, and fails closed outside local environments.
func APIKey(env, keyEnv string) func(http.Handler) http.Handler {
	if keyEnv == "" {
		keyEnv = "APP_API_KEY"
	}
	key := os.Getenv(keyEnv)
	required := key != "" || !isLocal(env)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if required {
				values := r.Header.Values("X-API-Key")
				if len(key) < 32 || len(values) != 1 || strings.Contains(values[0], ",") || subtle.ConstantTimeCompare(hash(values[0]), hash(key)) != 1 {
					writeSecurityError(w, http.StatusUnauthorized, "UNAUTHORIZED")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hash(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }

func isLocal(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}

func LimitJSONBody(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = 30 << 20
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				if r.ContentLength > maxBytes {
					writeSecurityError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE")
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("trailing JSON value")
	}
	return nil
}

func JSONErrorStatus(err error) int {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func ValidJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func writeSecurityError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": map[string]string{"code": code, "message": http.StatusText(status)}})
}
