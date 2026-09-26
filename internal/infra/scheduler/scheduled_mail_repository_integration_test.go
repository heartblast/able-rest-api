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
