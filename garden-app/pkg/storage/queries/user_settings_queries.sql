-- name: GetUserSetting :one
SELECT * FROM user_settings WHERE key = ? LIMIT 1;

-- name: UpsertUserSetting :exec
INSERT INTO user_settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: DeleteUserSetting :exec
DELETE FROM user_settings WHERE key = ?;
