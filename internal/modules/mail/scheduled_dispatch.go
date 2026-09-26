package mail

import "context"

// ScheduledDispatch는 예약 작업에서 메일 서비스를 호출한다.
type ScheduledDispatch struct {
	service *MailService
	message MailMessage
}

// NewScheduledDispatch는 예약 메일 명령을 생성한다.
func NewScheduledDispatch(service *MailService, message MailMessage) *ScheduledDispatch {
	return &ScheduledDispatch{service: service, message: message}
}

// Dispatch는 즉시 발송과 동일한 검증 및 발송 경로를 사용한다.
func (d *ScheduledDispatch) Dispatch(ctx context.Context) error {
	_, err := d.service.SendMail(ctx, d.message)
	return err
}
