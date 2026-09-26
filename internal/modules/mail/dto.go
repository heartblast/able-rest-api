package mail

import "github.com/heartblast/able-rest-api/internal/platform/http/response"

// SendMailRequest는 메일 발송 요청 DTO다.
type SendMailRequest struct {
	To          []string                    `json:"to" example:"user@example.com"`
	CC          []string                    `json:"cc,omitempty" example:"team@example.com"`
	BCC         []string                    `json:"bcc,omitempty" example:"audit@example.com"`
	Subject     string                      `json:"subject" example:"Welcome"`
	Body        string                      `json:"body" example:"Hello, this is a test mail."`
	IsHTML      bool                        `json:"is_html" example:"false"`
	Attachments []SendMailAttachmentRequest `json:"attachments,omitempty"`
}

// SendMailAttachmentRequest는 메일 첨부파일 요청 DTO다.
type SendMailAttachmentRequest struct {
	Filename      string `json:"filename" example:"guide.txt"`
	ContentType   string `json:"content_type" example:"text/plain"`
	ContentBase64 string `json:"content_base64" example:"SGVsbG8gd29ybGQ="`
}

// SendMailResponseData는 메일 발송 성공 응답 DTO다.
type SendMailResponseData struct {
	AcceptedRecipients int `json:"accepted_recipients" example:"1"`
}

// SendMailResponse는 메일 발송 성공 응답 DTO다.
type SendMailResponse = response.SuccessResponse[SendMailResponseData]
