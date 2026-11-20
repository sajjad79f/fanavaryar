-- 0006 (down): بازگشت حداقلی — جدول‌های جدید حذف و وضعیت‌های قدیمی بازیابی می‌شود
-- توجه: این migration فقط برای محیط‌های آزمایشی است؛ داده فرم‌ها از بین می‌رود.
DROP TABLE IF EXISTS report_assets;
DROP TABLE IF EXISTS form_versions;
DROP TABLE IF EXISTS form_templates;

ALTER TABLE reports
    DROP COLUMN IF EXISTS code,
    DROP COLUMN IF EXISTS report_date,
    DROP COLUMN IF EXISTS form_template_version_id,
    DROP COLUMN IF EXISTS form_data,
    DROP COLUMN IF EXISTS reject_reason;
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_status_check;
ALTER TABLE reports ADD CONSTRAINT reports_status_check
    CHECK (status IN ('draft', 'submitted', 'approved'));

ALTER TABLE assets RENAME COLUMN current_location_id TO location_id;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_status_check;
ALTER TABLE assets ADD CONSTRAINT assets_status_check
    CHECK (status IN ('active', 'under_repair', 'retired', 'disposed'));

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_check,
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS is_superadmin,
    DROP COLUMN IF EXISTS job_title;
