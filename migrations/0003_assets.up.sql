-- 0003: ماژول Assets — پرونده تجهیزات و تاریخچه جابه‌جایی

CREATE TABLE assets (
    id            BIGSERIAL PRIMARY KEY,
    asset_tag     TEXT NOT NULL UNIQUE,      -- شناسه تجهیز؛ مثل IT-PC-0142
    name          TEXT NOT NULL,
    category      TEXT NOT NULL,             -- server, network, pc, printer, ...
    status        TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'under_repair', 'retired', 'disposed')),
    serial_number TEXT,
    location_id   BIGINT REFERENCES locations(id) ON DELETE SET NULL,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX assets_location_idx ON assets (location_id);
CREATE INDEX assets_status_idx   ON assets (status);
CREATE INDEX assets_category_idx ON assets (category);

-- تاریخچه جابه‌جایی تجهیزات (الزام سند: ماژول Locations)
CREATE TABLE asset_moves (
    id               BIGSERIAL PRIMARY KEY,
    asset_id         BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    from_location_id BIGINT REFERENCES locations(id),
    to_location_id   BIGINT REFERENCES locations(id),
    reason           TEXT,
    moved_by         BIGINT REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX asset_moves_asset_idx ON asset_moves (asset_id, created_at);

-- رویدادهای چرخه عمر تجهیز؛ مبنای ماژول History یکپارچه در فاز ۲
CREATE TABLE asset_events (
    id         BIGSERIAL PRIMARY KEY,
    asset_id   BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,               -- created, updated, status_changed, moved
    details    JSONB,
    actor_id   BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX asset_events_asset_idx ON asset_events (asset_id, created_at);
