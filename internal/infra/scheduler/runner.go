package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/domain/port"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/platform/logger"
)

// Runner는 주기적으로 due job을 실행한다.
type Runner struct {
	cfg        config.SchedulerConfig
	log        logger.Logger
	jobService *service.JobService
	lock       port.SchedulerLock
}

// NewRunner는 Runner를 생성한다.
func NewRunner(cfg config.SchedulerConfig, log logger.Logger, jobService *service.JobService, lock port.SchedulerLock) *Runner {
	return &Runner{
		cfg:        cfg,
		log:        log,
		jobService: jobService,
		lock:       lock,
	}
}

// Run은 스케줄러 루프를 실행한다.
func (r *Runner) Run(ctx context.Context) error {
	if !r.cfg.Enabled {
		r.log.Info("scheduler disabled")
		return nil
	}

	location, err := time.LoadLocation(r.cfg.Timezone)
	if err != nil {
		return fmt.Errorf("scheduler timezone load failed: %w", err)
	}

	r.log.Info("scheduler started", "runner_id", r.cfg.RunnerID, "poll_interval", r.cfg.PollInterval.String(), "timezone", location.String())

	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	for {
		r.runOnce(ctx, time.Now().In(location))

		select {
		case <-ctx.Done():
			r.log.Info("scheduler stopped", "runner_id", r.cfg.RunnerID)
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runner) runOnce(ctx context.Context, now time.Time) {
	jobs := r.jobService.DueJobs(now)
	if len(jobs) == 0 {
		return
	}

	sem := make(chan struct{}, r.cfg.MaxParallelJobs)
	var wg sync.WaitGroup

	for _, job := range jobs {
		sem <- struct{}{}
		wg.Add(1)

		go func(job port.JobRunner) {
			defer func() {
				<-sem
				wg.Done()
			}()
			r.executeJob(ctx, job, now)
		}(job)
	}

	wg.Wait()
}

func (r *Runner) executeJob(ctx context.Context, job port.JobRunner, now time.Time) {
	definition := job.Definition()

	acquired, release, err := r.lock.Acquire(ctx, definition.ID)
	if err != nil {
		r.log.Error("scheduler lock acquire failed", "job_id", definition.ID, "error", err)
		_ = r.jobService.FinishExecution(ctx, definition.ID, nil, err, time.Now())
		return
	}
	if !acquired {
		r.log.Info("scheduler job skipped due to lock", "job_id", definition.ID)
		_ = r.jobService.FinishExecution(ctx, definition.ID, nil, nil, time.Now())
		return
	}
	defer func() {
		if release != nil {
			if releaseErr := release(); releaseErr != nil {
				r.log.Error("scheduler lock release failed", "job_id", definition.ID, "error", releaseErr)
			}
		}
	}()

	execution, err := r.jobService.StartExecution(ctx, definition.ID, r.cfg.RunnerID, now)
	if err != nil {
		r.log.Error("scheduler execution start failed", "job_id", definition.ID, "error", err)
		_ = r.jobService.FinishExecution(ctx, definition.ID, nil, err, time.Now())
		return
	}

	started := time.Now()
	runErr := job.Run(ctx)
	finishedAt := time.Now()

	if err := r.jobService.FinishExecution(ctx, definition.ID, execution, runErr, finishedAt); err != nil {
		r.log.Error("scheduler execution finish failed", "job_id", definition.ID, "error", err)
		return
	}

	if runErr != nil {
		r.log.Error("scheduler job failed", "job_id", definition.ID, "job_name", definition.Name, "duration", finishedAt.Sub(started).String(), "error", runErr)
		return
	}

	r.log.Info("scheduler job completed", "job_id", definition.ID, "job_name", definition.Name, "duration", finishedAt.Sub(started).String())
}
