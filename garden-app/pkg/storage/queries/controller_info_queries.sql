-- name: UpsertControllerInfo :exec
INSERT INTO garden_controller_info (garden_id, mac_address, ip_address, firmware_version, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (garden_id)
DO UPDATE SET
    mac_address = EXCLUDED.mac_address,
    ip_address = EXCLUDED.ip_address,
    firmware_version = EXCLUDED.firmware_version,
    updated_at = EXCLUDED.updated_at;

-- name: GetControllerInfo :one
SELECT * FROM garden_controller_info WHERE garden_id = ? LIMIT 1;

-- name: DeleteControllerInfo :exec
DELETE FROM garden_controller_info WHERE garden_id = ?;
