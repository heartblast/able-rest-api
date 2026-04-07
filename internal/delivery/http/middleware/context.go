package middleware

import "context"

type ctxKey string

const requestIDKey ctxKey = "request_id"

// WithRequestID는 요청 ID를 컨텍스트에 저장한다.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext는 컨텍스트에서 요청 ID를 조회한다.
func RequestIDFromContext(ctx context.Context) string {
	val, _ := ctx.Value(requestIDKey).(string)
	return val
}
