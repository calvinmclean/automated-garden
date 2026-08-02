ALTER TABLE water_schedules ADD COLUMN send_reminder BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE water_schedules
SET send_reminder = COALESCE(json_extract(notification_settings, '$.watering_reminder'), FALSE);
ALTER TABLE water_schedules DROP COLUMN notification_settings;
