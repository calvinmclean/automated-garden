-- name: GetZone :one
SELECT * FROM zones
WHERE id = ? LIMIT 1;

-- name: ListZones :many
SELECT * FROM zones WHERE garden_id = ?;

-- name: UpsertZone :exec
INSERT INTO zones (
  id, name, garden_id,
  details_description, details_notes,
  position, skip_count,
  created_at, end_date,
  water_schedule_ids
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  details_description = EXCLUDED.details_description,
  details_notes = EXCLUDED.details_notes,
  position = EXCLUDED.position,
  skip_count = EXCLUDED.skip_count,
  end_date = EXCLUDED.end_date,
  water_schedule_ids = EXCLUDED.water_schedule_ids;

-- name: SetZoneEndDate :exec
UPDATE zones
SET end_date = ?
WHERE id = ?;

-- name: FindZonesByWaterScheduleID :many
SELECT *
FROM zones
WHERE CONCAT(',', water_schedule_ids, ',') LIKE CONCAT('%,', ?, ',%');

-- name: DeleteZone :exec
DELETE FROM zones WHERE id = ?;
