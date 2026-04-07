package service

import (
	"context"
	"testing"
	"time"

	"my-api/internal/domain/model"
)

type stubScheduledJob struct {
	definition model.ScheduledJob
	err        error
}

func (j *stubScheduledJob) Definition() model.ScheduledJob {
	return j.definition
}

func (j *stubScheduledJob) Run(_ context.Context) error {
	return j.err
}

type stubExecutionStore struct {
	created []*model.JobExecution
	updated []*model.JobExecution
}

func (s *stubExecutionStore) Create(_ context.Context, execution *model.JobExecution) error {
	s.created = append(s.created, execution)
	return nil
}

func (s *stubExecutionStore) Update(_ context.Context, execution *model.JobExecution) error {
	s.updated = append(s.updated, execution)
	return nil
}

func TestJobServiceDueJobs(t *testing.T) {
	store := &stubExecutionStore{}
	job := &stubScheduledJob{
		definition: model.ScheduledJob{
			ID:              "job-1",
			Name:            "job-1",
			Enabled:         true,
			Interval:        time.Minute,
			NextRunAt:       time.Now().Add(-time.Second),
			AllowConcurrent: false,
		},
	}

	svc := NewJobService(store, job)
	dueJobs := svc.DueJobs(time.Now())
	if len(dueJobs) != 1 {
		t.Fatalf("expected 1 due job, got %d", len(dueJobs))
	}

	dueJobs = svc.DueJobs(time.Now())
	if len(dueJobs) != 0 {
		t.Fatalf("expected no due jobs while running, got %d", len(dueJobs))
	}
}

func TestJobServiceExecutionLifecycle(t *testing.T) {
	store := &stubExecutionStore{}
	job := &stubScheduledJob{
		definition: model.ScheduledJob{
			ID:       "job-1",
			Name:     "job-1",
			Enabled:  true,
			Interval: time.Minute,
		},
	}

	svc := NewJobService(store, job)
	startedAt := time.Now()
	execution, err := svc.StartExecution(context.Background(), "job-1", "runner-1", startedAt)
	if err != nil {
		t.Fatalf("StartExecution returned error: %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected 1 created execution, got %d", len(store.created))
	}

	if err := svc.FinishExecution(context.Background(), "job-1", execution, nil, startedAt.Add(time.Second)); err != nil {
		t.Fatalf("FinishExecution returned error: %v", err)
	}
	if len(store.updated) != 1 {
		t.Fatalf("expected 1 updated execution, got %d", len(store.updated))
	}
	if store.updated[0].Status != model.JobExecutionStatusSuccess {
		t.Fatalf("expected success status, got %s", store.updated[0].Status)
	}
}
