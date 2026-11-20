ALTER TABLE assets
    DROP COLUMN IF EXISTS responsible_person,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS manufacturer;
