#!/usr/bin/env bash
# بکاپ‌گیری فناوریار: دیتابیس PostgreSQL + فایل‌ها
# طبق سند بخش ۷: دیتابیس و فایل‌ها مستقل و منظم پشتیبان‌گیری می‌شوند
# و فرایند بازیابی باید دوره‌ای آزمایش شود.
#
# نصب: cp fanavaryar-backup.sh /usr/local/bin/ && chmod 750 /usr/local/bin/fanavaryar-backup.sh
# تست دستی: sudo -u fanavaryar fanavaryar-backup.sh
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/fanavaryar}"
FILES_DIR="${FILES_DIR:-/var/lib/fanavaryar/files}"
DB_NAME="${DB_NAME:-fanavaryar}"
DB_USER="${DB_USER:-fanavaryar}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"

STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="$BACKUP_DIR/$STAMP"
LOG_TAG="fanavaryar-backup"

log() { logger -t "$LOG_TAG" -- "$1"; echo "$1"; }

umask 077
mkdir -p "$DEST"

# ۱) بکاپ دیتابیس (فرمت Custom برای بازیابی انتخابی و سریع‌تر)
pg_dump --format=custom --no-owner --no-privileges \
        --file="$DEST/database.dump" \
        "$DB_NAME"
log "database dump written: $DEST/database.dump"

# ۲) بکاپ فایل‌ها (PDF و پیوست‌ها) — tar فشرده
if [ -d "$FILES_DIR" ] && [ -n "$(ls -A "$FILES_DIR" 2>/dev/null)" ]; then
    tar --create --gzip --file="$DEST/files.tar.gz" --directory="$FILES_DIR" .
    log "files archive written: $DEST/files.tar.gz"
fi

# ۳) نگهداری چک‌سام دیتابیس برای راستی‌آزمایی هنگام بازیابی
sha256sum "$DEST"/* > "$DEST/SHA256SUMS"

# ۴) حذف نسخه‌های قدیمی‌تر از سقف نگهداری
find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d \
     -mtime +"$RETENTION_DAYS" -exec rm -rf {} +
log "backup completed: $DEST (retention: ${RETENTION_DAYS}d)"
