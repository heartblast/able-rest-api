package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/textproto"
	"time"

	"github.com/heartblast/able-rest-api/internal/modules/mail"
	"github.com/heartblast/able-rest-api/internal/platform/logger"
)

// ScheduledMailStatus는 예약 메일의 영속 상태다.
type ScheduledMailStatus string

const (
	ScheduledMailPending    ScheduledMailStatus = "PENDING"
	ScheduledMailProcessing ScheduledMailStatus = "PROCESSING"
	ScheduledMailSent       ScheduledMailStatus = "SENT"
	ScheduledMailRetry      ScheduledMailStatus = "RETRY"
	ScheduledMailFailed     ScheduledMailStatus = "FAILED"
)

// ScheduledMail은 개별 예약 메일과 발송 시도 정보를 담는다.
type ScheduledMail struct {
	ID           string
	MessageID    string
	ScheduledAt  time.Time
	Payload      mail.MailMessage
	Status       ScheduledMailStatus
	AttemptCount int
	NextRetryAt  *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClaimToken   string
	LeaseUntil   *time.Time
}

// ScheduledMailRepository는 원자적 claim과 상태 전이를 제공한다.
type ScheduledMailRepository interface {
	Create(context.Context, *ScheduledMail) error
	ClaimDue(context.Context, time.Time, time.Duration) (*ScheduledMail, error)
	Complete(context.Context, string, string, ScheduledMailStatus, *time.Time, string) error
}

// ScheduleMail은 입력을 검증하고 개별 메일 예약을 영속화한다.
func ScheduleMail(ctx context.Context, store ScheduledMailRepository, scheduledAt time.Time, message mail.MailMessage) (string, error) {
	if store == nil || scheduledAt.IsZero() {
		return "", errors.New("예약 저장소와 예약 시간이 필요합니다")
	}
	if _, err := mail.NormalizeMessage(message); err != nil {
		return "", err
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	id := hex.EncodeToString(bytes)
	message.MessageID = ""
	item := &ScheduledMail{ID: id, MessageID: "<" + id + "@scheduled.able-rest-api.invalid>", ScheduledAt: scheduledAt.UTC(), Payload: message, Status: ScheduledMailPending}
	if err := store.Create(ctx, item); err != nil {
		return "", fmt.Errorf("예약 메일 저장 실패: %w", err)
	}
	return id, nil
}

// ScheduledMailDispatch는 기존 메일 서비스로 발송을 위임한다.
type ScheduledMailDispatch interface {
	Dispatch(context.Context, mail.MailMessage) error
}

// ScheduledMailJob은 영속 예약 메일을 한 건씩 처리한다.
type ScheduledMailJob struct {
	store       ScheduledMailRepository
	dispatch    ScheduledMailDispatch
	enabled     bool
	interval    time.Duration
	lease       time.Duration
	maxAttempts int
	retryDelay  time.Duration
	now         func() time.Time
	log         logger.Logger
}

// NewScheduledMailJob은 예약 메일 작업을 생성한다.
func NewScheduledMailJob(store ScheduledMailRepository, dispatch ScheduledMailDispatch, enabled bool, interval, lease time.Duration, maxAttempts int, retryDelay time.Duration, log logger.Logger) *ScheduledMailJob {
	return &ScheduledMailJob{store: store, dispatch: dispatch, enabled: enabled, interval: interval, lease: lease, maxAttempts: maxAttempts, retryDelay: retryDelay, now: time.Now, log: log}
}

// Definition은 작업 실행 주기를 반환한다.
func (j *ScheduledMailJob) Definition() ScheduledJob {
	return ScheduledJob{ID: "scheduled-mail", Name: "scheduled-mail", Enabled: j.enabled, Interval: j.interval, AllowConcurrent: false}
}

// Run은 실행 가능한 메일을 claim하고 트랜잭션 밖에서 발송한다.
func (j *ScheduledMailJob) Run(ctx context.Context) error {
	if !j.enabled {
		return nil
	}
	if j.store == nil || j.dispatch == nil || j.lease <= 0 || j.maxAttempts < 1 || j.retryDelay <= 0 {
		return errors.New("예약 메일 작업 설정이 올바르지 않습니다")
	}
	for {
		item, err := j.store.ClaimDue(ctx, j.now(), j.lease)
		if err != nil {
			return fmt.Errorf("예약 메일 claim 실패: %w", err)
		}
		if item == nil {
			return nil
		}
		if item.MessageID == "" {
			return fmt.Errorf("예약 메일 message_id 누락: reservation_id=%s", item.ID)
		}
		item.Payload.MessageID = item.MessageID
		if j.log != nil {
			j.log.Info("scheduled mail dispatch", "reservation_id", item.ID, "message_id", item.MessageID, "attempt", item.AttemptCount)
		}
		status := ScheduledMailSent
		var next *time.Time
		lastError := ""
		if item.AttemptCount > j.maxAttempts {
			status = ScheduledMailFailed
			lastError = "발송 중단 후 최대 시도 횟수 초과"
		} else if err := j.dispatch.Dispatch(ctx, item.Payload); err != nil {
			lastError = err.Error()
			status = ScheduledMailFailed
			var smtpError *textproto.Error
			permanentSMTP := errors.As(err, &smtpError) && smtpError.Code >= 500 && smtpError.Code < 600
			if !errors.Is(err, mail.ErrInvalidInput) && !errors.Is(err, mail.ErrDisabled) && !permanentSMTP && item.AttemptCount < j.maxAttempts {
				status = ScheduledMailRetry
				// 지수 지연은 1시간으로 제한한다.
				delay := j.retryDelay
				for n := 1; n < item.AttemptCount && delay < time.Hour; n++ {
					delay *= 2
				}
				if delay > time.Hour {
					delay = time.Hour
				}
				when := j.now().Add(delay)
				next = &when
			}
		}
		if err := j.store.Complete(ctx, item.ID, item.ClaimToken, status, next, lastError); err != nil {
			return fmt.Errorf("예약 메일 상태 기록 실패: reservation_id=%s message_id=%s: %w", item.ID, item.MessageID, err)
		}
		if j.log != nil {
			j.log.Info("scheduled mail completed", "reservation_id", item.ID, "message_id", item.MessageID, "status", status, "attempt", item.AttemptCount)
		}
	}
}
