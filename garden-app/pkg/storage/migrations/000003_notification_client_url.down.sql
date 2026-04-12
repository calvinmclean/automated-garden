-- Re-add type and options columns
ALTER TABLE notification_clients ADD COLUMN type TEXT NOT NULL DEFAULT '';
ALTER TABLE notification_clients ADD COLUMN options JSON NOT NULL DEFAULT '{}';

-- Migrate pushover URLs back to type+options format
UPDATE notification_clients
SET type = 'pushover',
    options = json('{"app_token":"' || substr(url, instr(url, ':') + 10, instr(substr(url, instr(url, ':') + 10), '@') - 1) || '","recipient_token":"' || substr(url, instr(url, '@') + 1, instr(substr(url, instr(url, '@') + 1), '/') - 1) || '"}')
WHERE url LIKE 'pushover://%';

-- Migrate fake URLs back to type+options format
UPDATE notification_clients
SET type = 'fake',
    options = json('{}')
WHERE url LIKE 'fake://%';

-- Drop url column
ALTER TABLE notification_clients DROP COLUMN url;