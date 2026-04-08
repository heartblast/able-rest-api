# Goal

Add or evolve the scheduler process using the repository's existing layered architecture and config patterns.

## Scope

- separate `cmd/scheduler` process
- scheduler config section and env overrides
- scheduled job and execution models
- scheduler ports and runner
- lock strategy for duplicate execution prevention
- execution history persistence

## Constraints

- read the existing codebase before editing
- keep API server and scheduler process separate
- put domain concepts in `internal/domain`
- keep job orchestration in app/services or scheduler infra, not in handlers
- preserve graceful shutdown behavior
- keep logs free of sensitive payloads

## Read First

- `cmd/server/main.go`
- `cmd/scheduler/main.go`
- `internal/infra/config/config.go`
- `internal/domain/model/scheduled_job.go`
- `internal/domain/model/job_execution.go`
- `internal/domain/port/job_runner.go`
- `internal/domain/port/job_execution_repository.go`
- `internal/domain/port/scheduler_lock.go`
- `internal/app/service/job_service.go`
- `internal/infra/scheduler/runner.go`
- `internal/infra/scheduler/postgres_lock.go`

## Config Expectations

- `enabled`
- `poll_interval`
- `timezone`
- `runner_id`
- `lock_provider`
- `max_parallel_jobs`
- `shutdown_timeout`

## Verify

- `gofmt`
- `go test ./...`
- config example updated
- scheduler wiring builds cleanly

## Final Output

- implemented behavior
- architectural decisions
- modified files
- verification result
- remaining limits
