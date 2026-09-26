package user

import (
	"github.com/go-chi/chi/v5"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/delivery/http/handler"
)

// Routes는 사용자 모듈의 HTTP 경로를 등록한다.
func Routes(svc *service.UserService) func(chi.Router) {
	return func(api chi.Router) {
		h := handler.NewUserHandler(svc)
		api.Route("/users", func(users chi.Router) {
			users.Get("/", h.ListUsers)
			users.Post("/", h.CreateUser)
			users.Get("/{id}", h.GetUser)
		})
	}
}
