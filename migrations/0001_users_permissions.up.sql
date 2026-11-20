-- 0001: هسته کاربران، مجوزها، نشست‌ها و Audit Log
-- (فاز ۱ سند معماری — ماژول Users & Permissions)

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    full_name     TEXT        NOT NULL,
    position      TEXT,                        -- سمت سازمانی؛ توصیفی و بدون مجوز
    password_hash TEXT        NOT NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    scope_type    TEXT        NOT NULL DEFAULT 'all'
                  CHECK (scope_type IN ('all', 'building', 'floor', 'room', 'assigned')),
    scope_ref_id  BIGINT,                      -- مرجع محدوده اطلاعاتی (اختیاری)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id       SERIAL PRIMARY KEY,
    module   TEXT NOT NULL,          -- assets, locations, users, ...
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
    description TEXT
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

-- کاتالوگ مجوزها مطابق جدول بخش ۵ سند معماری
INSERT INTO permissions (module, action, title_fa) VALUES
    ('assets',       'view',        'مشاهده تجهیزات'),
    ('assets',       'create',      'ایجاد تجهیز'),
    ('assets',       'edit',        'ویرایش تجهیز'),
    ('assets',       'delete',      'حذف تجهیز'),
    ('assets',       'approve',     'تأیید تجهیز'),
    ('locations',    'view',        'مشاهده مکان‌ها'),
    ('locations',    'create',      'ایجاد مکان'),
    ('locations',    'edit',        'ویرایش مکان'),
    ('locations',    'delete',      'حذف مکان'),
    ('forms',        'view',            'مشاهده فرم‌ها'),
    ('forms',        'create_template', 'ایجاد قالب فرم'),
    ('forms',        'edit_template',   'ویرایش قالب فرم'),
    ('forms',        'publish',         'انتشار فرم'),
    ('forms',        'disable',         'غیرفعال‌سازی فرم'),
    ('reports',      'view',        'مشاهده گزارش‌ها'),
    ('reports',      'create',      'ثبت گزارش'),
    ('reports',      'edit_draft',  'ویرایش پیش‌نویس'),
    ('reports',      'approve',     'تأیید گزارش'),
    ('reports',      'correction',  'ثبت اصلاحیه'),
    ('documents',    'view',        'مشاهده اسناد'),
    ('documents',    'download',    'دانلود اسناد'),
    ('documents',    'sign',        'امضای اسناد'),
    ('history',      'view',        'مشاهده تاریخچه'),
    ('history',      'export',      'دریافت خروجی تاریخچه'),
    ('schedules',    'view',        'مشاهده برنامه‌ریزی'),
    ('schedules',    'create',      'ایجاد فعالیت'),
    ('schedules',    'edit',        'ویرایش فعالیت'),
    ('schedules',    'cancel',      'لغو فعالیت'),
    ('users',        'view',        'مشاهده کاربران'),
    ('users',        'create',      'ایجاد کاربر'),
    ('users',        'edit',        'ویرایش کاربر'),
    ('users',        'deactivate',  'غیرفعال‌سازی کاربر'),
    ('permissions',  'grant',       'تخصیص مجوزها'),
    ('settings',     'view',        'مشاهده تنظیمات'),
    ('settings',     'change',      'تغییر تنظیمات'),
    ('integrations', 'manage',      'مدیریت اتصال‌ها');

-- الگوهای دسترسی آماده (اختیاری؛ جایگزین مجوز نهایی کاربر نمی‌شوند)
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
      ('reports','view'), ('reports','create'), ('reports','edit_draft'),
      ('history','view')
  );

INSERT INTO permission_template_items (template_id, permission_id)
SELECT t.id, p.id FROM permission_templates t, permissions p
WHERE t.name = 'support_expert'
  AND (p.module, p.action) IN (
      ('assets','view'), ('locations','view'),
      ('reports','view'), ('reports','create'),
      ('history','view')
  );
