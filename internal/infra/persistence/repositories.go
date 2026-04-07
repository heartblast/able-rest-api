package persistence

import (
	"database/sql"
	"fmt"

	"able-rest-api/internal/domain/repository"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/db/hsqldb"
	"able-rest-api/internal/infra/db/mysql"
	"able-rest-api/internal/infra/db/oracle"
	"able-rest-api/internal/infra/db/postgres"
)

// Repositories는 애플리케이션이 사용하는 저장소 묶음이다.
type Repositories struct {
	UserRepository repository.UserRepository
}

// NewRepositories는 벤더별 저장소 구현체를 조립한다.
func NewRepositories(vendor config.DBVendor, db *sql.DB) (*Repositories, error) {
	repos := &Repositories{}

	switch vendor {
	case config.DBVendorPostgres:
		repos.UserRepository = postgres.NewUserRepository(db)
	case config.DBVendorMySQL:
		repos.UserRepository = mysql.NewUserRepository(db)
	case config.DBVendorOracle:
		repos.UserRepository = oracle.NewUserRepository(db)
	case config.DBVendorHSQLDB:
		repos.UserRepository = hsqldb.NewUserRepository(db)
	default:
		return nil, fmt.Errorf("지원하지 않는 repository vendor: %s", vendor)
	}

	return repos, nil
}
