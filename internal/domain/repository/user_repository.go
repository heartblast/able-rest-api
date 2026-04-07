package repository

import (
	"context"

	"my-api/internal/domain/model"
)

// UserFilter는 사용자 조회 조건을 표현한다.
type UserFilter struct {
	Limit  int
	Offset int
}

// UserRepository는 사용자 저장소 계약만 정의한다.
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
	List(ctx context.Context, filter UserFilter) ([]model.User, error)
	Create(ctx context.Context, user *model.User) error
}
