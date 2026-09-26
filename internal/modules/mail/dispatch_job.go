package mail

import (
	"context"
	"time"

	"able-rest-api/internal/domain/model"
)

// MailDispatchJob는 향후 예약 메일 발송 작업을 연결하기 위한 기본 작업이다.
type MailDispatchJob struct{}

// NewMailDispatchJob는 MailDispatchJob를 생성한다.
func NewMailDispatchJob() *MailDispatchJob {
	return &MailDispatchJob{}
}

// Definition은 작업 정의를 반환한다.
func (j *MailDispatchJob) Definition() model.ScheduledJob {
	return model.ScheduledJob{
		ID:              "mail-dispatch",
		Name:            "mail-dispatch",
		Enabled:         true,
		Interval:        time.Minute,
		AllowConcurrent: false,
		MaxRetries:      0,
	}
}

// Run은 실제 예약 메일 발송 로직이 연결될 위치다.
func (j *MailDispatchJob) Run(_ context.Context) error {
	return nil
}
