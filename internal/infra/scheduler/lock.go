package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/modules/scheduler"
)

type noopLock struct{}

func (noopLock) Acquire(_ context.Context, _ string) (bool, func() error, error) {
	return true, func() error { return nil }, nil
}

// NewLock은 설정에 맞는 락 구현을 생성한다.
func NewLock(cfg config.SchedulerConfig, vendor config.DBVendor, db *sql.DB) (scheduler.SchedulerLock, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.LockProvider)) {
	case "", "none":
		return noopLock{}, nil
	case "postgres":
		if vendor != config.DBVendorPostgres {
			return nil, fmt.Errorf("postgres lock provider is not supported for vendor: %s", vendor)
		}
		return NewPostgresLock(db), nil
	default:
		return nil, fmt.Errorf("unsupported scheduler lock provider: %s", cfg.LockProvider)
	}
}
