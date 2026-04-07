package model

import "time"

// ScheduledJob는 등록된 스케줄 작업 정의다.
type ScheduledJob struct {
	ID              string
	Name            string
	Enabled         bool
	Interval        time.Duration
	NextRunAt       time.Time
	AllowConcurrent bool
	MaxRetries      int
}
