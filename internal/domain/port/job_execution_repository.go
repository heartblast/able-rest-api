package port

import (
	"context"

	"able-rest-api/internal/domain/model"
)

// JobExecutionRepository는 작업 실행 이력 저장 계약이다.
type JobExecutionRepository interface {
	Create(ctx context.Context, execution *model.JobExecution) error
	Update(ctx context.Context, execution *model.JobExecution) error
}
