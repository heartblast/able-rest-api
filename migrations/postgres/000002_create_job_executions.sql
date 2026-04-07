CREATE TABLE IF NOT EXISTS job_executions (
    id VARCHAR(64) PRIMARY KEY,
    job_id VARCHAR(100) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT NULL,
    runner_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_executions_job_id_started_at
    ON job_executions (job_id, started_at DESC);
