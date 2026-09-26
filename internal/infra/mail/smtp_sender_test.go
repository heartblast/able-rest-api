package mail

import (
	"strings"
	"testing"

	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/modules/mail"
)

func TestBuildMessageWithAttachment(t *testing.T) {
	payload, err := buildMessage(config.SMTPConfig{
		FromAddress: "no-reply@example.com",
		FromName:    "Mailer",
	}, mail.MailMessage{
		MessageID: "<abc123@scheduled.able-rest-api.invalid>",
		To:        []string{"user@example.com"},
		Subject:   "hello",
		Body:      "world",
		Attachments: []mail.MailAttachment{
			{
				Filename:    "guide.txt",
				ContentType: "text/plain",
				Content:     []byte("Hello world"),
			},
		},
	})
	if err != nil {
		t.Fatalf("buildMessage returned error: %v", err)
	}

	content := string(payload)
	if !strings.Contains(content, "Content-Type: multipart/mixed;") {
		t.Fatalf("expected multipart content type, got %s", content)
	}
	if !strings.Contains(content, "Message-ID: <abc123@scheduled.able-rest-api.invalid>\r\n") {
		t.Fatalf("multipart Message-ID header 누락: %s", content)
	}
	if !strings.Contains(content, `Content-Disposition: attachment; filename="guide.txt"`) {
		t.Fatalf("expected attachment disposition header, got %s", content)
	}
	if !strings.Contains(content, "Content-Transfer-Encoding: base64") {
		t.Fatalf("expected base64 header, got %s", content)
	}
	if !strings.Contains(content, "SGVsbG8gd29ybGQ=") {
		t.Fatalf("expected base64 attachment payload, got %s", content)
	}
}

func TestBuildMessageWithoutAttachment(t *testing.T) {
	payload, err := buildMessage(config.SMTPConfig{
		FromAddress: "no-reply@example.com",
	}, mail.MailMessage{
		MessageID: "<def456@scheduled.able-rest-api.invalid>",
		To:        []string{"user@example.com"},
		Subject:   "hello",
		Body:      "world",
	})
	if err != nil {
		t.Fatalf("buildMessage returned error: %v", err)
	}

	content := string(payload)
	if strings.Contains(content, "multipart/mixed") {
		t.Fatalf("expected simple message, got %s", content)
	}
	if !strings.Contains(content, "Content-Type: text/plain; charset=UTF-8") {
		t.Fatalf("expected text/plain header, got %s", content)
	}
	if !strings.Contains(content, "Message-ID: <def456@scheduled.able-rest-api.invalid>\r\n") {
		t.Fatalf("simple Message-ID header 누락: %s", content)
	}
}
