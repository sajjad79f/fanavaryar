-- 0001_init — baseline یکپارچه اسکیمای فناوریار (نسخه فشرده‌شده مایگریشن‌های ۰۰۰۱ تا ۰۰۰۹)
--
-- این فایل وضعیت نهایی و تمیز دیتابیس را از صفر می‌سازد و جایگزین تاریخچه
-- مایگریشن‌های قبلی است. همه ستون‌ها با شکل نهایی‌شان ساخته می‌شوند (بدون
-- ALTERهای میانی)، کاتالوگ مجوزها با کدهای قرارداد v1 و عنوان‌های فارسی
-- یکدست، و الگوهای دسترسی آماده.
--
-- فیچرهای جدید هم‌زمان با baseline:
--   ۱) form_templates.kind — نوع فرم: گزارش فعالیت یا مشخصات سخت‌افزاری تجهیز
--   ۲) assets.spec_form_version_id / assets.spec_data — مشخصات فرمی تجهیز
--      متصل به نسخه‌ای تغییرناپذیر از فرم (همان الگوی گزارش‌ها)
--
-- دیتابیس باید از صفر ساخته شود (schema خالی)؛ اجرای این فایل روی دیتابیس
-- دارای schema_migrations قدیمی باعث ردشدن بی‌صدای مایگریشن می‌شود.

-- ---------- کاربران و مجوزها ----------

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    full_name     TEXT        NOT NULL,
    position      TEXT,                        -- سمت سازمانی قدیمی؛ توصیفی و بدون مجوز
    password_hash TEXT        NOT NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    scope_type    TEXT        NOT NULL DEFAULT 'all'
                  CHECK (scope_type IN ('all', 'building', 'floor', 'room', 'assigned')),
    scope_ref_id  BIGINT,                      -- مرجع محدوده اطلاعاتی (اختیاری)
    role          TEXT        NOT NULL DEFAULT 'user'
                  CHECK (role IN ('admin', 'supervisor', 'user')),
    is_superadmin BOOLEAN     NOT NULL DEFAULT false,
    job_title     TEXT,                        -- سمت سازمانی نمایشی (قرارداد v1)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id       SERIAL PRIMARY KEY,
    module   TEXT NOT NULL,          -- assets, locations, forms, ...
    action   TEXT NOT NULL,          -- view, create, edit, ...
    title_fa TEXT NOT NULL,          -- نام نمایشی فارسی
    UNIQUE (module, action)
);

CREATE TABLE user_permissions (
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id INT    NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by    BIGINT REFERENCES users(id),
    granted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, permission_id)
);

-- الگوهای دسترسی اختیاری: صرفاً مجموعه آماده مجوز برای تسریع تخصیص
CREATE TABLE permission_templates (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permission_template_items (
    template_id   INT NOT NULL REFERENCES permission_templates(id) ON DELETE CASCADE,
    permission_id INT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (template_id, permission_id)
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,        -- فقط SHA-256 توکن؛ خود توکن ذخیره نمی‌شود
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip         TEXT
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE audit_log (
    id             BIGSERIAL PRIMARY KEY,
    actor_id       BIGINT,        -- بدون FK؛ سوابق حتی پس از حذف حساب نگه‌داری می‌شود
    actor_username TEXT,
    module         TEXT NOT NULL,
    action         TEXT NOT NULL,
    entity_type    TEXT NOT NULL,
    entity_id      TEXT,
    before_data    JSONB,
    after_data     JSONB,
    ip             TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_module_idx    ON audit_log (module, created_at);
CREATE INDEX audit_log_entity_idx    ON audit_log (entity_type, entity_id);
CREATE INDEX audit_log_actor_idx     ON audit_log (actor_id);

-- ---------- مکان‌ها ----------

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

-- ---------- فرم‌ها (قالب + نسخه‌بندی تغییرناپذیر) ----------

CREATE TABLE form_templates (
    id                 BIGSERIAL PRIMARY KEY,
    code               TEXT NOT NULL UNIQUE,   -- انگلیسی؛ حروف بزرگ/کوچک، رقم، خط تیره و زیرخط
    name               TEXT NOT NULL,
    description        TEXT,
    kind               TEXT NOT NULL DEFAULT 'report'
                       CHECK (kind IN ('report', 'hardware_spec')),  -- گزارش فعالیت یا مشخصات سخت‌افزاری
    is_active          BOOLEAN NOT NULL DEFAULT true,
    requires_approval  BOOLEAN NOT NULL DEFAULT true,  -- false = گزارش‌ها بدون تایید مدیر نهایی می‌شوند
    current_version_id BIGINT,
    created_by         BIGINT REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE form_versions (
    id             BIGSERIAL PRIMARY KEY,
    template_id    BIGINT NOT NULL REFERENCES form_templates(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    fields         JSONB NOT NULL,           -- آرایه FieldDef
    created_by     BIGINT REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (template_id, version_number)
);

ALTER TABLE form_templates
    ADD CONSTRAINT form_templates_current_fk
    FOREIGN KEY (current_version_id) REFERENCES form_versions(id);

-- ---------- تجهیزات ----------

CREATE TABLE assets (
    id                   BIGSERIAL PRIMARY KEY,
    asset_tag            TEXT NOT NULL UNIQUE,      -- شناسه تجهیز؛ مثل IT-PC-0142
    name                 TEXT NOT NULL,
    category             TEXT NOT NULL,             -- server, network, pc, printer, ...
    status               TEXT NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'repair', 'storage', 'retired')),
    serial_number        TEXT,
    current_location_id  BIGINT REFERENCES locations(id) ON DELETE SET NULL,
    manufacturer         TEXT,
    model                TEXT,
    responsible_person   TEXT,
    notes                TEXT,
    -- مشخصات فرمی: اتصال به نسخه‌ای تغییرناپذیر از فرم مشخصات سخت‌افزاری
    spec_form_version_id BIGINT REFERENCES form_versions(id),
    spec_data            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX assets_location_idx ON assets (current_location_id);
CREATE INDEX assets_status_idx   ON assets (status);
CREATE INDEX assets_category_idx ON assets (category);
CREATE INDEX assets_spec_idx     ON assets (spec_form_version_id);

-- تاریخچه جابه‌جایی تجهیزات
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

-- رویدادهای چرخه عمر تجهیز
CREATE TABLE asset_events (
    id         BIGSERIAL PRIMARY KEY,
    asset_id   BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,               -- created, updated, status_changed, moved
    details    JSONB,
    actor_id   BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX asset_events_asset_idx ON asset_events (asset_id, created_at);

-- ---------- گزارش‌ها ----------

-- چرخه وضعیت: pending → approved | rejected
-- فرم‌های دارای requires_approval=false در لحظه ثبت مستقیم approved می‌شوند.
CREATE TABLE reports (
    id                      BIGSERIAL PRIMARY KEY,
    asset_id                BIGINT REFERENCES assets(id),   -- ستون قدیمی؛ مرجع فعال report_assets است
    title                   TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'approved', 'rejected')),
    author_id               BIGINT NOT NULL REFERENCES users(id),
    approved_by             BIGINT REFERENCES users(id),
    approved_at             TIMESTAMPTZ,
    code                    TEXT UNIQUE,                    -- مثل R-20260101-0007
    report_date             DATE,
    form_template_version_id BIGINT REFERENCES form_versions(id),
    form_data               JSONB,
    reject_reason           TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reports_asset_idx  ON reports (asset_id, created_at);
CREATE INDEX reports_status_idx ON reports (status);
CREATE INDEX reports_author_idx ON reports (author_id);

-- اتصال چندتجهیزه گزارش‌ها
CREATE TABLE report_assets (
    report_id BIGINT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    asset_id  BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    PRIMARY KEY (report_id, asset_id)
);
CREATE INDEX report_assets_asset_idx ON report_assets (asset_id);

-- ---------- کاتالوگ مجوزها (قرارداد v1) ----------

INSERT INTO permissions (module, action, title_fa) VALUES
    ('assets',       'view',             'مشاهده تجهیزات'),
    ('assets',       'create',           'ثبت تجهیز'),
    ('assets',       'edit',             'ویرایش تجهیز'),
    ('assets',       'delete',           'حذف تجهیز'),
    ('locations',    'view',             'مشاهده مکان‌ها'),
    ('locations',    'create',           'ایجاد مکان'),
    ('locations',    'edit',             'ویرایش مکان'),
    ('locations',    'delete',           'حذف مکان'),
    ('forms',        'view',             'مشاهده فرم‌ها'),
    ('forms',        'template_create',  'ایجاد قالب فرم'),
    ('forms',        'template_edit',    'ویرایش قالب فرم'),
    ('forms',        'publish',          'انتشار فرم'),
    ('forms',        'disable',          'غیرفعال‌سازی فرم'),
    ('reports',      'view',             'مشاهده گزارش‌ها'),
    ('reports',      'create',           'ثبت گزارش'),
    ('reports',      'approve',          'بررسی گزارش'),
    ('users',        'view',             'مشاهده کاربران'),
    ('users',        'create',           'ایجاد کاربر'),
    ('users',        'edit',             'ویرایش کاربر'),
    ('users',        'deactivate',       'فعال/غیرفعال‌سازی'),
    ('users',        'edit_permissions', 'تعیین مجوزهای کاربران'),
    ('audit',        'view',             'مشاهده دفتر رویدادها');

-- ---------- الگوهای دسترسی آماده ----------

INSERT INTO permission_templates (name, description) VALUES
    ('network_expert', 'الگوی کارشناس شبکه'),
    ('support_expert', 'الگوی کارشناس پشتیبانی'),
    ('sysadmin',       'الگوی مدیر سامانه — همه مجوزها');

INSERT INTO permission_template_items (template_id, permission_id)
SELECT t.id, p.id FROM permission_templates t, permissions p
WHERE t.name = 'sysadmin';

INSERT INTO permission_template_items (template_id, permission_id)
SELECT t.id, p.id FROM permission_templates t, permissions p
WHERE t.name = 'network_expert'
  AND (p.module, p.action) IN (
      ('assets','view'), ('assets','create'), ('assets','edit'),
      ('locations','view'), ('locations','edit'),
      ('reports','view'), ('reports','create'),
      ('audit','view')
  );

INSERT INTO permission_template_items (template_id, permission_id)
SELECT t.id, p.id FROM permission_templates t, permissions p
WHERE t.name = 'support_expert'
  AND (p.module, p.action) IN (
      ('assets','view'), ('locations','view'),
      ('reports','view'), ('reports','create'),
      ('audit','view')
  );
