-- Convert StartDate from RFC3339 to DateOnly format (YYYY-MM-DD)
-- This ensures dates are stored without time components, making timezone handling consistent

-- Update water_schedules table
UPDATE water_schedules
SET start_date = substr(start_date, 1, 10)
WHERE start_date != '' AND length(start_date) > 10;
