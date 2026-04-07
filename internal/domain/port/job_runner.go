package port

import (
	"context"

	"my-api/internal/domain/model"
)

// JobRunner는 스케줄러가 실행할 작업 계약이다.
type JobRunner interface {
	Definition() model.ScheduledJob
	Run(ctx context.Context) error
}
