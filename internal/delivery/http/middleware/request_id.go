package middleware

import (
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestID는 chi request id를 내부 컨텍스트 키에도 저장한다.
func RequestID(next http.Handler) http.Handler {
	base := chimiddleware.RequestID
	return base(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := chimiddleware.GetReqID(r.Context())
		next.ServeHTTP(w, r.WithContext(WithRequestID(r.Context(), requestID)))
	}))
}
