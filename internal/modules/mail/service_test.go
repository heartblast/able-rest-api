package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubMailSender struct {
	message MailMessage
	err     error
}

func (s *stubMailSender) Send(_ context.Context, message MailMessage) error {
	s.message = message
	return s.err
}

func TestMailServiceSendMail(t *testing.T) {
	sender := &stubMailSender{}
	svc := NewMailService(true, sender)

	acceptedRecipients, err := svc.SendMail(context.Background(), MailMessage{
		To:      []string{"USER@example.com", "user@example.com"},
		CC:      []string{"cc@example.com", "user@example.com"},
		BCC:     []string{"bcc@example.com", "cc@example.com"},
		Subject: " hello ",
		Body:    " world ",
		Attachments: []MailAttachment{
			{
				Filename:      " guide.txt ",
				ContentType:   "text/plain",
				ContentBase64: "SGVsbG8gd29ybGQ=",
			},
		},
	})
	if err != nil {
		t.Fatalf("SendMail returned error: %v", err)
	}
	if acceptedRecipients != 3 {
		t.Fatalf("expected 3 accepted recipients, got %d", acceptedRecipients)
	}

	if len(sender.message.To) != 1 {
		t.Fatalf("expected deduplicated to recipients, got %v", sender.message.To)
	}
	if len(sender.message.CC) != 1 || sender.message.CC[0] != "cc@example.com" {
		t.Fatalf("expected cc recipients to be deduplicated, got %v", sender.message.CC)
	}
	if len(sender.message.BCC) != 1 || sender.message.BCC[0] != "bcc@example.com" {
		t.Fatalf("expected bcc recipients to be deduplicated, got %v", sender.message.BCC)
	}
	if sender.message.Subject != "hello" || sender.message.Body != "world" {
		t.Fatalf("expected subject/body to be trimmed, got %#v", sender.message)
	}
	if len(sender.message.Attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(sender.message.Attachments))
	}
	if sender.message.Attachments[0].Filename != "guide.txt" {
		t.Fatalf("expected sanitized filename, got %q", sender.message.Attachments[0].Filename)
	}
	if string(sender.message.Attachments[0].Content) != "Hello world" {
		t.Fatalf("expected decoded attachment content, got %q", string(sender.message.Attachments[0].Content))
	}
}

func TestMailServiceValidation(t *testing.T) {
	sender := &stubMailSender{}
	svc := NewMailService(true, sender)

	_, err := svc.SendMail(context.Background(), MailMessage{
		To:      []string{"not-an-email"},
		Subject: "test",
		Body:    "body",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestMailServiceAttachmentValidation(t *testing.T) {
	sender := &stubMailSender{}
	svc := NewMailService(true, sender)

	_, err := svc.SendMail(context.Background(), MailMessage{
		To:      []string{"user@example.com"},
		Subject: "test",
		Body:    "body",
		Attachments: []MailAttachment{
			{
				Filename:      "bad.txt",
				ContentBase64: "%%%invalid%%%",
			},
		},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestMailServiceAttachmentCountValidation(t *testing.T) {
	sender := &stubMailSender{}
	svc := NewMailService(true, sender)

	attachments := make([]MailAttachment, 0, maxAttachmentCount+1)
	for i := 0; i < maxAttachmentCount+1; i++ {
		attachments = append(attachments, MailAttachment{
			Filename:      "file.txt",
			ContentBase64: "SGVsbG8=",
		})
	}

	_, err := svc.SendMail(context.Background(), MailMessage{
		To:          []string{"user@example.com"},
		Subject:     "test",
		Body:        "body",
		Attachments: attachments,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "최대") {
		t.Fatalf("expected max attachment count message, got %v", err)
	}
}

func TestMailServiceDisabled(t *testing.T) {
	svc := NewMailService(false, nil)
	_, err := svc.SendMail(context.Background(), MailMessage{
		To:      []string{"user@example.com"},
		Subject: "test",
		Body:    "body",
	})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}
