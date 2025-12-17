CREATE TABLE IF NOT EXISTS notifications (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_uid VARCHAR(128) NOT NULL,
    type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT,
    item_id BIGINT UNSIGNED NULL,
    conversation_id BIGINT UNSIGNED NULL,
    purchase_id BIGINT UNSIGNED NULL,
    read_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_notifications_user_uid (user_uid),
    INDEX idx_notifications_item_id (item_id),
    INDEX idx_notifications_conv_id (conversation_id),
    INDEX idx_notifications_purchase_id (purchase_id),
    INDEX idx_notifications_read (user_uid, read_at)
);
