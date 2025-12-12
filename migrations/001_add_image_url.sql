-- Add image_url column to items for storing hosted image URLs
ALTER TABLE items
    ADD COLUMN IF NOT EXISTS image_url VARCHAR(512) NULL AFTER price;
