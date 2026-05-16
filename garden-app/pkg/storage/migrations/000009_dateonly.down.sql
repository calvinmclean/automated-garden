-- Revert StartDate from DateOnly back to RFC3339 format
-- This restores the original timestamp format

-- Update water_schedules table
UPDATE water_schedules
SET start_date = start_date || 'T00:00:00Z'
WHERE start_date != '' AND length(start_date) = 10;
