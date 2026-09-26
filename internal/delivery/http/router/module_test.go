package router

import (
	"database/sql"
	"net/http"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	mailmodule "github.com/heartblast/able-rest-api/internal/modules/mail"
	"github.com/heartblast/able-rest-api/internal/modules/user"
	usermodule "github.com/heartblast/able-rest-api/internal/modules/user"
	"github.com/heartblast/able-rest-api/internal/platform/logger"
)

// newTestRouter는 운영 환경과 동일한 모듈 구성을 테스트에 적용한다.
func newTestRouter(cfg *config.Config, log logger.Logger, db *sql.DB, repo user.UserRepository, mailService *mailmodule.MailService) http.Handler {
	return New(cfg, log, db, usermodule.Routes(usermodule.NewUserService(repo)), mailmodule.Routes(mailService))
}
