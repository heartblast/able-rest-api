package middleware

import (
	"net/http"
	"strings"
)

// JSONContentType는 JSON 기반 응답 헤더를 강제하고 JSON 요청을 검증한다.
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && !strings.Contains(strings.ToLower(contentType), "application/json") {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"Content-Type은 application/json 이어야 합니다"}}`))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
