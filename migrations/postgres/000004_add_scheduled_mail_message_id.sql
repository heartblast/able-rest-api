ALTER TABLE scheduled_mails ADD COLUMN IF NOT EXISTS message_id VARCHAR(255);
UPDATE scheduled_mails SET message_id = '<' || id || '@scheduled.able-rest-api.invalid>' WHERE message_id IS NULL;
ALTER TABLE scheduled_mails ALTER COLUMN message_id SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_scheduled_mails_message_id ON scheduled_mails (message_id);
