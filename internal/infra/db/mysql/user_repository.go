package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/db/dialect"
	"able-rest-api/internal/modules/user"
)

// UserRepository는 MySQL 사용자 저장소 구현체다.
type UserRepository struct {
	db      *sql.DB
	dialect dialect.Dialect
}

// NewUserRepository는 MySQL 사용자 저장소를 생성한다.
func NewUserRepository(db *sql.DB) *UserRepository {
	d, _ := dialect.New(config.DBVendorMySQL)
	return &UserRepository{
		db:      db,
		dialect: d,
	}
}

// GetByID는 사용자 단건을 조회한다.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	row := r.db.QueryRowContext(ctx, r.dialect.GetUserByIDQuery(), id)

	var user user.User
	if err := row.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("mysql 사용자 조회 실패: %w", err)
	}

	return &user, nil
}

// List는 사용자 목록을 조회한다.
func (r *UserRepository) List(ctx context.Context, filter user.UserFilter) ([]user.User, error) {
	rows, err := r.db.QueryContext(ctx, r.dialect.ListUsersQuery(filter.Limit, filter.Offset))
	if err != nil {
		return nil, fmt.Errorf("mysql 사용자 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	users := make([]user.User, 0)
	for rows.Next() {
		var user user.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("mysql 사용자 목록 scan 실패: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql 사용자 목록 rows 에러: %w", err)
	}

	return users, nil
}

// Create는 사용자를 생성한다.
func (r *UserRepository) Create(ctx context.Context, user *user.User) error {
	result, err := r.db.ExecContext(ctx, r.dialect.CreateUserQuery(), user.Name, user.Email)
	if err != nil {
		return fmt.Errorf("mysql 사용자 생성 실패: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql 사용자 ID 조회 실패: %w", err)
	}

	user.ID = id
	created, err := r.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("mysql 사용자 생성 후 재조회 실패: %w", err)
	}
	if created == nil {
		return errors.New("mysql 사용자 생성 후 데이터가 존재하지 않습니다")
	}

	user.CreatedAt = created.CreatedAt
	user.UpdatedAt = created.UpdatedAt
	return nil
}
