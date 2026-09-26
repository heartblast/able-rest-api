package response

import (
	"encoding/json"
	"net/http"

	"github.com/heartblast/able-rest-api/internal/delivery/http/middleware"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteSuccess는 공통 성공 응답을 기록한다.
func WriteSuccess[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
	writeJSON(w, status, SuccessResponse[T]{Success: true, RequestID: middleware.RequestIDFromContext(r.Context()), Data: data})
}

// WriteError는 공통 오류 응답을 기록한다.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Success: false, RequestID: middleware.RequestIDFromContext(r.Context()), Error: ErrorDetail{Code: code, Message: message}})
}
