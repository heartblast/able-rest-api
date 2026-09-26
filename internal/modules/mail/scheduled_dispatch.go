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

// MessageDispatch는 저장된 개별 메일을 기존 서비스로 전달한다.
type MessageDispatch struct{ service *MailService }

// NewMessageDispatch는 개별 예약 메일 발송 명령을 생성한다.
func NewMessageDispatch(service *MailService) *MessageDispatch {
	return &MessageDispatch{service: service}
}

// Dispatch는 저장된 메일의 입력값을 검증하고 발송한다.
func (d *MessageDispatch) Dispatch(ctx context.Context, message MailMessage) error {
	_, err := d.service.SendMail(ctx, message)
	return err
}
