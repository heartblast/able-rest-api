package handler

import (
	"database/sql"
	"net/http"

	"able-rest-api/internal/delivery/http/dto"
)

// HealthHandler는 상태 점검 핸들러다.
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler는 HealthHandler를 생성한다.
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, r, http.StatusOK, dto.HealthData{Status: "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "DB_NOT_READY", "DB 연결이 준비되지 않았습니다")
		return
	}

	writeSuccess(w, r, http.StatusOK, dto.HealthData{Status: "ready"})
}
