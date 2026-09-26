package scheduler

import "context"

// JobExecutionRepository는 작업 실행 이력 저장 계약이다.
type JobExecutionRepository interface {
	Create(ctx context.Context, execution *JobExecution) error
	Update(ctx context.Context, execution *JobExecution) error
}
