-- Down migration to revert weather_control from WeatherScaler format back to ScaleControl format
-- New format: {client_id, interpolation, input_min, input_max, factor_min, factor_max}
-- Old format: {baseline_value, factor, range, client_id}

-- Revert rain_control: rain scales down only
-- baseline_value = input_min
-- range = input_max - input_min
-- factor = 1.0 - factor_min
UPDATE water_schedules
SET weather_control = json_set(
    weather_control,
    '$.rain_control',
    json_object(
        'client_id', json_extract(weather_control, '$.rain_control.client_id'),
        'baseline_value', json_extract(weather_control, '$.rain_control.input_min'),
        'range', json_extract(weather_control, '$.rain_control.input_max') - json_extract(weather_control, '$.rain_control.input_min'),
        'factor', 1.0 - json_extract(weather_control, '$.rain_control.factor_min')
    )
)
WHERE json_extract(weather_control, '$.rain_control') IS NOT NULL
  AND json_extract(weather_control, '$.rain_control.input_min') IS NOT NULL;

-- Revert temperature_control: temperature scales up and down
-- baseline_value = (input_min + input_max) / 2
-- range = (input_max - input_min) / 2
-- factor = factor_max - 1.0
UPDATE water_schedules
SET weather_control = json_set(
    weather_control,
    '$.temperature_control',
    json_object(
        'client_id', json_extract(weather_control, '$.temperature_control.client_id'),
        'baseline_value', (json_extract(weather_control, '$.temperature_control.input_min') + json_extract(weather_control, '$.temperature_control.input_max')) / 2.0,
        'range', (json_extract(weather_control, '$.temperature_control.input_max') - json_extract(weather_control, '$.temperature_control.input_min')) / 2.0,
        'factor', json_extract(weather_control, '$.temperature_control.factor_max') - 1.0
    )
)
WHERE json_extract(weather_control, '$.temperature_control') IS NOT NULL
  AND json_extract(weather_control, '$.temperature_control.input_min') IS NOT NULL;
