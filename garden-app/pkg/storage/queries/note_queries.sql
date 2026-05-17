-- name: GetNote :one
SELECT * FROM notes
WHERE id = ? LIMIT 1;

-- name: ListNotes :many
SELECT * FROM notes;

-- name: UpsertNote :exec
INSERT INTO notes (
  id, title, content,
  created_at,
  garden_id, zone_id
) VALUES (
  ?, ?, ?, ?, ?, ?
) ON CONFLICT (id)
DO UPDATE SET
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  garden_id = EXCLUDED.garden_id,
  zone_id = EXCLUDED.zone_id;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = ?;
