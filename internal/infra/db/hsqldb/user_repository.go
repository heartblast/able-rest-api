package hsqldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/heartblast/able-rest-api/internal/modules/user"
)

// UserRepository는 HSQLDB 확장 포인트용 skeleton 구현체다.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository는 HSQLDB 사용자 저장소를 생성한다.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID는 향후 HSQLDB SQL 구현으로 대체한다.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	_ = ctx
	_ = id
	return nil, fmt.Errorf("hsqldb user repository TODO: 선택한 JDBC 브리지/드라이버에 맞춘 조회 로직을 구현하세요")
}

// List는 향후 HSQLDB SQL 구현으로 대체한다.
func (r *UserRepository) List(ctx context.Context, filter user.UserFilter) ([]user.User, error) {
	_ = ctx
	_ = filter
	return nil, fmt.Errorf("hsqldb user repository TODO: LIMIT/OFFSET 및 스캔 로직을 구현하세요")
}

// Create는 향후 HSQLDB SQL 구현으로 대체한다.
func (r *UserRepository) Create(ctx context.Context, user *user.User) error {
	_ = ctx
	_ = user
	return fmt.Errorf("hsqldb user repository TODO: identity 전략과 생성 후 재조회 로직을 구현하세요")
}
