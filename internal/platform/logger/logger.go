package logger

import (
	"log/slog"
	"os"
)

// Logger는 구조화 로깅 추상화다.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type slogLogger struct {
	logger *slog.Logger
}

// New는 환경별 로그 레벨을 적용한 로거를 생성한다.
func New(env string) Logger {
	level := slog.LevelInfo
	if env == "local" || env == "dev" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// Info는 info 로그를 기록한다.
func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Error는 error 로그를 기록한다.
func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}
