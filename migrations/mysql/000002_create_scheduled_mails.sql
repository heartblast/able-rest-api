CREATE TABLE IF NOT EXISTS scheduled_mails (
    id VARCHAR(64) PRIMARY KEY,
    scheduled_at DATETIME(6) NOT NULL,
    payload JSON NOT NULL,
    status VARCHAR(20) NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    next_retry_at DATETIME(6) NULL,
    last_error VARCHAR(1024) NULL,
    claim_token VARCHAR(32) NULL,
    lease_until DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    INDEX idx_scheduled_mails_due (status, scheduled_at, next_retry_at, lease_until)
);
