-- Convert the legacy single temperature/humidity sensor configuration into the
-- new generic sensors array. The array index is used as the sensor_id in MQTT.
UPDATE gardens
SET controller_config = json_set(
    json_remove(
        ifnull(controller_config, '{}'),
        '$.temperature_humidity_pin',
        '$.temperature_humidity_interval'
    ),
    '$.sensors',
    json_array(
        json_object(
            'name', 'Ambient',
            'type', 'DHT22',
            'pin', json_extract(controller_config, '$.temperature_humidity_pin'),
            'interval', coalesce(json_extract(controller_config, '$.temperature_humidity_interval'), '5s')
        )
    )
)
WHERE temp_humid_sensor = TRUE
  AND json_extract(controller_config, '$.temperature_humidity_pin') IS NOT NULL;

-- Drop the now-obsolete temp_humid_sensor column.
ALTER TABLE gardens DROP COLUMN temp_humid_sensor;
