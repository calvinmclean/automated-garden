ALTER TABLE water_schedules ADD COLUMN notification_settings TEXT;
UPDATE water_schedules
SET notification_settings = json_object(
    'watering_reminder', json(CASE WHEN send_reminder THEN 'true' ELSE 'false' END),
    'watering_errors', json('false')
);
ALTER TABLE water_schedules DROP COLUMN send_reminder;
