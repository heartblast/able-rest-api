package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"
)

// PostgresLock은 advisory lock 기반 락 구현이다.
type PostgresLock struct {
	db *sql.DB
}

// NewPostgresLock는 PostgresLock을 생성한다.
func NewPostgresLock(db *sql.DB) *PostgresLock {
	return &PostgresLock{db: db}
}

// Acquire는 작업 단위 advisory lock을 획득한다.
func (l *PostgresLock) Acquire(ctx context.Context, jobID string) (bool, func() error, error) {
	if l.db == nil {
		return false, nil, errors.New("postgres lock requires db")
	}

	key := int64(crc32.ChecksumIEEE([]byte(jobID)))
	var acquired bool
	if err := l.db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, nil, fmt.Errorf("pg_try_advisory_lock failed: %w", err)
	}

	release := func() error {
		var released bool
		if err := l.db.QueryRowContext(context.Background(), "SELECT pg_advisory_unlock($1)", key).Scan(&released); err != nil {
			return fmt.Errorf("pg_advisory_unlock failed: %w", err)
		}
		return nil
	}

	return acquired, release, nil
}
