package mail

import (
	"github.com/go-chi/chi/v5"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/delivery/http/handler"
)

// Routes는 메일 모듈의 HTTP 경로를 등록한다.
func Routes(svc *service.MailService) func(chi.Router) {
	return func(api chi.Router) {
		h := handler.NewMailHandler(svc)
		api.Route("/mail", func(mail chi.Router) {
			mail.Post("/send", h.Send)
		})
	}
}
