package handler

import (
	"database/sql"
	"net/http"

	"my-api/internal/delivery/http/dto"
)

// HealthHandler는 상태 점검 핸들러다.
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler는 HealthHandler를 생성한다.
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health godoc
// @Summary Liveness 확인
// @Description 프로세스가 살아있는지 확인한다
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, r, http.StatusOK, dto.HealthData{Status: "ok"})
}

// Ready godoc
// @Summary Readiness 확인
// @Description DB 연결 가능 여부를 확인한다
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /ready [get]
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "DB_NOT_READY", "DB 연결이 준비되지 않았습니다")
		return
	}

	writeSuccess(w, r, http.StatusOK, dto.HealthData{Status: "ready"})
}
