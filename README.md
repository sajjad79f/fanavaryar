# فناوریار — سامانه یکپارچه مدیریت و مستندسازی فناوری اطلاعات

اسکلت پیاده‌سازی فاز ۱ بر اساس سند معماری فنی:
**Modular Monolith** با **Go + Chi**، **PostgreSQL**، **React + TypeScript** روی **Debian 13 (Trixie)**.

## ساختار پروژه

```
fanavaryar/
├── cmd/fanavaryar/            نقطه ورود
├── internal/
│   ├── bootstrap/             سیم‌کشی ماژول‌ها + سرور HTTP + کاربر مدیر اولیه
│   ├── platform/              زیرساخت مشترک (config, database, auth, audit, httpx, events, appkit)
│   └── modules/
│       ├── users/             کاربران و مجوزهای ریزدانه
│       ├── locations/         مکان‌ها (ماژول الگو — کامل با تست)
│       ├── assets/            تجهیزات، وضعیت، جابه‌جایی، محدوده اطلاعاتی
│       └── reports/           گزارش فعالیت‌ها با چرخه تأیید
├── migrations/                Migrationهای SQL (embed شده)
├── deploy/
│   ├── systemd/               سرویس fanavaryar.service
│   ├── nginx/                 پیکربندی Reverse Proxy + HTTPS
│   ├── env/                   نمونه متغیرهای محیطی
│   └── backup/                اسکریپت و تایمر بکاپ
├── scripts/                   اسکریپت نصب سرور Debian
└── web/                       فرانت‌اند React + TypeScript (Vite)
```

هر ماژول چهار لایه دارد: `domain` (مدل و قواعد)، `application` (سناریوها)،
`infrastructure` (PostgreSQL)، `api` (REST). ماژول‌ها به جداول هم دسترسی
مستقیم ندارند؛ ارتباط با Interface داخلی یا رویداد دامنه‌ای انجام می‌شود
(نمونه: `locations.AssetLocationChecker` و `assets.LocationReader` که در
`internal/bootstrap/wire.go` سیم‌کشی شده‌اند).

## توسعه محلی

پیش‌نیاز: Go 1.23+، Node.js 20+، PostgreSQL 15+.

```bash
# ۱) دیتابیس
sudo -u postgres psql -c "CREATE USER fanavaryar_dev PASSWORD 'devpass';"
sudo -u postgres psql -c "CREATE DATABASE fanavaryar_dev OWNER fanavaryar_dev;"

# ۲) متغیرهای محیطی
export FANAVARYAR_DB_DSN="postgres://fanavaryar_dev:devpass@127.0.0.1:5432/fanavaryar_dev?sslmode=disable"
export FANAVARYAR_ADMIN_PASSWORD="ChangeMe-1234"   # فقط اولین اجرا

# ۳) اجرا (Migrationها خودکار اعمال می‌شوند)
go run ./cmd/fanavaryar

# ۴) تست‌ها
go test ./...
```

ورود: `POST /api/auth/login` با `admin` و رمزی که در متغیر بالا ست کردید.

## فرانت‌اند

```bash
cd web
npm install
npm run dev        # http://localhost:5173 — API به :8080 پروکسی می‌شود
npm run build      # خروجی در web/dist برای ارائه توسط خود Go یا Nginx
```

## استقرار روی Debian 13

راهنمای کامل در سند «راهنمای پیاده‌سازی فناوریار» است؛ خلاصه:

```bash
sudo bash scripts/install-server.sh       # آماده‌سازی سرور خام
make build && make web                    # ساخت backend و frontend
sudo make deploy                          # یا دستورهای بخش استقرار راهنما
```

فایل‌های `deploy/systemd/fanavaryar.service`، `deploy/nginx/fanavaryar.conf`
و `deploy/env/fanavaryar.env.example` را کپی و متناسب با محیط ویرایش کنید.

## بکاپ

```bash
sudo deploy/backup/fanavaryar-backup.sh   # دستی
# دوره‌ای: systemctl enable --now fanavaryar-backup.timer
```

## نکات امنیتی کلیدی

- همه مجوزها در Backend با `auth.RequirePermission` بررسی می‌شوند.
- رمزها با Argon2id هش می‌شوند؛ توکن نشست فقط به شکل SHA-256 در دیتابیس است.
- تغییر مجوزها و عملیات حساس در `audit_log` با وضعیت قبل/بعد ثبت می‌شود.
- در عملیاتی: `FANAVARYAR_COOKIE_SECURE=true` + HTTPS الزامی.
