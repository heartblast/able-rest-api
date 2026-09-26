package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"path/filepath"
	"strings"

	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/domain/port"
)

var (
	ErrDisabled = errors.New("disabled")
)

const (
	maxAttachmentCount     = 5
	maxAttachmentSizeBytes = 5 * 1024 * 1024
	maxTotalAttachmentSize = 20 * 1024 * 1024
	maxRecipients          = 100
)

// MailService는 메일 발송 유스케이스를 담당한다.
type MailService struct {
	enabled bool
	sender  port.MailSender
}

// NewMailService는 MailService를 생성한다.
func NewMailService(enabled bool, sender port.MailSender) *MailService {
	return &MailService{
		enabled: enabled,
		sender:  sender,
	}
}

// SendMail은 메일 발송 요청을 검증하고 전송한다.
func (s *MailService) SendMail(ctx context.Context, message model.MailMessage) (int, error) {
	if !s.enabled || s.sender == nil {
		return 0, ErrDisabled
	}

	to, err := normalizeAddresses(message.To, true)
	if err != nil {
		return 0, err
	}
	cc, err := normalizeAddresses(message.CC, false)
	if err != nil {
		return 0, err
	}
	bcc, err := normalizeAddresses(message.BCC, false)
	if err != nil {
		return 0, err
	}
	if len(message.To)+len(message.CC)+len(message.BCC) > maxRecipients {
		return 0, fmt.Errorf("%w: 수신자는 최대 %d명까지 허용됩니다", ErrInvalidInput, maxRecipients)
	}

	subject := strings.TrimSpace(message.Subject)
	body := strings.TrimSpace(message.Body)
	if subject == "" {
		return 0, fmt.Errorf("%w: subject는 필수입니다", ErrInvalidInput)
	}
	if hasHeaderControl(subject) {
		return 0, fmt.Errorf("%w: subject에 제어 문자를 사용할 수 없습니다", ErrInvalidInput)
	}
	if body == "" {
		return 0, fmt.Errorf("%w: body는 필수입니다", ErrInvalidInput)
	}

	attachments, err := normalizeAttachments(message.Attachments)
	if err != nil {
		return 0, err
	}

	message.To = to
	message.CC = excludeDuplicates(cc, to)
	message.BCC = excludeDuplicates(bcc, append(append([]string{}, to...), cc...))
	message.Subject = subject
	message.Body = body
	message.Attachments = attachments

	if err := s.sender.Send(ctx, message); err != nil {
		return 0, fmt.Errorf("메일 발송 실패: %w", err)
	}

	return len(message.To) + len(message.CC) + len(message.BCC), nil
}

func normalizeAddresses(values []string, required bool) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		address, err := mail.ParseAddress(normalized)
		if err != nil || address.Address != normalized || hasHeaderControl(normalized) {
			return nil, fmt.Errorf("%w: 이메일 형식이 올바르지 않습니다", ErrInvalidInput)
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	if required && len(result) == 0 {
		return nil, fmt.Errorf("%w: to는 하나 이상의 이메일이 필요합니다", ErrInvalidInput)
	}

	return result, nil
}

func excludeDuplicates(values []string, existing []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(existing))
	for _, value := range existing {
		seen[value] = struct{}{}
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func normalizeAttachments(values []model.MailAttachment) ([]model.MailAttachment, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) > maxAttachmentCount {
		return nil, fmt.Errorf("%w: attachments는 최대 %d개까지 허용됩니다", ErrInvalidInput, maxAttachmentCount)
	}

	result := make([]model.MailAttachment, 0, len(values))
	totalSize := 0

	for _, value := range values {
		filename := strings.TrimSpace(value.Filename)
		if filename == "" {
			return nil, fmt.Errorf("%w: attachment.filename은 필수입니다", ErrInvalidInput)
		}
		if hasHeaderControl(filename) {
			return nil, fmt.Errorf("%w: attachment.filename이 올바르지 않습니다", ErrInvalidInput)
		}
		filename = filepath.Base(filename)
		if filename == "." || filename == "" {
			return nil, fmt.Errorf("%w: attachment.filename이 올바르지 않습니다", ErrInvalidInput)
		}

		contentBase64 := strings.TrimSpace(value.ContentBase64)
		if contentBase64 == "" {
			return nil, fmt.Errorf("%w: attachment.content_base64는 필수입니다", ErrInvalidInput)
		}

		decoded, err := base64.StdEncoding.DecodeString(contentBase64)
		if err != nil {
			return nil, fmt.Errorf("%w: attachment.content_base64가 올바르지 않습니다", ErrInvalidInput)
		}
		if len(decoded) == 0 {
			return nil, fmt.Errorf("%w: attachment.content_base64가 비어 있습니다", ErrInvalidInput)
		}
		if len(decoded) > maxAttachmentSizeBytes {
			return nil, fmt.Errorf("%w: attachment는 최대 %d bytes까지 허용됩니다", ErrInvalidInput, maxAttachmentSizeBytes)
		}

		totalSize += len(decoded)
		if totalSize > maxTotalAttachmentSize {
			return nil, fmt.Errorf("%w: attachments 총합은 최대 %d bytes까지 허용됩니다", ErrInvalidInput, maxTotalAttachmentSize)
		}

		contentType := strings.TrimSpace(value.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil || len(params) != 0 || hasHeaderControl(contentType) {
			return nil, fmt.Errorf("%w: attachment.content_type이 올바르지 않습니다", ErrInvalidInput)
		}
		contentType = mediaType

		result = append(result, model.MailAttachment{
			Filename:    filename,
			ContentType: contentType,
			Content:     decoded,
		})
	}

	return result, nil
}

func hasHeaderControl(value string) bool {
	for _, ch := range value {
		if ch < 32 || ch == 127 {
			return true
		}
	}
	return false
}
