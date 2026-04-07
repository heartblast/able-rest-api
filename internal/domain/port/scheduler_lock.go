package port

import "context"

// SchedulerLock은 중복 실행 방지용 락 계약이다.
type SchedulerLock interface {
	Acquire(ctx context.Context, jobID string) (acquired bool, release func() error, err error)
}
