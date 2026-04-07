package model

import "time"

type JobExecutionStatus string

const (
	JobExecutionStatusRunning JobExecutionStatus = "running"
	JobExecutionStatusSuccess JobExecutionStatus = "success"
	JobExecutionStatusFailed  JobExecutionStatus = "failed"
	JobExecutionStatusSkipped JobExecutionStatus = "skipped"
)

// JobExecution은 스케줄 작업 실행 이력이다.
type JobExecution struct {
	ID           string
	JobID        string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Status       JobExecutionStatus
	ErrorMessage string
	RunnerID     string
}
