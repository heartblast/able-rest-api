package router

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"able-rest-api/docs"
	custommw "able-rest-api/internal/delivery/http/middleware"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/platform/http/health"
	"able-rest-api/internal/platform/logger"
)

// Module은 보호된 API 경로에 업무 라우트를 등록한다.
type Module func(chi.Router)

// New는 공통 HTTP 라우터에 업무 모듈을 등록한다.
func New(cfg *config.Config, log logger.Logger, db *sql.DB, modules ...Module) http.Handler {
	r := chi.NewRouter()

	r.Use(custommw.RequestID)
	r.Use(custommw.Logging(log))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(custommw.JSONContentType)
	r.Use(custommw.LimitJSONBody(cfg.Security.MaxRequestBodyBytes))

	healthHandler := health.NewHealthHandler(db)

	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.With(custommw.APIKey(cfg.App.Env, cfg.Security.APIKeyEnv)).Get("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(docs.OpenAPIJSON)
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(custommw.APIKey(cfg.App.Env, cfg.Security.APIKeyEnv))
		for _, register := range modules {
			register(api)
		}
	})

	if cfg.Swagger.Enabled {
		r.With(custommw.APIKey(cfg.App.Env, cfg.Security.APIKeyEnv)).Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/openapi.json"),
		))
	}

	return r
}
