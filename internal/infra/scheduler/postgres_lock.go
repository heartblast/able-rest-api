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
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, fmt.Errorf("postgres lock transaction failed: %w", err)
	}
	var acquired bool
	if err := tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", key).Scan(&acquired); err != nil {
		_ = tx.Rollback()
		return false, nil, fmt.Errorf("pg_try_advisory_xact_lock failed: %w", err)
	}
	if !acquired {
		_ = tx.Rollback()
		return false, nil, nil
	}

	release := func() error {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("postgres lock rollback failed: %w", err)
		}
		return nil
	}

	return acquired, release, nil
}
