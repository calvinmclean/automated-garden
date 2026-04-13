-- Add url column to notification_clients
ALTER TABLE notification_clients ADD COLUMN url TEXT NOT NULL DEFAULT '';

-- Migrate pushover clients: convert type+options to shoutrrr URL format
-- pushover://shoutrrr:{app_token}@{recipient_token}/
UPDATE notification_clients
SET url = 'pushover://shoutrrr:' || json_extract(options, '$.app_token') || '@' || json_extract(options, '$.recipient_token') || '/'
WHERE type = 'pushover'
  AND json_extract(options, '$.app_token') IS NOT NULL
  AND json_extract(options, '$.recipient_token') IS NOT NULL;

-- Migrate fake clients: convert to fake:// URL format
UPDATE notification_clients
SET url = 'fake://'
WHERE type = 'fake';

-- Drop old columns
ALTER TABLE notification_clients DROP COLUMN type;
ALTER TABLE notification_clients DROP COLUMN options;