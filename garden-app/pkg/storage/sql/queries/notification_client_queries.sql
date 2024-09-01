-- name: GetNotificationClient :one
SELECT * FROM notification_clients
WHERE id = ? LIMIT 1;

-- name: ListNotificationClients :many
SELECT * FROM notification_clients;

-- name: UpsertNotificationClient :exec
INSERT INTO notification_clients (
  id, name, type, options
) VALUES (
  ?, ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  type = EXCLUDED.type,
  options = EXCLUDED.options;

-- name: DeleteNotificationClient :exec
DELETE FROM notification_clients WHERE id = ?;
