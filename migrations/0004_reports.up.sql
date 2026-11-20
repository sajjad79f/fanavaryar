-- 0004: ماژول Reports — ثبت فعالیت‌ها و گزارش‌های تجهیزات
-- چرخه وضعیت: draft → submitted → approved
-- گزارش تأییدشده از مسیر API عادی قابل ویرایش نیست؛ اصلاح با گزارش جدید انجام می‌شود.

CREATE TABLE reports (
    id          BIGSERIAL PRIMARY KEY,
    asset_id    BIGINT REFERENCES assets(id),
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'submitted', 'approved')),
    author_id   BIGINT NOT NULL REFERENCES users(id),
    approved_by BIGINT REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reports_asset_idx  ON reports (asset_id, created_at);
CREATE INDEX reports_status_idx ON reports (status);
CREATE INDEX reports_author_idx ON reports (author_id);
