-- 0002: ماژول Locations — ساختمان، طبقه، اتاق و رک
-- سلسله‌مراتب دقیق: ساختمان ← طبقه ← اتاق ← رک

CREATE TABLE locations (
    id         BIGSERIAL PRIMARY KEY,
    parent_id  BIGINT REFERENCES locations(id) ON DELETE RESTRICT,
    kind       TEXT NOT NULL CHECK (kind IN ('building', 'floor', 'room', 'rack')),
    name       TEXT NOT NULL,
    code       TEXT UNIQUE,                 -- شناسه اختصاری اختیاری؛ مثلاً BLD-01
    notes      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX locations_parent_idx ON locations (parent_id);
CREATE INDEX locations_kind_idx   ON locations (kind);

-- قواعد ساختار در سطح دیتابیس (علاوه بر اعتبارسنجی در لایه Domain):
-- ریشه‌ها فقط ساختمان هستند
ALTER TABLE locations ADD CONSTRAINT locations_root_is_building
    CHECK (parent_id IS NOT NULL OR kind = 'building');
