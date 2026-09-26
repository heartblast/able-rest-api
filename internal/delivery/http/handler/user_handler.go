package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/delivery/http/dto"
	"able-rest-api/internal/delivery/http/middleware"
	"able-rest-api/internal/domain/model"
)

// UserHandler는 사용자 HTTP 요청을 처리한다.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler는 UserHandler를 생성한다.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_ID", "id는 숫자여야 합니다")
		return
	}

	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		case errors.Is(err, service.ErrNotFound):
			writeError(w, r, http.StatusNotFound, "NOT_FOUND", "사용자를 찾을 수 없습니다")
		default:
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "사용자 조회 중 오류가 발생했습니다")
		}
		return
	}

	writeSuccess(w, r, http.StatusOK, toUserResponse(*user))
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	users, err := h.service.ListUsers(r.Context(), limit, offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "사용자 목록 조회 중 오류가 발생했습니다")
		return
	}

	items := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, toUserResponse(user))
	}

	writeSuccess(w, r, http.StatusOK, dto.UserListResponse{
		Items: items,
		Count: len(items),
	})
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := middleware.DecodeJSON(r, &req); err != nil {
		if middleware.JSONErrorStatus(err) == http.StatusRequestEntityTooLarge {
			writeError(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "요청 본문이 너무 큽니다")
			return
		}
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "JSON 본문이 올바르지 않습니다")
		return
	}

	user, err := h.service.CreateUser(r.Context(), req.Name, req.Email)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}

		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "사용자 생성 중 오류가 발생했습니다")
		return
	}

	writeSuccess(w, r, http.StatusCreated, toUserResponse(*user))
}

func toUserResponse(user model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
