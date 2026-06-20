-- Best-effort rollback: convert the first sensor back to the legacy
-- temperature/humidity config fields.
UPDATE gardens
SET controller_config = json_set(
    json_remove(controller_config, '$.sensors'),
    '$.temperature_humidity_pin', json_extract(controller_config, '$.sensors[0].pin'),
    '$.temperature_humidity_interval', json_extract(controller_config, '$.sensors[0].interval')
)
WHERE json_array_length(json_extract(controller_config, '$.sensors')) > 0;

-- Restore the temp_humid_sensor column.
ALTER TABLE gardens ADD COLUMN temp_humid_sensor BOOL NOT NULL DEFAULT FALSE;
