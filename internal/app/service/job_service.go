package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/domain/port"
)

type registeredJob struct {
	runner  port.JobRunner
	job     model.ScheduledJob
	running bool
}

// JobService는 스케줄 작업 등록과 실행 이력 처리를 담당한다.
type JobService struct {
	mu             sync.Mutex
	jobs           map[string]*registeredJob
	executionStore port.JobExecutionRepository
}

// NewJobService는 JobService를 생성한다.
func NewJobService(executionStore port.JobExecutionRepository, runners ...port.JobRunner) *JobService {
	svc := &JobService{
		jobs:           make(map[string]*registeredJob, len(runners)),
		executionStore: executionStore,
	}
	for _, runner := range runners {
		svc.Register(runner)
	}
	return svc
}

// Register는 스케줄 작업을 등록한다.
func (s *JobService) Register(runner port.JobRunner) {
	definition := runner.Definition()
	if definition.NextRunAt.IsZero() {
		definition.NextRunAt = time.Now()
	}
	if definition.Interval <= 0 {
		definition.Interval = time.Minute
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[definition.ID] = &registeredJob{
		runner: runner,
		job:    definition,
	}
}

// DueJobs는 현재 시각 기준으로 실행 가능한 작업을 가져온다.
func (s *JobService) DueJobs(now time.Time) []port.JobRunner {
	s.mu.Lock()
	defer s.mu.Unlock()

	runners := make([]port.JobRunner, 0, len(s.jobs))
	for _, entry := range s.jobs {
		if !entry.job.Enabled {
			continue
		}
		if !entry.job.AllowConcurrent && entry.running {
			continue
		}
		if now.Before(entry.job.NextRunAt) {
			continue
		}

		entry.running = true
		entry.job.NextRunAt = now.Add(entry.job.Interval)
		runners = append(runners, entry.runner)
	}

	return runners
}

// StartExecution은 작업 실행 이력을 생성한다.
func (s *JobService) StartExecution(ctx context.Context, jobID, runnerID string, startedAt time.Time) (*model.JobExecution, error) {
	execution := &model.JobExecution{
		ID:        newExecutionID(),
		JobID:     jobID,
		StartedAt: startedAt,
		Status:    model.JobExecutionStatusRunning,
		RunnerID:  runnerID,
	}
	if s.executionStore != nil {
		if err := s.executionStore.Create(ctx, execution); err != nil {
			return nil, fmt.Errorf("작업 실행 이력 생성 실패: %w", err)
		}
	}
	return execution, nil
}

// FinishExecution은 작업 실행 이력을 완료 상태로 갱신한다.
func (s *JobService) FinishExecution(ctx context.Context, jobID string, execution *model.JobExecution, runErr error, finishedAt time.Time) error {
	s.mu.Lock()
	if entry, ok := s.jobs[jobID]; ok {
		entry.running = false
	}
	s.mu.Unlock()

	if execution == nil {
		return nil
	}

	execution.FinishedAt = &finishedAt
	if runErr != nil {
		execution.Status = model.JobExecutionStatusFailed
		execution.ErrorMessage = runErr.Error()
	} else {
		execution.Status = model.JobExecutionStatusSuccess
	}

	if s.executionStore != nil {
		if err := s.executionStore.Update(ctx, execution); err != nil {
			return fmt.Errorf("작업 실행 이력 갱신 실패: %w", err)
		}
	}
	return nil
}

func newExecutionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("exec-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
