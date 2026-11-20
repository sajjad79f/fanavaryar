-- 0007: رفع خطای ۵۰۰ در مودال دسترسی‌ها و الگوهای دسترسی
--
-- ریشهٔ مشکل: کوئری الگوها (user_repo.go / Templates) ستون t.created_at را
-- می‌خواند، ولی جدول permission_templates در ۰۰۰۱ بدون این ستون ساخته شده بود
-- و هیچ مایگریشن بعدی آن را اضافه نکرده بود؛ نتیجه: GET /permission-templates
-- همیشه «column t.created_at does not exist» می‌داد و مودال «دادن دسترسی»
-- پیام «یک خطای داخلی رخ داد؛ بعداً دوباره تلاش کنید» نشان می‌داد.
--
-- Idempotent است: روی دیتابیس‌های قدیمی ستون را اضافه می‌کند و برای رکوردهای
-- موجود زمانِ حال ثبت می‌شود؛ روی دیتابیس‌های تازه هم بدون خطا اجرا می‌شود.

ALTER TABLE permission_templates
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
