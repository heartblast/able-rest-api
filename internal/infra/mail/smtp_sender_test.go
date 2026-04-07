package mail

import (
	"strings"
	"testing"

	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/infra/config"
)

func TestBuildMessageWithAttachment(t *testing.T) {
	payload, err := buildMessage(config.SMTPConfig{
		FromAddress: "no-reply@example.com",
		FromName:    "Mailer",
	}, model.MailMessage{
		To:      []string{"user@example.com"},
		Subject: "hello",
		Body:    "world",
		Attachments: []model.MailAttachment{
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
	}, model.MailMessage{
		To:      []string{"user@example.com"},
		Subject: "hello",
		Body:    "world",
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
}
