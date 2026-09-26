package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	"github.com/heartblast/able-rest-api/internal/modules/mail"
)

var _ mail.MailSender = (*SMTPSender)(nil)

// SMTPSender는 SMTP 기반 메일 발송 구현체다.
type SMTPSender struct {
	cfg config.SMTPConfig
}

// NewSMTPSender는 SMTPSender를 생성한다.
func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// Send는 SMTP 서버를 통해 메일을 발송한다.
func (s *SMTPSender) Send(ctx context.Context, message mail.MailMessage) error {
	address := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	dialer := &net.Dialer{Timeout: s.cfg.Timeout}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("SMTP 연결 실패: %w", err)
	}

	deadline := time.Now().Add(s.cfg.Timeout)
	_ = conn.SetDeadline(deadline)

	if s.cfg.TLS {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: s.cfg.Host,
			MinVersion: tls.VersionTLS12,
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return fmt.Errorf("SMTP TLS handshake 실패: %w", err)
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("SMTP 클라이언트 생성 실패: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP hello 실패: %w", err)
	}

	if s.cfg.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP 서버가 STARTTLS를 지원하지 않습니다")
		}
		if err := client.StartTLS(&tls.Config{
			ServerName: s.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}); err != nil {
			return fmt.Errorf("SMTP STARTTLS 실패: %w", err)
		}
	}

	if err := s.authenticate(client); err != nil {
		return err
	}

	if err := client.Mail(s.cfg.FromAddress); err != nil {
		return fmt.Errorf("발신자 설정 실패: %w", err)
	}

	for _, recipient := range allRecipients(message) {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("수신자 설정 실패: %w", err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("메일 데이터 시작 실패: %w", err)
	}

	payload, err := buildMessage(s.cfg, message)
	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("메일 본문 생성 실패: %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return fmt.Errorf("메일 본문 전송 실패: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("메일 본문 종료 실패: %w", err)
	}

	// DATA 완료 응답 뒤 종료 오류는 이미 수락된 메일의 재발송 사유가 아니다.
	_ = client.Quit()

	return nil
}

func (s *SMTPSender) authenticate(client *smtp.Client) error {
	switch strings.ToLower(strings.TrimSpace(s.cfg.AuthType)) {
	case "", "plain":
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 인증 실패: %w", err)
		}
	case "none":
	default:
		return fmt.Errorf("지원하지 않는 SMTP 인증 방식입니다")
	}
	return nil
}

func buildMessage(cfg config.SMTPConfig, message mail.MailMessage) ([]byte, error) {
	if len(message.Attachments) == 0 {
		return []byte(buildSimpleMessage(cfg, message)), nil
	}
	return buildMultipartMessage(cfg, message)
}

func buildSimpleMessage(cfg config.SMTPConfig, message mail.MailMessage) string {
	headers := []string{
		fmt.Sprintf("From: %s", formatFrom(cfg.FromName, cfg.FromAddress)),
		fmt.Sprintf("To: %s", strings.Join(message.To, ", ")),
		fmt.Sprintf("Subject: %s", message.Subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: %s", bodyContentType(message.IsHTML)),
	}

	if len(message.CC) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(message.CC, ", ")))
	}
	if message.MessageID != "" {
		headers = append(headers, "Message-ID: "+message.MessageID)
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + message.Body
}

func buildMultipartMessage(cfg config.SMTPConfig, message mail.MailMessage) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	headers := []string{
		fmt.Sprintf("From: %s", formatFrom(cfg.FromName, cfg.FromAddress)),
		fmt.Sprintf("To: %s", strings.Join(message.To, ", ")),
		fmt.Sprintf("Subject: %s", message.Subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/mixed; boundary=%q", writer.Boundary()),
	}
	if len(message.CC) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(message.CC, ", ")))
	}
	if message.MessageID != "" {
		headers = append(headers, "Message-ID: "+message.MessageID)
	}
	if _, err := buf.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n"); err != nil {
		return nil, err
	}

	bodyHeader := textproto.MIMEHeader{}
	bodyHeader.Set("Content-Type", bodyContentType(message.IsHTML))
	bodyHeader.Set("Content-Transfer-Encoding", "8bit")

	bodyPart, err := writer.CreatePart(bodyHeader)
	if err != nil {
		return nil, err
	}
	if _, err := bodyPart.Write([]byte(message.Body)); err != nil {
		return nil, err
	}

	for _, attachment := range message.Attachments {
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", fmt.Sprintf("%s; name=%q", attachment.ContentType, sanitizeHeaderValue(attachment.Filename)))
		partHeader.Set("Content-Transfer-Encoding", "base64")
		partHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizeHeaderValue(attachment.Filename)))

		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, err
		}
		if err := writeBase64WithCRLF(part, attachment.Content); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func bodyContentType(isHTML bool) string {
	if isHTML {
		return "text/html; charset=UTF-8"
	}
	return "text/plain; charset=UTF-8"
}

func formatFrom(name, address string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return address
	}
	return fmt.Sprintf("\"%s\" <%s>", sanitizeHeaderValue(name), address)
}

func sanitizeHeaderValue(value string) string {
	value = strings.ReplaceAll(value, "\"", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return strings.TrimSpace(value)
}

func allRecipients(message mail.MailMessage) []string {
	recipients := make([]string, 0, len(message.To)+len(message.CC)+len(message.BCC))
	recipients = append(recipients, message.To...)
	recipients = append(recipients, message.CC...)
	recipients = append(recipients, message.BCC...)
	return recipients
}

func writeBase64WithCRLF(w io.Writer, content []byte) error {
	encoded := base64.StdEncoding.EncodeToString(content)
	for len(encoded) > 76 {
		if _, err := w.Write([]byte(encoded[:76] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		if _, err := w.Write([]byte(encoded + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}
