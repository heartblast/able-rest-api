package response

// ErrorResponse는 공통 실패 응답이다.
type ErrorResponse struct {
	Success   bool        `json:"success" example:"false"`
	RequestID string      `json:"request_id,omitempty" example:"7f5f1f8e-6fd7-4d33-bfdf-6d414dc2b0b1"`
	Error     ErrorDetail `json:"error"`
}

// ErrorDetail은 실패 상세를 표현한다.
type ErrorDetail struct {
	Code    string `json:"code" example:"VALIDATION_ERROR"`
	Message string `json:"message" example:"잘못된 요청입니다"`
}

// SuccessResponse는 공통 성공 응답 래퍼다.
type SuccessResponse[T any] struct {
	Success   bool   `json:"success" example:"true"`
	RequestID string `json:"request_id,omitempty" example:"7f5f1f8e-6fd7-4d33-bfdf-6d414dc2b0b1"`
	Data      T      `json:"data"`
}

// HealthData는 상태 점검 응답 본문이다.
type HealthData struct {
	Status string `json:"status" example:"ok"`
}
