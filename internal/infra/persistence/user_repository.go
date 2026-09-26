package persistence

import (
	"database/sql"
	"fmt"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	"github.com/heartblast/able-rest-api/internal/infra/db/hsqldb"
	"github.com/heartblast/able-rest-api/internal/infra/db/mysql"
	"github.com/heartblast/able-rest-api/internal/infra/db/oracle"
	"github.com/heartblast/able-rest-api/internal/infra/db/postgres"
	"github.com/heartblast/able-rest-api/internal/modules/user"
)

// NewUserRepository는 벤더별 사용자 저장소 구현체를 선택한다.
func NewUserRepository(vendor config.DBVendor, db *sql.DB) (user.UserRepository, error) {
	switch vendor {
	case config.DBVendorPostgres:
		return postgres.NewUserRepository(db), nil
	case config.DBVendorMySQL:
		return mysql.NewUserRepository(db), nil
	case config.DBVendorOracle:
		return oracle.NewUserRepository(db), nil
	case config.DBVendorHSQLDB:
		return hsqldb.NewUserRepository(db), nil
	default:
		return nil, fmt.Errorf("지원하지 않는 repository vendor: %s", vendor)
	}
}
