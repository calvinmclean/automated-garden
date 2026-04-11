-- name: GetNotificationClient :one
SELECT * FROM notification_clients
WHERE id = ? LIMIT 1;

-- name: ListNotificationClients :many
SELECT * FROM notification_clients;

-- name: UpsertNotificationClient :exec
INSERT INTO notification_clients (
  id, name, url
) VALUES (
  ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  name = EXCLUDED.name,
  url = EXCLUDED.url;

-- name: DeleteNotificationClient :exec
DELETE FROM notification_clients WHERE id = ?;
