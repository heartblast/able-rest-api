package scheduler

import (
	"context"
	"database/sql"
	"fmt"

	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/modules/scheduler"
)

type noopExecutionRepository struct{}

func (noopExecutionRepository) Create(_ context.Context, _ *scheduler.JobExecution) error { return nil }
func (noopExecutionRepository) Update(_ context.Context, _ *scheduler.JobExecution) error { return nil }

type postgresExecutionRepository struct {
	db *sql.DB
}

func (r *postgresExecutionRepository) Create(ctx context.Context, execution *scheduler.JobExecution) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO job_executions (id, job_id, started_at, status, error_message, runner_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		execution.ID,
		execution.JobID,
		execution.StartedAt,
		string(execution.Status),
		execution.ErrorMessage,
		execution.RunnerID,
	)
	if err != nil {
		return fmt.Errorf("insert job_executions failed: %w", err)
	}
	return nil
}

func (r *postgresExecutionRepository) Update(ctx context.Context, execution *scheduler.JobExecution) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE job_executions SET finished_at = $2, status = $3, error_message = $4, runner_id = $5 WHERE id = $1`,
		execution.ID,
		execution.FinishedAt,
		string(execution.Status),
		execution.ErrorMessage,
		execution.RunnerID,
	)
	if err != nil {
		return fmt.Errorf("update job_executions failed: %w", err)
	}
	return nil
}

// NewExecutionRepository는 설정에 맞는 실행 이력 저장소를 생성한다.
func NewExecutionRepository(vendor config.DBVendor, db *sql.DB) scheduler.JobExecutionRepository {
	if vendor == config.DBVendorPostgres && db != nil {
		return &postgresExecutionRepository{db: db}
	}
	return noopExecutionRepository{}
}
