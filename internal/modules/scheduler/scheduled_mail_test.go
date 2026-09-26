package scheduler

import (
	"context"
	"errors"
	"net/textproto"
	"sync"
	"testing"
	"time"

	"able-rest-api/internal/modules/mail"
)

type memoryScheduledMailStore struct {
	mu    sync.Mutex
	item  ScheduledMail
	fail  bool
	claim int
}

func (s *memoryScheduledMailStore) Create(_ context.Context, item *ScheduledMail) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.item = *item
	return nil
}

func (s *memoryScheduledMailStore) ClaimDue(_ context.Context, now time.Time, lease time.Duration) (*ScheduledMail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return nil, errors.New("db unavailable")
	}
	i := &s.item
	due := i.Status == ScheduledMailPending && !i.ScheduledAt.After(now) ||
		i.Status == ScheduledMailRetry && i.NextRetryAt != nil && !i.NextRetryAt.After(now) ||
		i.Status == ScheduledMailProcessing && i.LeaseUntil != nil && !i.LeaseUntil.After(now)
	if !due {
		return nil, nil
	}
	i.Status = ScheduledMailProcessing
	i.AttemptCount++
	i.ClaimToken = "token"
	until := now.Add(lease)
	i.LeaseUntil = &until
	s.claim++
	copy := *i
	return &copy, nil
}

func (s *memoryScheduledMailStore) Complete(_ context.Context, id, token string, status ScheduledMailStatus, next *time.Time, lastError string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail || id != s.item.ID || token != s.item.ClaimToken || s.item.Status != ScheduledMailProcessing {
		return errors.New("claim mismatch")
	}
	s.item.Status, s.item.NextRetryAt, s.item.LastError = status, next, lastError
	s.item.ClaimToken = ""
	s.item.LeaseUntil = nil
	return nil
}

type scheduledMailDispatchStub struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (d *scheduledMailDispatchStub) Dispatch(_ context.Context, _ mail.MailMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	return d.err
}

func newScheduledMailTestJob(store *memoryScheduledMailStore, dispatch *scheduledMailDispatchStub, now time.Time) *ScheduledMailJob {
	job := NewScheduledMailJob(store, dispatch, true, time.Second, time.Minute, 2, 10*time.Second)
	job.now = func() time.Time { return now }
	return job
}

func TestScheduledMailDueAndIdempotent(t *testing.T) {
	now := time.Now().UTC()
	store := &memoryScheduledMailStore{item: ScheduledMail{ID: "1", Status: ScheduledMailPending, ScheduledAt: now.Add(time.Second)}}
	dispatch := &scheduledMailDispatchStub{}
	job := newScheduledMailTestJob(store, dispatch, now)
	if err := job.Run(context.Background()); err != nil || dispatch.calls != 0 {
		t.Fatalf("미래 예약 발송: calls=%d err=%v", dispatch.calls, err)
	}
	job.now = func() time.Time { return now.Add(2 * time.Second) }
	for range 2 {
		if err := job.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if dispatch.calls != 1 || store.item.Status != ScheduledMailSent || store.claim != 1 {
		t.Fatalf("중복 발송 또는 상태 오류: calls=%d item=%+v", dispatch.calls, store.item)
	}
}

func TestScheduledMailRetryAndFailure(t *testing.T) {
	now := time.Now().UTC()
	store := &memoryScheduledMailStore{item: ScheduledMail{ID: "1", Status: ScheduledMailPending, ScheduledAt: now}}
	dispatch := &scheduledMailDispatchStub{err: errors.New("smtp unavailable")}
	job := newScheduledMailTestJob(store, dispatch, now)
	if err := job.Run(context.Background()); err != nil || store.item.Status != ScheduledMailRetry || store.item.AttemptCount != 1 || store.item.NextRetryAt == nil || !store.item.NextRetryAt.Equal(now.Add(10*time.Second)) {
		t.Fatalf("재시도 기록 오류: item=%+v err=%v", store.item, err)
	}
	job.now = func() time.Time { return now.Add(9 * time.Second) }
	if err := job.Run(context.Background()); err != nil || dispatch.calls != 1 {
		t.Fatalf("조기 재시도: calls=%d err=%v", dispatch.calls, err)
	}
	job.now = func() time.Time { return now.Add(10 * time.Second) }
	if err := job.Run(context.Background()); err != nil || store.item.Status != ScheduledMailFailed || store.item.AttemptCount != 2 || dispatch.calls != 2 {
		t.Fatalf("최대 시도 처리 오류: item=%+v calls=%d err=%v", store.item, dispatch.calls, err)
	}
}

func TestScheduledMailRecoveryAndFailClosed(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-time.Second)
	future := now.Add(time.Second)
	store := &memoryScheduledMailStore{item: ScheduledMail{ID: "1", Status: ScheduledMailProcessing, ScheduledAt: now.Add(-time.Hour), AttemptCount: 1, LeaseUntil: &future}}
	dispatch := &scheduledMailDispatchStub{}
	job := newScheduledMailTestJob(store, dispatch, now)
	if err := job.Run(context.Background()); err != nil || dispatch.calls != 0 {
		t.Fatalf("유효한 lease를 중복 claim: calls=%d err=%v", dispatch.calls, err)
	}
	store.item.LeaseUntil = &old
	if err := job.Run(context.Background()); err != nil || dispatch.calls != 1 || store.item.AttemptCount != 2 || store.item.Status != ScheduledMailSent {
		t.Fatalf("고착 복구 오류: item=%+v calls=%d err=%v", store.item, dispatch.calls, err)
	}
	store.fail = true
	if err := job.Run(context.Background()); err == nil || dispatch.calls != 1 {
		t.Fatalf("DB 오류 후 발송: calls=%d err=%v", dispatch.calls, err)
	}
}

func TestScheduledMailPermanentSMTPFailure(t *testing.T) {
	now := time.Now().UTC()
	store := &memoryScheduledMailStore{item: ScheduledMail{ID: "1", Status: ScheduledMailPending, ScheduledAt: now}}
	dispatch := &scheduledMailDispatchStub{err: &textproto.Error{Code: 550, Msg: "recipient rejected"}}
	if err := newScheduledMailTestJob(store, dispatch, now).Run(context.Background()); err != nil || store.item.Status != ScheduledMailFailed || store.item.NextRetryAt != nil {
		t.Fatalf("영구 실패 상태 오류: item=%+v err=%v", store.item, err)
	}
}

func TestScheduledMailConcurrentClaim(t *testing.T) {
	now := time.Now().UTC()
	store := &memoryScheduledMailStore{item: ScheduledMail{ID: "1", Status: ScheduledMailPending, ScheduledAt: now}}
	dispatch := &scheduledMailDispatchStub{}
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := newScheduledMailTestJob(store, dispatch, now).Run(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if store.claim != 1 || dispatch.calls != 1 {
		t.Fatalf("중복 claim: claim=%d calls=%d", store.claim, dispatch.calls)
	}
}

func TestScheduleMailPreservesAttachmentForDispatch(t *testing.T) {
	store := &memoryScheduledMailStore{}
	message := mail.MailMessage{To: []string{"user@example.com"}, Subject: "Subject", Body: "Body", Attachments: []mail.MailAttachment{{Filename: "note.txt", ContentType: "text/plain", ContentBase64: "aGVsbG8="}}}
	id, err := ScheduleMail(context.Background(), store, time.Now().UTC(), message)
	if err != nil || id == "" || store.item.ID != id || store.item.Payload.Attachments[0].ContentBase64 != "aGVsbG8=" {
		t.Fatalf("예약 payload 오류: id=%q item=%+v err=%v", id, store.item, err)
	}
	if _, err := mail.NormalizeMessage(store.item.Payload); err != nil {
		t.Fatalf("저장된 첨부파일을 발송할 수 없습니다: %v", err)
	}
	message.To = nil
	if _, err := ScheduleMail(context.Background(), store, time.Now().UTC(), message); !errors.Is(err, mail.ErrInvalidInput) {
		t.Fatalf("잘못된 예약을 허용했습니다: %v", err)
	}
}
