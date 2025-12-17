-- Add status column to items for listing state control
ALTER TABLE items
ADD COLUMN status ENUM('listed','paused','sold') NOT NULL DEFAULT 'listed' AFTER price;

-- Backfill existing rows to listed
UPDATE items SET status = 'listed' WHERE status IS NULL OR status = '';
