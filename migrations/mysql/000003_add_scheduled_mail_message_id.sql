ALTER TABLE scheduled_mails ADD COLUMN message_id VARCHAR(255) NULL;
UPDATE scheduled_mails SET message_id = CONCAT('<', id, '@scheduled.able-rest-api.invalid>') WHERE message_id IS NULL;
ALTER TABLE scheduled_mails MODIFY COLUMN message_id VARCHAR(255) NOT NULL;
CREATE UNIQUE INDEX idx_scheduled_mails_message_id ON scheduled_mails (message_id);
