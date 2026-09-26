package scheduler

import "context"

// JobRunner는 스케줄러가 실행할 작업 계약이다.
type JobRunner interface {
	Definition() ScheduledJob
	Run(ctx context.Context) error
}
