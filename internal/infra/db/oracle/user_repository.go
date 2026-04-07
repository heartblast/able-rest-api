package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/domain/repository"
)

// UserRepository는 Oracle 확장 포인트용 skeleton 구현체다.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository는 Oracle 사용자 저장소를 생성한다.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID는 향후 Oracle SQL 구현으로 대체한다.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	_ = ctx
	_ = id
	return nil, fmt.Errorf("oracle user repository TODO: 별도 드라이버와 RETURNING/sequence 전략을 연결하세요")
}

// List는 향후 Oracle SQL 구현으로 대체한다.
func (r *UserRepository) List(ctx context.Context, filter repository.UserFilter) ([]model.User, error) {
	_ = ctx
	_ = filter
	return nil, fmt.Errorf("oracle user repository TODO: OFFSET/FETCH 및 스캔 로직을 구현하세요")
}

// Create는 향후 Oracle SQL 구현으로 대체한다.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	_ = ctx
	_ = user
	return fmt.Errorf("oracle user repository TODO: sequence 또는 identity 전략과 드라이버 RETURNING 지원을 구현하세요")
}
