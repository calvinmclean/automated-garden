-- Migration to convert weather_control from ScaleControl format to WeatherScaler format
-- Old format: {baseline_value, factor, range, client_id}
-- New format: {client_id, interpolation, input_min, input_max, factor_min, factor_max}

-- Update rain_control: rain scales down only
-- input_min = baseline_value
-- input_max = baseline_value + range
-- factor_min = 1.0 - factor
-- factor_max = 1.0
UPDATE water_schedules
SET weather_control = json_set(
    weather_control,
    '$.rain_control',
    json_object(
        'client_id', json_extract(weather_control, '$.rain_control.client_id'),
        'interpolation', 'linear',
        'input_min', json_extract(weather_control, '$.rain_control.baseline_value'),
        'input_max', json_extract(weather_control, '$.rain_control.baseline_value') + json_extract(weather_control, '$.rain_control.range'),
        'factor_min', 1.0 - json_extract(weather_control, '$.rain_control.factor'),
        'factor_max', 1.0
    )
)
WHERE json_extract(weather_control, '$.rain_control') IS NOT NULL
  AND json_extract(weather_control, '$.rain_control.baseline_value') IS NOT NULL;

-- Update temperature_control: temperature scales up and down
-- input_min = baseline_value - range
-- input_max = baseline_value + range
-- factor_min = 1.0 - factor
-- factor_max = 1.0 + factor
UPDATE water_schedules
SET weather_control = json_set(
    weather_control,
    '$.temperature_control',
    json_object(
        'client_id', json_extract(weather_control, '$.temperature_control.client_id'),
        'interpolation', 'linear',
        'input_min', json_extract(weather_control, '$.temperature_control.baseline_value') - json_extract(weather_control, '$.temperature_control.range'),
        'input_max', json_extract(weather_control, '$.temperature_control.baseline_value') + json_extract(weather_control, '$.temperature_control.range'),
        'factor_min', 1.0 - json_extract(weather_control, '$.temperature_control.factor'),
        'factor_max', 1.0 + json_extract(weather_control, '$.temperature_control.factor')
    )
)
WHERE json_extract(weather_control, '$.temperature_control') IS NOT NULL
  AND json_extract(weather_control, '$.temperature_control.baseline_value') IS NOT NULL;
