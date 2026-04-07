package model

// MailMessage는 메일 발송 요청 도메인 모델이다.
type MailMessage struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	IsHTML      bool
	Attachments []MailAttachment
}

// MailAttachment는 메일 첨부파일 도메인 모델이다.
type MailAttachment struct {
	Filename      string
	ContentType   string
	ContentBase64 string
	Content       []byte
}
