CREATE TABLE IF NOT EXISTS notes (
    id VARCHAR(20) PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    created_at DATETIME NOT NULL,
    garden_id VARCHAR(20),
    zone_id VARCHAR(20),
    FOREIGN KEY (garden_id) REFERENCES gardens(id),
    FOREIGN KEY (zone_id) REFERENCES zones(id)
);
