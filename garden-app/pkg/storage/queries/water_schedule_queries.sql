-- name: GetWaterSchedule :one
SELECT * FROM water_schedules
WHERE id = ? LIMIT 1;

-- name: ListAllWaterSchedules :many
SELECT * FROM water_schedules;

-- name: ListActiveWaterSchedules :many
SELECT * FROM water_schedules WHERE end_date IS NULL
   OR end_date > ?;

-- name: FindWaterSchedulesByWeatherClientID :many
SELECT id, name, description, duration, interval, start_date, start_time, end_date, active_period_start_month, active_period_end_month, weather_control, notification_client_id, send_reminder FROM water_schedules
WHERE weather_control IS NOT NULL AND (
    json_extract(weather_control, '$.rain_control.client_id') = ?
    OR json_extract(weather_control, '$.temperature_control.client_id') = ?
);

-- name: UpsertWaterSchedule :exec
INSERT INTO water_schedules (
  id, name, description,
  duration, interval,
  start_date, start_time,
  end_date,
  active_period_start_month, active_period_end_month,
  weather_control,
  notification_client_id,
  send_reminder
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  duration = EXCLUDED.duration,
  interval = EXCLUDED.interval,
  start_date = EXCLUDED.start_date,
  start_time = EXCLUDED.start_time,
  end_date = EXCLUDED.end_date,
  active_period_start_month = EXCLUDED.active_period_start_month,
  active_period_end_month = EXCLUDED.active_period_end_month,
  weather_control = EXCLUDED.weather_control,
  notification_client_id = EXCLUDED.notification_client_id,
  send_reminder = EXCLUDED.send_reminder;

-- name: SetWaterScheduleEndDate :exec
UPDATE water_schedules
SET end_date = ?
WHERE id = ?;

-- name: DeleteWaterSchedule :exec
DELETE FROM water_schedules WHERE id = ?;
