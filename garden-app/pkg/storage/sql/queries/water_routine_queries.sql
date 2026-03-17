-- name: GetWaterRoutine :one
SELECT * FROM water_routines
WHERE id = ? LIMIT 1;

-- name: ListWaterRoutines :many
SELECT * FROM water_routines;

-- name: UpsertWaterRoutine :exec
INSERT INTO water_routines (
  id, name, steps
) VALUES (
  ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  steps = EXCLUDED.steps;

-- name: DeleteWaterRoutine :exec
DELETE FROM water_routines WHERE id = ?;
