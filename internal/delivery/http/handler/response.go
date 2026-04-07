package handler

import (
	"encoding/json"
	"net/http"

	"my-api/internal/delivery/http/dto"
	"my-api/internal/delivery/http/middleware"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSuccess[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
	writeJSON(w, status, dto.SuccessResponse[T]{
		Success:   true,
		RequestID: middleware.RequestIDFromContext(r.Context()),
		Data:      data,
	})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, dto.ErrorResponse{
		Success:   false,
		RequestID: middleware.RequestIDFromContext(r.Context()),
		Error: dto.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
