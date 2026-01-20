-- Migration: Create event_images table
-- One-to-many relationship with events table for storing multiple event images

CREATE TABLE event_images (
    id VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
    event_id VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
    url VARCHAR(500) COLLATE utf8mb4_unicode_ci NOT NULL,
    caption VARCHAR(255) COLLATE utf8mb4_unicode_ci,
    alt_text VARCHAR(255) COLLATE utf8mb4_unicode_ci,
    display_order INT DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_event_images_event (event_id),
    INDEX idx_event_images_order (event_id, display_order),
    CONSTRAINT fk_event_images_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
