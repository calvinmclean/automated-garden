-- Add unique constraint on topic_prefix to ensure no duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_gardens_topic_prefix ON gardens(topic_prefix);
