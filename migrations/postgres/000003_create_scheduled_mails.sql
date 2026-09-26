CREATE TABLE IF NOT EXISTS scheduled_mails (
    id VARCHAR(64) PRIMARY KEY,
    scheduled_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'SENT', 'RETRY', 'FAILED')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_retry_at TIMESTAMPTZ NULL,
    last_error VARCHAR(1024) NULL,
    claim_token VARCHAR(32) NULL,
    lease_until TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scheduled_mails_due ON scheduled_mails (status, scheduled_at, next_retry_at, lease_until);
