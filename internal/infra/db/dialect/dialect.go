package dialect

import (
	"fmt"

	"github.com/heartblast/able-rest-api/internal/infra/config"
)

// Dialect는 DBMS별 SQL 차이를 격리한다.
type Dialect interface {
	Name() string
	Placeholder(index int) string
	Pagination(limit, offset int) string
	CreateUserQuery() string
	GetUserByIDQuery() string
	ListUsersQuery(limit, offset int) string
}

// New는 벤더에 맞는 Dialect를 반환한다.
func New(vendor config.DBVendor) (Dialect, error) {
	switch vendor {
	case config.DBVendorPostgres:
		return postgresDialect{}, nil
	case config.DBVendorMySQL:
		return mysqlDialect{}, nil
	case config.DBVendorOracle:
		return oracleDialect{}, nil
	case config.DBVendorHSQLDB:
		return hsqldbDialect{}, nil
	default:
		return nil, fmt.Errorf("지원하지 않는 dialect vendor: %s", vendor)
	}
}
