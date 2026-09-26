package mail

import (
	"context"
	"errors"
	"strings"
	"testing"

	"able-rest-api/internal/domain/model"
)

func TestMailHeaderAndRecipientValidation(t *testing.T) {
	sender := &captureMailSender{}
	svc := NewMailService(true, sender)
	cases := []struct {
		name    string
		message model.MailMessage
	}{
		{"subject injection", model.MailMessage{To: []string{"a@example.com"}, Subject: "Hi\r\nBcc: victim@example.com", Body: "hello"}},
		{"display name recipient", model.MailMessage{To: []string{"Eve <eve@example.com>"}, Subject: "Hi", Body: "hello"}},
		{"attachment content type injection", model.MailMessage{To: []string{"a@example.com"}, Subject: "Hi", Body: "hello", Attachments: []model.MailAttachment{{Filename: "x.txt", ContentType: "text/plain\r\nX-Test: injected", ContentBase64: "YQ=="}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.SendMail(context.Background(), tc.message); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	addresses := make([]string, 101)
	for i := range addresses {
		addresses[i] = strings.Repeat("a", i+1) + "@example.com"
	}
	if _, err := svc.SendMail(context.Background(), model.MailMessage{To: addresses, Subject: "Hi", Body: "hello"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("too many recipients: %v", err)
	}
	if len(sender.messages) != 0 {
		t.Fatal("invalid mail was sent")
	}
}

type captureMailSender struct{ messages []model.MailMessage }

func (s *captureMailSender) Send(_ context.Context, msg model.MailMessage) error {
	s.messages = append(s.messages, msg)
	return nil
}
