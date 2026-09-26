package mail

import "context"

// MailSender는 메일 발송 구현체가 따라야 할 계약이다.
type MailSender interface {
	Send(ctx context.Context, message MailMessage) error
}
