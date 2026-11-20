-- 0005: فیلدهای تکمیلی پرونده تجهیز — سازنده، مدل و مسئول/تحویل‌گیرنده
-- هم‌راستا با رابط کاربری نسخه کامل فرانت‌اند (فیلدهای توصیفی، اختیاری)
ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS manufacturer TEXT,
    ADD COLUMN IF NOT EXISTS model TEXT,
    ADD COLUMN IF NOT EXISTS responsible_person TEXT;
