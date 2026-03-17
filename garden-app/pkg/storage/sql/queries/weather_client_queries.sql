-- name: GetWeatherClient :one
SELECT * FROM weather_clients
WHERE id = ? LIMIT 1;

-- name: ListWeatherClients :many
SELECT * FROM weather_clients;

-- name: UpsertWeatherClient :exec
INSERT INTO weather_clients (
  id, type, options
) VALUES (
  ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  type = EXCLUDED.type,
  options = EXCLUDED.options;

-- name: DeleteWeatherClient :exec
DELETE FROM weather_clients WHERE id = ?;
