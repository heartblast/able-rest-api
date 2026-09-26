package handler

import (
	"errors"
	"net/http"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/delivery/http/dto"
	"able-rest-api/internal/delivery/http/middleware"
	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/platform/http/response"
)

// MailHandler는 메일 발송 HTTP 요청을 처리한다.
type MailHandler struct {
	service *service.MailService
}

// NewMailHandler는 MailHandler를 생성한다.
func NewMailHandler(svc *service.MailService) *MailHandler {
	return &MailHandler{service: svc}
}

func (h *MailHandler) Send(w http.ResponseWriter, r *http.Request) {
	var req dto.SendMailRequest
	if err := middleware.DecodeJSON(r, &req); err != nil {
		if middleware.JSONErrorStatus(err) == http.StatusRequestEntityTooLarge {
			response.WriteError(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "요청 본문이 너무 큽니다")
			return
		}
		response.WriteError(w, r, http.StatusBadRequest, "INVALID_JSON", "JSON 본문이 올바르지 않습니다")
		return
	}

	message := model.MailMessage{
		To:      req.To,
		CC:      req.CC,
		BCC:     req.BCC,
		Subject: req.Subject,
		Body:    req.Body,
		IsHTML:  req.IsHTML,
	}
	if len(req.Attachments) > 0 {
		message.Attachments = make([]model.MailAttachment, 0, len(req.Attachments))
		for _, attachment := range req.Attachments {
			message.Attachments = append(message.Attachments, model.MailAttachment{
				Filename:      attachment.Filename,
				ContentType:   attachment.ContentType,
				ContentBase64: attachment.ContentBase64,
			})
		}
	}
	acceptedRecipients, err := h.service.SendMail(r.Context(), message)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			response.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		case errors.Is(err, service.ErrDisabled):
			response.WriteError(w, r, http.StatusServiceUnavailable, "MAIL_DISABLED", "메일 발송 기능이 비활성화되어 있습니다")
		default:
			response.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "메일 발송 중 오류가 발생했습니다")
		}
		return
	}

	response.WriteSuccess(w, r, http.StatusAccepted, dto.SendMailResponseData{
		AcceptedRecipients: acceptedRecipients,
	})
}
