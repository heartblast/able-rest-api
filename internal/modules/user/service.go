package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/domain/repository"
)

var (
	// ErrInvalidInput은 입력값 검증 실패를 의미한다.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound는 조회 대상이 없음을 의미한다.
	ErrNotFound = errors.New("not found")
)

// UserService는 사용자 유스케이스를 담당한다.
type UserService struct {
	repo repository.UserRepository
}

// NewUserService는 UserService를 생성한다.
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// CreateUser는 사용자 생성 유스케이스를 처리한다.
func (s *UserService) CreateUser(ctx context.Context, name, email string) (*model.User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))

	if name == "" || email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: name/email 형식이 올바르지 않습니다", ErrInvalidInput)
	}

	user := &model.User{
		Name:  name,
		Email: email,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("사용자 생성 실패: %w", err)
	}

	return user, nil
}

// GetUser는 ID로 사용자를 조회한다.
func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id는 1 이상이어야 합니다", ErrInvalidInput)
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	return user, nil
}

// ListUsers는 사용자 목록을 조회한다.
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]model.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	users, err := s.repo.List(ctx, repository.UserFilter{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("사용자 목록 조회 실패: %w", err)
	}

	return users, nil
}
