CREATE TABLE IF NOT EXISTS user_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO user_settings (key, value) VALUES ('units', 'metric');
