package health

import (
	"database/sql"
	"net/http"

	"github.com/heartblast/able-rest-api/internal/platform/http/response"
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
	response.WriteSuccess(w, r, http.StatusOK, response.HealthData{Status: "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		response.WriteError(w, r, http.StatusServiceUnavailable, "DB_NOT_READY", "DB 연결이 준비되지 않았습니다")
		return
	}

	response.WriteSuccess(w, r, http.StatusOK, response.HealthData{Status: "ready"})
}
