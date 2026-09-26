package scheduler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	"github.com/heartblast/able-rest-api/internal/modules/scheduler"
)

type scheduledMailRepository struct {
	db     *sql.DB
	vendor config.DBVendor
}

// NewScheduledMailRepository는 지원 DB의 예약 메일 저장소를 생성한다.
func NewScheduledMailRepository(vendor config.DBVendor, db *sql.DB) (scheduler.ScheduledMailRepository, error) {
	if db == nil || (vendor != config.DBVendorPostgres && vendor != config.DBVendorMySQL) {
		return nil, errors.New("예약 메일에는 PostgreSQL 또는 MySQL 연결이 필요합니다")
	}
	return &scheduledMailRepository{db: db, vendor: vendor}, nil
}

func (r *scheduledMailRepository) bind(pg, mysql string) string {
	if r.vendor == config.DBVendorPostgres {
		return pg
	}
	return mysql
}

// Create는 신규 예약 메일을 저장한다.
func (r *scheduledMailRepository) Create(ctx context.Context, item *scheduler.ScheduledMail) error {
	if item == nil || item.ID == "" || item.MessageID != "<"+item.ID+"@scheduled.able-rest-api.invalid>" || item.ScheduledAt.IsZero() || len(item.ID) > 64 || item.Status != "" && item.Status != scheduler.ScheduledMailPending {
		return errors.New("예약 메일 입력이 올바르지 않습니다")
	}
	payload, err := json.Marshal(item.Payload)
	if err != nil || len(payload) > 30<<20 {
		return errors.New("예약 메일 payload가 올바르지 않습니다")
	}
	now := time.Now().UTC()
	args := []any{item.ID, item.MessageID, item.ScheduledAt.UTC(), string(payload), now}
	if r.vendor == config.DBVendorMySQL {
		args = append(args, now)
	}
	_, err = r.db.ExecContext(ctx, r.bind(
		`INSERT INTO scheduled_mails (id, message_id, scheduled_at, payload, status, attempt_count, created_at, updated_at) VALUES ($1, $2, $3, $4, 'PENDING', 0, $5, $5)`,
		`INSERT INTO scheduled_mails (id, message_id, scheduled_at, payload, status, attempt_count, created_at, updated_at) VALUES (?, ?, ?, ?, 'PENDING', 0, ?, ?)`),
		args...)
	return err
}

// ClaimDue는 행 잠금으로 한 건을 선택하고 짧은 트랜잭션에서 claim한다.
func (r *scheduledMailRepository) ClaimDue(ctx context.Context, now time.Time, lease time.Duration) (*scheduler.ScheduledMail, error) {
	if lease <= 0 {
		return nil, errors.New("claim lease가 필요합니다")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := r.bind(
		`SELECT id, message_id, scheduled_at, payload, status, attempt_count, next_retry_at, last_error, created_at, updated_at FROM scheduled_mails WHERE (status = 'PENDING' AND scheduled_at <= $1) OR (status = 'RETRY' AND next_retry_at <= $1) OR (status = 'PROCESSING' AND lease_until <= $1) ORDER BY scheduled_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`,
		`SELECT id, message_id, scheduled_at, payload, status, attempt_count, next_retry_at, last_error, created_at, updated_at FROM scheduled_mails WHERE (status = 'PENDING' AND scheduled_at <= ?) OR (status = 'RETRY' AND next_retry_at <= ?) OR (status = 'PROCESSING' AND lease_until <= ?) ORDER BY scheduled_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`)
	args := []any{now.UTC()}
	if r.vendor == config.DBVendorMySQL {
		args = []any{now.UTC(), now.UTC(), now.UTC()}
	}
	var item scheduler.ScheduledMail
	var payload []byte
	var nextRetry sql.NullTime
	var lastError sql.NullString
	err = tx.QueryRowContext(ctx, query, args...).Scan(&item.ID, &item.MessageID, &item.ScheduledAt, &payload, &item.Status, &item.AttemptCount, &nextRetry, &lastError, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(payload, &item.Payload); err != nil {
		return nil, fmt.Errorf("예약 메일 payload 해석 실패: %w", err)
	}
	if nextRetry.Valid {
		item.NextRetryAt = &nextRetry.Time
	}
	item.LastError = lastError.String
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	item.ClaimToken = hex.EncodeToString(tokenBytes)
	until := now.UTC().Add(lease)
	item.LeaseUntil = &until
	item.AttemptCount++
	update := r.bind(
		`UPDATE scheduled_mails SET status = 'PROCESSING', attempt_count = $2, claim_token = $3, lease_until = $4, next_retry_at = NULL, updated_at = $5 WHERE id = $1`,
		`UPDATE scheduled_mails SET status = 'PROCESSING', attempt_count = ?, claim_token = ?, lease_until = ?, next_retry_at = NULL, updated_at = ? WHERE id = ?`)
	updateArgs := []any{item.ID, item.AttemptCount, item.ClaimToken, until, now.UTC()}
	if r.vendor == config.DBVendorMySQL {
		updateArgs = []any{item.AttemptCount, item.ClaimToken, until, now.UTC(), item.ID}
	}
	if _, err := tx.ExecContext(ctx, update, updateArgs...); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item.Status = scheduler.ScheduledMailProcessing
	return &item, nil
}

// Complete는 claim token이 일치할 때에만 상태를 확정한다.
func (r *scheduledMailRepository) Complete(ctx context.Context, id, token string, status scheduler.ScheduledMailStatus, next *time.Time, lastError string) error {
	if id == "" || token == "" || status != scheduler.ScheduledMailSent && status != scheduler.ScheduledMailRetry && status != scheduler.ScheduledMailFailed {
		return errors.New("예약 메일 완료 상태가 올바르지 않습니다")
	}
	if status == scheduler.ScheduledMailRetry && next == nil || status != scheduler.ScheduledMailRetry && next != nil {
		return errors.New("예약 메일 재시도 시간이 올바르지 않습니다")
	}
	if len(lastError) > 1024 {
		lastError = lastError[:1024]
	}
	var retryAt any
	if next != nil {
		retryAt = next.UTC()
	}
	query := r.bind(
		`UPDATE scheduled_mails SET status = $3, next_retry_at = $4, last_error = $5, claim_token = NULL, lease_until = NULL, updated_at = $6 WHERE id = $1 AND claim_token = $2 AND status = 'PROCESSING'`,
		`UPDATE scheduled_mails SET status = ?, next_retry_at = ?, last_error = ?, claim_token = NULL, lease_until = NULL, updated_at = ? WHERE id = ? AND claim_token = ? AND status = 'PROCESSING'`)
	args := []any{id, token, string(status), retryAt, lastError, time.Now().UTC()}
	if r.vendor == config.DBVendorMySQL {
		args = []any{string(status), retryAt, lastError, time.Now().UTC(), id, token}
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("예약 메일 claim이 만료되었거나 변경되었습니다")
	}
	return nil
}
