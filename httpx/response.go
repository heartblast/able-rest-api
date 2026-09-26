// Package httpx는 공통 JSON 응답을 제공한다.
package httpx

import (
	"net/http"

	"github.com/heartblast/able-rest-api/internal/platform/http/response"
)

// Success는 공통 성공 응답을 기록한다.
func Success[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
	response.WriteSuccess(w, r, status, data)
}

// Error는 공통 오류 응답을 기록한다.
func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	response.WriteError(w, r, status, code, message)
}
