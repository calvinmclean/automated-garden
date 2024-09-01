-- name: GetGarden :one
SELECT * FROM gardens
WHERE id = ? LIMIT 1;

-- name: ListGardens :many
SELECT * FROM gardens;

-- name: UpsertGarden :exec
INSERT INTO gardens (
  id, name, topic_prefix,
  max_zones, temp_humid_sensor,
  created_at, end_date,
  notification_client_id, notification_settings,
  controller_config, light_schedule
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  topic_prefix = EXCLUDED.topic_prefix,
  max_zones = EXCLUDED.max_zones,
  temp_humid_sensor = EXCLUDED.temp_humid_sensor,
  end_date = EXCLUDED.end_date,
  notification_client_id = EXCLUDED.notification_client_id,
  notification_settings = EXCLUDED.notification_settings,
  controller_config = EXCLUDED.controller_config,
  light_schedule = EXCLUDED.light_schedule;

-- name: SetGardenEndDate :exec
UPDATE gardens
SET end_date = ?
WHERE id = ?;

-- name: DeleteGarden :exec
DELETE FROM gardens WHERE id = ?;
