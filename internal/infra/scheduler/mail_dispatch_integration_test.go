package scheduler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	"github.com/heartblast/able-rest-api/internal/modules/mail"
	jobmodule "github.com/heartblast/able-rest-api/internal/modules/scheduler"
)

type recordingSender struct {
	messages []mail.MailMessage
	err      error
}

func (s *recordingSender) Send(_ context.Context, message mail.MailMessage) error {
	s.messages = append(s.messages, message)
	return s.err
}

type recordingExecutions struct {
	created []jobmodule.JobExecution
	updated []jobmodule.JobExecution
}

func (s *recordingExecutions) Create(_ context.Context, execution *jobmodule.JobExecution) error {
	s.created = append(s.created, *execution)
	return nil
}

func (s *recordingExecutions) Update(_ context.Context, execution *jobmodule.JobExecution) error {
	s.updated = append(s.updated, *execution)
	return nil
}

type quietLogger struct{}

func (quietLogger) Info(string, ...any)  {}
func (quietLogger) Error(string, ...any) {}

func runMailJob(t *testing.T, enabled bool, sender *recordingSender, message mail.MailMessage) *recordingExecutions {
	t.Helper()
	service := mail.NewMailService(true, sender)
	command := mail.NewScheduledDispatch(service, message)
	job := jobmodule.NewMailDispatchJob(command, enabled, time.Minute)
	store := &recordingExecutions{}
	runner := NewRunner(config.SchedulerConfig{Enabled: true, MaxParallelJobs: 1, RunnerID: "test"}, quietLogger{}, jobmodule.NewJobService(store, job), noopLock{})
	now := time.Now().Add(time.Second)
	runner.runOnce(context.Background(), now)
	runner.runOnce(context.Background(), now)
	return store
}

func TestMailDispatchExecutionSuccess(t *testing.T) {
	sender := &recordingSender{}
	message := mail.MailMessage{
		To: []string{"to@example.com"}, CC: []string{"cc@example.com"}, BCC: []string{"bcc@example.com"},
		Subject: "Subject", Body: "Body", IsHTML: true,
		Attachments: []mail.MailAttachment{{Filename: "note.txt", ContentType: "text/plain", ContentBase64: "aGVsbG8="}},
	}
	store := runMailJob(t, true, sender, message)
	if len(sender.messages) != 1 {
		t.Fatalf("발송 횟수: %d", len(sender.messages))
	}
	want := message
	want.Attachments = []mail.MailAttachment{{Filename: "note.txt", ContentType: "text/plain", Content: []byte("hello")}}
	if !reflect.DeepEqual(sender.messages[0], want) {
		t.Fatalf("전달된 메일이 다릅니다: %#v", sender.messages[0])
	}
	if len(store.created) != 1 || store.created[0].Status != jobmodule.JobExecutionStatusRunning || len(store.updated) != 1 || store.updated[0].Status != jobmodule.JobExecutionStatusSuccess {
		t.Fatalf("실행 이력이 올바르지 않습니다: created=%#v updated=%#v", store.created, store.updated)
	}
}

func TestMailDispatchExecutionFailure(t *testing.T) {
	sender := &recordingSender{err: errors.New("smtp unavailable")}
	store := runMailJob(t, true, sender, mail.MailMessage{To: []string{"to@example.com"}, Subject: "Subject", Body: "Body"})
	if len(sender.messages) != 1 || len(store.updated) != 1 || store.updated[0].Status != jobmodule.JobExecutionStatusFailed || store.updated[0].ErrorMessage == "" {
		t.Fatalf("발송 실패가 기록되지 않았습니다: sent=%d updated=%#v", len(sender.messages), store.updated)
	}
}

func TestMailDispatchDisabledAndInvalid(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		message mail.MailMessage
		status  jobmodule.JobExecutionStatus
	}{
		{"disabled", false, mail.MailMessage{To: []string{"to@example.com"}, Subject: "Subject", Body: "Body"}, ""},
		{"invalid", true, mail.MailMessage{To: []string{"bad-address"}, Subject: "Subject", Body: "Body"}, jobmodule.JobExecutionStatusFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender := &recordingSender{}
			store := runMailJob(t, tc.enabled, sender, tc.message)
			if len(sender.messages) != 0 {
				t.Fatalf("불필요한 SMTP 호출: %d", len(sender.messages))
			}
			if tc.status == "" && (len(store.created) != 0 || len(store.updated) != 0) {
				t.Fatalf("비활성 작업의 실행 이력이 생성되었습니다: %#v", store)
			}
			if tc.status != "" && (len(store.updated) != 1 || store.updated[0].Status != tc.status) {
				t.Fatalf("잘못된 입력의 실행 상태가 다릅니다: %#v", store.updated)
			}
		})
	}
}
