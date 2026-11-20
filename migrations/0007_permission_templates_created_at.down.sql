-- حذف ستون created_at در صورت بازگشت این نسخه (Down)
ALTER TABLE permission_templates
    DROP COLUMN IF EXISTS created_at;
