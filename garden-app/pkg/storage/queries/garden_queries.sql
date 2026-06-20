-- name: GetGarden :one
SELECT g.*, ci.mac_address, ci.ip_address, ci.firmware_version, ci.updated_at
FROM gardens g
LEFT JOIN garden_controller_info ci ON g.id = ci.garden_id
WHERE g.id = ? LIMIT 1;

-- name: ListAllGardens :many
SELECT g.*, ci.mac_address, ci.ip_address, ci.firmware_version, ci.updated_at
FROM gardens g
LEFT JOIN garden_controller_info ci ON g.id = ci.garden_id;

-- name: ListActiveGardens :many
SELECT g.*, ci.mac_address, ci.ip_address, ci.firmware_version, ci.updated_at
FROM gardens g
LEFT JOIN garden_controller_info ci ON g.id = ci.garden_id
WHERE g.end_date IS NULL
   OR g.end_date > ?;

-- name: UpsertGarden :exec
INSERT INTO gardens (
  id, name, topic_prefix,
  max_zones,
  created_at, end_date,
  notification_client_id, notification_settings,
  controller_config, light_schedule, fan_schedule
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  topic_prefix = EXCLUDED.topic_prefix,
  max_zones = EXCLUDED.max_zones,
  end_date = EXCLUDED.end_date,
  notification_client_id = EXCLUDED.notification_client_id,
  notification_settings = EXCLUDED.notification_settings,
  controller_config = EXCLUDED.controller_config,
  light_schedule = EXCLUDED.light_schedule,
  fan_schedule = EXCLUDED.fan_schedule;

-- name: SetGardenEndDate :exec
UPDATE gardens
SET end_date = ?
WHERE id = ?;

-- name: DeleteGarden :exec
DELETE FROM gardens WHERE id = ?;

-- name: GetGardenByTopicPrefix :one
SELECT g.*, ci.mac_address, ci.ip_address, ci.firmware_version, ci.updated_at
FROM gardens g
LEFT JOIN garden_controller_info ci ON g.id = ci.garden_id
WHERE g.topic_prefix = ? LIMIT 1;
