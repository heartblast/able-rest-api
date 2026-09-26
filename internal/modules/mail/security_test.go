package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMailHeaderAndRecipientValidation(t *testing.T) {
	sender := &captureMailSender{}
	svc := NewMailService(true, sender)
	cases := []struct {
		name    string
		message MailMessage
	}{
		{"subject injection", MailMessage{To: []string{"a@example.com"}, Subject: "Hi\r\nBcc: victim@example.com", Body: "hello"}},
		{"display name recipient", MailMessage{To: []string{"Eve <eve@example.com>"}, Subject: "Hi", Body: "hello"}},
		{"attachment content type injection", MailMessage{To: []string{"a@example.com"}, Subject: "Hi", Body: "hello", Attachments: []MailAttachment{{Filename: "x.txt", ContentType: "text/plain\r\nX-Test: injected", ContentBase64: "YQ=="}}}},
		{"message id injection", MailMessage{MessageID: "<ok@example.com>\r\nBcc: victim@example.com", To: []string{"a@example.com"}, Subject: "Hi", Body: "hello"}},
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
	if _, err := svc.SendMail(context.Background(), MailMessage{To: addresses, Subject: "Hi", Body: "hello"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("too many recipients: %v", err)
	}
	if len(sender.messages) != 0 {
		t.Fatal("invalid mail was sent")
	}
}

type captureMailSender struct{ messages []MailMessage }

func (s *captureMailSender) Send(_ context.Context, msg MailMessage) error {
	s.messages = append(s.messages, msg)
	return nil
}
