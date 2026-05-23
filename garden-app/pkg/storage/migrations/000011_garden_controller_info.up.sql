CREATE TABLE IF NOT EXISTS garden_controller_info (
    garden_id VARCHAR(20) PRIMARY KEY,
    mac_address TEXT,
    ip_address TEXT,
    firmware_version TEXT,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (garden_id) REFERENCES gardens(id) ON DELETE CASCADE
);
