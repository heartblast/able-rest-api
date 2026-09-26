package dto

import "able-rest-api/internal/platform/http/response"

// ErrorResponse는 공통 실패 응답이다.
type ErrorResponse = response.ErrorResponse

// ErrorDetail은 실패 상세를 표현한다.
type ErrorDetail = response.ErrorDetail

// SuccessResponse는 공통 성공 응답 래퍼다.
type SuccessResponse[T any] = response.SuccessResponse[T]

// HealthData는 상태 점검 응답 본문이다.
type HealthData = response.HealthData
