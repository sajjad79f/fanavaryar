#!/usr/bin/env bash
# بازیابی بکاپ فناوریار — فقط روی سرور هدف و با آگاهی کامل اجرا شود.
# استفاده:
#   sudo bash restore-backup.sh /var/backups/fanavaryar/<STAMP> [--files-only|--db-only]
# هشدار: بازیابی دیتابیس، داده‌های فعلی همان پایگاه را جایگزین می‌کند.
set -euo pipefail

if [ $# -lt 1 ]; then
    echo "usage: $0 <backup-dir> [--files-only|--db-only]" >&2
    exit 1
fi

SRC="$1"
MODE="${2:-all}"
FILES_DIR="${FILES_DIR:-/var/lib/fanavaryar/files}"
DB_NAME="${DB_NAME:-fanavaryar}"

# ۱) راستی‌آزمایی تمامیت فایل‌ها قبل از هر اقدامی
if [ -f "$SRC/SHA256SUMS" ]; then
    (cd "$SRC" && sha256sum --check SHA256SUMS)
    echo "integrity check: OK"
else
    echo "warning: SHA256SUMS missing" >&2
fi

# ۲) دیتابیس
if [ "$MODE" != "--files-only" ] && [ -f "$SRC/database.dump" ]; then
    echo "restoring database $DB_NAME ..."
    # اتصال‌های فعال باید بسته شوند؛ سرویس را موقتاً متوقف کنید:
    #   systemctl stop fanavaryar
    dropdb --if-exists "$DB_NAME"
    createdb --owner=fanavaryar "$DB_NAME"
    pg_restore --no-owner --no-privileges --dbname="$DB_NAME" "$SRC/database.dump"
    echo "database restored."
fi

# ۳) فایل‌ها
if [ "$MODE" != "--db-only" ] && [ -f "$SRC/files.tar.gz" ]; then
    echo "restoring files to $FILES_DIR ..."
    mkdir -p "$FILES_DIR"
    tar --extract --gzip --file="$SRC/files.tar.gz" --directory="$FILES_DIR"
    chown -R fanavaryar:fanavaryar "$FILES_DIR"
    echo "files restored."
fi

echo "done. remember to: systemctl start fanavaryar"
