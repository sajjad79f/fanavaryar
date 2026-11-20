-- 0006: هم‌راستاسازی بک‌اند با قرارداد API نسخه ۱ (/api/v1) و فرانت‌اند کامل فناوریار
--
-- تغییرها:
--   ۱) کاربران: ستون نقش (admin/supervisor/user)، پرچم ابرمدیر و سمت سازمانی (job_title)
--   ۲) کاتالوگ مجوزها: کدهای مفرد/قدیمی به کدهای جمعِ موردانتظار فرانت‌اند تبدیل می‌شود
--      (history→audit، forms.create_template→forms.template_create و ...)
--   ۳) وضعیت‌های تجهیز هم‌سان فرانت‌اند: active | repair | storage | retired
--   ۴) ماژول فرم‌ها: قالب + نسخه‌بندی تغییرناپذیر (fields به‌صورت JSONB)
--   ۵) گزارش‌ها: کد گزارش، تاریخ گزارش، اتصال به نسخه فرم، form_data و چند تجهیزه
--      (چرخه: pending → approved | rejected)

-- ---------- ۱) کاربران ----------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user',
    ADD COLUMN IF NOT EXISTS is_superadmin BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS job_title TEXT;

UPDATE users SET job_title = position
WHERE job_title IS NULL AND position IS NOT NULL;

UPDATE users SET role = 'admin', is_superadmin = true WHERE username = 'admin';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'supervisor', 'user'));

-- ---------- ۲) کاتالوگ مجوزها ----------
-- تغییر نام ماژول‌ها/عملیات‌های قدیمی به کدهای جمعِ فرانت‌اند؛
-- شناسه‌ها حفظ می‌شوند تا ارجاع user_permissions و الگوها از بین نرود.
UPDATE permissions SET module = 'audit' WHERE module = 'history';
UPDATE permissions SET action = 'template_create' WHERE module = 'forms' AND action = 'create_template';
UPDATE permissions SET action = 'template_edit'   WHERE module = 'forms' AND action = 'edit_template';

-- کدهایی که در قرارداد v1 وجود ندارند (UI از آن‌ها استفاده نمی‌کند)
DELETE FROM permissions WHERE (module, action) IN (
    ('assets', 'approve'),
    ('reports', 'edit_draft'),
    ('reports', 'correction'),
    ('documents', 'view'), ('documents', 'download'), ('documents', 'sign'),
    ('schedules', 'view'), ('schedules', 'create'), ('schedules', 'edit'), ('schedules', 'cancel'),
    ('settings', 'view'), ('settings', 'change'),
    ('integrations', 'manage'),
    ('audit', 'export'),
    ('permissions', 'grant')
);

INSERT INTO permissions (module, action, title_fa) VALUES
    ('users', 'edit_permissions', 'تعیین مجوزهای کاربران')
ON CONFLICT (module, action) DO NOTHING;

-- عنوان‌های نمایشی یکدست
UPDATE permissions SET title_fa = 'مشاهده تجهیزات'        WHERE module='assets'   AND action='view';
UPDATE permissions SET title_fa = 'ثبت تجهیز'             WHERE module='assets'   AND action='create';
UPDATE permissions SET title_fa = 'ویرایش تجهیز'          WHERE module='assets'   AND action='edit';
UPDATE permissions SET title_fa = 'حذف تجهیز'             WHERE module='assets'   AND action='delete';
UPDATE permissions SET title_fa = 'مشاهده مکان‌ها'        WHERE module='locations' AND action='view';
UPDATE permissions SET title_fa = 'ایجاد مکان'            WHERE module='locations' AND action='create';
UPDATE permissions SET title_fa = 'ویرایش مکان'           WHERE module='locations' AND action='edit';
UPDATE permissions SET title_fa = 'حذف مکان'              WHERE module='locations' AND action='delete';
UPDATE permissions SET title_fa = 'مشاهده فرم‌ها'         WHERE module='forms'    AND action='view';
UPDATE permissions SET title_fa = 'ایجاد قالب فرم'        WHERE module='forms'    AND action='template_create';
UPDATE permissions SET title_fa = 'ویرایش قالب فرم'       WHERE module='forms'    AND action='template_edit';
UPDATE permissions SET title_fa = 'انتشار فرم'            WHERE module='forms'    AND action='publish';
UPDATE permissions SET title_fa = 'غیرفعال‌سازی فرم'      WHERE module='forms'    AND action='disable';
UPDATE permissions SET title_fa = 'مشاهده گزارش‌ها'       WHERE module='reports'  AND action='view';
UPDATE permissions SET title_fa = 'ثبت گزارش'             WHERE module='reports'  AND action='create';
UPDATE permissions SET title_fa = 'بررسی گزارش'           WHERE module='reports'  AND action='approve';
UPDATE permissions SET title_fa = 'مشاهده کاربران'        WHERE module='users'    AND action='view';
UPDATE permissions SET title_fa = 'ایجاد کاربر'           WHERE module='users'    AND action='create';
UPDATE permissions SET title_fa = 'ویرایش کاربر'          WHERE module='users'    AND action='edit';
UPDATE permissions SET title_fa = 'فعال/غیرفعال‌سازی'     WHERE module='users'    AND action='deactivate';
UPDATE permissions SET title_fa = 'تعیین مجوزهای کاربران' WHERE module='users'    AND action='edit_permissions';
UPDATE permissions SET title_fa = 'مشاهده دفتر رویدادها'  WHERE module='audit'    AND action='view';

-- ---------- ۳) وضعیت‌های تجهیز ----------
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_status_check;
UPDATE assets SET status = 'repair'  WHERE status = 'under_repair';
UPDATE assets SET status = 'retired' WHERE status = 'disposed';
ALTER TABLE assets ADD CONSTRAINT assets_status_check
    CHECK (status IN ('active', 'repair', 'storage', 'retired'));

ALTER TABLE assets RENAME COLUMN location_id TO current_location_id;

-- ---------- ۴) ماژول فرم‌ها ----------
CREATE TABLE form_templates (
    id                 BIGSERIAL PRIMARY KEY,
    code               TEXT NOT NULL UNIQUE,
    name               TEXT NOT NULL,
    description        TEXT,
    is_active          BOOLEAN NOT NULL DEFAULT true,
    current_version_id BIGINT,
    created_by         BIGINT REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE form_versions (
    id             BIGSERIAL PRIMARY KEY,
    template_id    BIGINT NOT NULL REFERENCES form_templates(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    fields         JSONB NOT NULL,
    created_by     BIGINT REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (template_id, version_number)
);

ALTER TABLE form_templates
    ADD CONSTRAINT form_templates_current_fk
    FOREIGN KEY (current_version_id) REFERENCES form_versions(id);

-- ---------- ۵) گزارش‌ها ----------
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_status_check;
UPDATE reports SET status = 'pending' WHERE status IN ('draft', 'submitted');
ALTER TABLE reports ADD CONSTRAINT reports_status_check
    CHECK (status IN ('pending', 'approved', 'rejected'));

ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS code TEXT UNIQUE,
    ADD COLUMN IF NOT EXISTS report_date DATE,
    ADD COLUMN IF NOT EXISTS form_template_version_id BIGINT REFERENCES form_versions(id),
    ADD COLUMN IF NOT EXISTS form_data JSONB,
    ADD COLUMN IF NOT EXISTS reject_reason TEXT;

UPDATE reports SET report_date = COALESCE(created_at::date, CURRENT_DATE) WHERE report_date IS NULL;
UPDATE reports SET form_data = jsonb_build_object('body', body) WHERE form_data IS NULL;
UPDATE reports SET code = 'R-OLD-' || lpad(id::text, 4, '0') WHERE code IS NULL;

-- اتصال چندتجهیزه گزارش‌ها؛ داده قدیمی از ستون asset_id منتقل می‌شود
CREATE TABLE report_assets (
    report_id BIGINT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    asset_id  BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    PRIMARY KEY (report_id, asset_id)
);
CREATE INDEX report_assets_asset_idx ON report_assets (asset_id);

INSERT INTO report_assets (report_id, asset_id)
SELECT id, asset_id FROM reports WHERE asset_id IS NOT NULL
ON CONFLICT DO NOTHING;
