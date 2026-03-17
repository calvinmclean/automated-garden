CREATE TABLE IF NOT EXISTS gardens (
    id VARCHAR(20) PRIMARY KEY,
    name TEXT NOT NULL,
    topic_prefix TEXT NOT NULL,

    max_zones INT NOT NULL,
    temp_humid_sensor BOOL NOT NULL DEFAULT FALSE,

    created_at DATETIME NOT NULL,
    end_date DATETIME,

    notification_client_id VARCHAR(20),
    notification_settings TEXT, -- JSON
    controller_config TEXT, -- JSON
    light_schedule TEXT, -- JSON
    FOREIGN KEY (notification_client_id) REFERENCES notification_clients(id)
);

CREATE TABLE IF NOT EXISTS zones (
    id VARCHAR(20) PRIMARY KEY,
    name TEXT NOT NULL,
    garden_id VARCHAR(20) NOT NULL,

    details_description TEXT,
    details_notes TEXT,

    position INT,
    skip_count INT,

    created_at DATETIME NOT NULL,
    end_date DATETIME,

    -- comma-separated list of IDs
    water_schedule_ids TEXT,

    FOREIGN KEY (garden_id) REFERENCES gardens(id)
);

CREATE TABLE IF NOT EXISTS water_schedules (
    id VARCHAR(20) PRIMARY KEY,
    name TEXT,
    description TEXT,

    duration INT NOT NULL,
    interval INT NOT NULL,

    start_date DATETIME NOT NULL,
    start_time VARCHAR(14) NOT NULL, -- Format: 15:04:05Z07:00

    end_date DATETIME,

    active_period_start_month TEXT,
    active_period_end_month TEXT,

    weather_control TEXT, -- JSON

    notification_client_id VARCHAR(20),
    FOREIGN KEY (notification_client_id) REFERENCES notification_clients(id)
);

CREATE TABLE IF NOT EXISTS weather_clients (
    id VARCHAR(20) PRIMARY KEY,
    type TEXT NOT NULL,
    options JSON NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_clients (
    id VARCHAR(20) PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    options JSON NOT NULL
);

CREATE TABLE IF NOT EXISTS water_routines (
    id VARCHAR(20) PRIMARY KEY,
    name TEXT NOT NULL,
    steps JSON NOT NULL
);
