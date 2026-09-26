package user

import "able-rest-api/internal/platform/http/response"

// CreateUserRequest는 사용자 생성 요청 DTO다.
type CreateUserRequest struct {
	Name  string `json:"name" example:"Alice"`
	Email string `json:"email" example:"alice@example.com"`
}

// UserResponse는 외부 API 응답 전용 DTO다.
type UserResponse struct {
	ID        int64  `json:"id" example:"1"`
	Name      string `json:"name" example:"Alice"`
	Email     string `json:"email" example:"alice@example.com"`
	CreatedAt string `json:"created_at" example:"2026-04-06T10:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-04-06T10:00:00Z"`
}

// UserListResponse는 사용자 목록 응답 DTO다.
type UserListResponse struct {
	Items []UserResponse `json:"items"`
	Count int            `json:"count" example:"1"`
}

// UserGetResponse는 단건 조회 성공 응답 DTO다.
type UserGetResponse = response.SuccessResponse[UserResponse]

// UserCreateResponse는 생성 성공 응답 DTO다.
type UserCreateResponse = response.SuccessResponse[UserResponse]

// UserListEnvelopeResponse는 목록 조회 성공 응답 DTO다.
type UserListEnvelopeResponse = response.SuccessResponse[UserListResponse]
