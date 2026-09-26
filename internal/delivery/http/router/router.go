package router

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"able-rest-api/docs"
	"able-rest-api/internal/app/service"
	"able-rest-api/internal/delivery/http/handler"
	custommw "able-rest-api/internal/delivery/http/middleware"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/persistence"
	"able-rest-api/internal/platform/logger"
)

// New는 HTTP 라우터를 생성한다.
func New(cfg *config.Config, log logger.Logger, db *sql.DB, repos *persistence.Repositories, mailService *service.MailService) http.Handler {
	r := chi.NewRouter()

	r.Use(custommw.RequestID)
	r.Use(custommw.Logging(log))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(custommw.JSONContentType)

	healthHandler := handler.NewHealthHandler(db)
	userHandler := handler.NewUserHandler(service.NewUserService(repos.UserRepository))
	mailHandler := handler.NewMailHandler(mailService)

	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Get("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(docs.OpenAPIJSON)
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/users", func(users chi.Router) {
			users.Get("/", userHandler.ListUsers)
			users.Post("/", userHandler.CreateUser)
			users.Get("/{id}", userHandler.GetUser)
		})
		api.Route("/mail", func(mail chi.Router) {
			mail.Post("/send", mailHandler.Send)
		})
	})

	if cfg.Swagger.Enabled {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/openapi.json"),
		))
	}

	return r
}
