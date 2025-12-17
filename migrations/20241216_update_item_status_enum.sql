-- Extend item status to include in_transaction
ALTER TABLE items
MODIFY status ENUM('listed','paused','in_transaction','sold') NOT NULL DEFAULT 'listed';
