package scheduler

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/modules/mail"
	job "able-rest-api/internal/modules/scheduler"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresScheduledMailClaim(t *testing.T) {
	dsn := os.Getenv("TEST_SCHEDULED_MAIL_PG_DSN")
	if dsn == "" {
		t.Skip("전용 PostgreSQL 테스트 DSN이 없습니다")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	ctx := context.Background()
	migration, err := os.ReadFile("../../../migrations/postgres/000003_create_scheduled_mails.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	messageIDMigration, err := os.ReadFile("../../../migrations/postgres/000004_add_scheduled_mail_message_id.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(messageIDMigration)); err != nil {
		t.Fatal(err)
	}
	store, err := NewScheduledMailRepository(config.DBVendorPostgres, db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	id, err := job.ScheduleMail(ctx, store, now.Add(-time.Second), mail.MailMessage{To: []string{"user@example.com"}, Subject: "Subject", Body: "Body"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM scheduled_mails WHERE id = $1`, id)
	var wg sync.WaitGroup
	results := make(chan *job.ScheduledMail, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := store.ClaimDue(ctx, now, time.Minute)
			if err != nil {
				t.Error(err)
			}
			results <- item
		}()
	}
	wg.Wait()
	close(results)
	claimed := 0
	var item *job.ScheduledMail
	for result := range results {
		if result != nil {
			claimed++
			item = result
		}
	}
	if claimed != 1 || item == nil || item.AttemptCount != 1 {
		t.Fatalf("동시 claim 결과: claimed=%d item=%+v", claimed, item)
	}
	if item.MessageID == "" {
		t.Fatal("영속 Message-ID가 없습니다")
	}
	if err := store.Complete(ctx, id, "stale-token", job.ScheduledMailSent, nil, ""); err == nil {
		t.Fatal("다른 token으로 완료했습니다")
	}
	if err := store.Complete(ctx, id, item.ClaimToken, job.ScheduledMailSent, nil, ""); err != nil {
		t.Fatal(err)
	}
	again, err := store.ClaimDue(ctx, now.Add(time.Hour), time.Minute)
	if err != nil || again != nil {
		t.Fatalf("완료된 예약을 재claim했습니다: item=%+v err=%v", again, err)
	}
}

func TestPostgresScheduledMailMessageIDAfterCompletionFailure(t *testing.T) {
	dsn := os.Getenv("TEST_SCHEDULED_MAIL_PG_DSN")
	if dsn == "" {
		t.Skip("전용 PostgreSQL 테스트 DSN이 없습니다")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	for _, path := range []string{"../../../migrations/postgres/000003_create_scheduled_mails.sql", "../../../migrations/postgres/000004_add_scheduled_mail_message_id.sql"} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewScheduledMailRepository(config.DBVendorPostgres, db)
	if err != nil {
		t.Fatal(err)
	}
	id, err := job.ScheduleMail(ctx, store, time.Now().Add(-time.Minute), mail.MailMessage{To: []string{"user@example.com"}, Subject: "Subject", Body: "Body"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM scheduled_mails WHERE id = $1`, id)
	first, err := store.ClaimDue(ctx, time.Now(), time.Second)
	if err != nil || first == nil || first.ID != id {
		t.Fatalf("첫 claim 실패: item=%+v err=%v", first, err)
	}
	// SMTP 수락 뒤 Complete를 호출하지 못한 상태를 재시작 후 재claim으로 재현한다.
	reopened, err := NewScheduledMailRepository(config.DBVendorPostgres, db)
	if err != nil {
		t.Fatal(err)
	}
	second, err := reopened.ClaimDue(ctx, time.Now().Add(2*time.Second), time.Second)
	if err != nil || second == nil || second.ID != id || second.MessageID != first.MessageID || second.ClaimToken == first.ClaimToken {
		t.Fatalf("재claim Message-ID 오류: first=%+v second=%+v err=%v", first, second, err)
	}
	if err := reopened.Complete(ctx, id, second.ClaimToken, job.ScheduledMailSent, nil, ""); err != nil {
		t.Fatal(err)
	}
}
