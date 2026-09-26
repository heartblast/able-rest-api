package scheduler

import (
	"context"
	"errors"
	"time"
)

// MailDispatchCommand는 예약 메일 발송을 수행하는 최소 계약이다.
type MailDispatchCommand interface {
	Dispatch(ctx context.Context) error
}

// MailDispatchJob는 설정된 주기에 예약 메일 명령을 실행한다.
type MailDispatchJob struct {
	command  MailDispatchCommand
	enabled  bool
	interval time.Duration
}

// NewMailDispatchJob는 MailDispatchJob를 생성한다.
func NewMailDispatchJob(command MailDispatchCommand, enabled bool, interval time.Duration) *MailDispatchJob {
	return &MailDispatchJob{command: command, enabled: enabled, interval: interval}
}

// Definition은 작업 정의를 반환한다.
func (j *MailDispatchJob) Definition() ScheduledJob {
	return ScheduledJob{
		ID:              "mail-dispatch",
		Name:            "mail-dispatch",
		Enabled:         j.enabled,
		Interval:        j.interval,
		AllowConcurrent: false,
		MaxRetries:      0,
	}
}

// Run은 메일 모듈의 명령에 발송을 위임한다.
func (j *MailDispatchJob) Run(ctx context.Context) error {
	if !j.enabled {
		return nil
	}
	if j.command == nil {
		return errors.New("예약 메일 명령이 설정되지 않았습니다")
	}
	return j.command.Dispatch(ctx)
}
