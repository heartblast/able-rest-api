package router

import (
	"database/sql"
	"net/http"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/persistence"
	mailmodule "able-rest-api/internal/modules/mail"
	usermodule "able-rest-api/internal/modules/user"
	"able-rest-api/internal/platform/logger"
)

// newTestRouter는 운영 환경과 동일한 모듈 구성을 테스트에 적용한다.
func newTestRouter(cfg *config.Config, log logger.Logger, db *sql.DB, repos *persistence.Repositories, mailService *service.MailService) http.Handler {
	return New(cfg, log, db, usermodule.Routes(service.NewUserService(repos.UserRepository)), mailmodule.Routes(mailService))
}
