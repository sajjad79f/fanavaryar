#!/usr/bin/env bash
# آماده‌سازی سرور خام Debian 13 (Trixie) برای فناوریار
# اجرا با root:  sudo bash install-server.sh
# این اسکریپت فقط زیرساخت پایه را می‌سازد؛ ساخت برنامه با make و کپی فایل‌ها
# طبق راهنمای پیاده‌سازی انجام می‌شود.
set -euo pipefail

APP_USER="fanavaryar"
APP_GROUP="fanavaryar"
DB_PASSWORD="${FANAVARYAR_DB_PASSWORD:-}"

[ "$(id -u)" -eq 0 ] || { echo "run as root (sudo)"; exit 1; }

log() { echo "[install] $*"; }

# ---------- ۱) به‌روزرسانی و بسته‌های پایه ----------
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get -y upgrade
apt-get -y install --no-install-recommends \
    postgresql nginx git curl ca-certificates \
    unattended-upgrades util-linux
log "base packages installed"

# به‌روزرسانی امنیتی خودکار برای بسته‌های امنیتی
dpkg-reconfigure --frontend noninteractive unattended-upgrades || true
systemctl enable --now unattended-upgrades || true

# ---------- ۲) حساب سرویس اختصاصی (بدون شل) ----------
if ! id "$APP_USER" >/dev/null 2>&1; then
    useradd --system --home-dir /opt/fanavaryar --shell /usr/sbin/nologin "$APP_USER"
    log "service user created: $APP_USER"
fi

# ---------- ۳) PostgreSQL: نقش و پایگاه داده ----------
if [ -z "$DB_PASSWORD" ]; then
    DB_PASSWORD="$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24)"
    log "generated random DB password (set it in /etc/fanavaryar/fanavaryar.env)"
fi
sudo_db() { sudo -u postgres psql -v ON_ERROR_STOP=1 -c "$1"; }

sudo_db "DO \$\$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='${APP_USER}') THEN
    CREATE ROLE ${APP_USER} LOGIN PASSWORD '${DB_PASSWORD}';
  END IF;
\$\$"
sudo_db "SELECT 'CREATE DATABASE ${APP_USER} OWNER ${APP_USER}'
  WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname='${APP_USER}')\gexec"
log "postgres role/database ready"

# فقط اتصال محلی با رمز؛ اتصال شبکه بیرونی باز نیست
PG_HBA="/etc/postgresql/*/main/pg_hba.conf"
if grep -q "^host.*${APP_USER}.*scram-sha-256" $PG_HBA 2>/dev/null; then
    log "pg_hba already has rule for ${APP_USER}"
else
    echo "host    ${APP_USER}    ${APP_USER}    127.0.0.1/32    scram-sha-256" >> $PG_HBA
    systemctl reload postgresql
    log "pg_hba rule added"
fi

# ---------- ۴) دایرکتوری‌ها ----------
install -d -o root -g "$APP_GROUP" -m 750 /opt/fanavaryar
install -d -o root -g "$APP_GROUP" -m 750 /opt/fanavaryar/bin
install -d -o root -g "$APP_GROUP" -m 750 /etc/fanavaryar
install -d -o "$APP_USER" -g "$APP_GROUP" -m 750 /var/lib/fanavaryar/files
install -d -o "$APP_USER" -g "$APP_GROUP" -m 750 /var/backups/fanavaryar
install -d -m 750 /etc/nginx/ssl
log "directories created"

# ---------- ۵) فایروال nftables: فقط ۲۲/۸۰/۴۴۳ ----------
cat > /etc/nftables.conf <<'NFT'
#!/usr/sbin/nft -f
flush ruleset

table inet filter {
    chain input {
        type filter hook input priority 0; policy drop;
        iif lo accept
        ct state established,related accept
        tcp dport 22 ct state new limit rate 15/minute accept comment "SSH"
        tcp dport 80 accept comment "HTTP (redirect + ACME)"
        tcp dport 443 accept comment "HTTPS"
        ip protocol icmp accept
        ip6 nexthdr icmpv6 accept
        counter reject with icmpx type admin-prohibited
    }
    chain forward { type filter hook forward priority 0; policy drop; }
    chain output  { type filter hook output priority 0; policy accept; }
}
NFT
systemctl enable --now nftables
systemctl restart nftables
log "nftables firewall active (22/80/443)"

# ---------- ۶) دسترسی PostgreSQL برای بکاپ ----------
# کاربر fanavaryar با pg_dump local (peer) پشتیبان می‌گیرد
if ! grep -q "^local.*${APP_USER}" $PG_HBA 2>/dev/null; then
    echo "local   ${APP_USER}    ${APP_USER}                     peer" >> $PG_HBA
    systemctl reload postgresql
fi
log "backup access configured"

# ---------- ۷) پیام‌های پایانی ----------
cat <<EOF

================ نکته‌های دستی بعد از اسکریپت ================
1) فایل پیکربندی: cp deploy/env/fanavaryar.env.example /etc/fanavaryar/fanavaryar.env
   رمز دیتابیس این اجرا: $DB_PASSWORD  (در همان فایل ست شود)
2) کپی سرویس: cp deploy/systemd/fanavaryar.service /etc/systemd/system/
   systemctl daemon-reload && systemctl enable --now fanavaryar
3) کپی nginx: cp deploy/nginx/fanavaryar.conf /etc/nginx/sites-available/fanavaryar.conf
   ویرایش server_name + مسیر گواهی؛ ln -s در sites-enabled؛ nginx -t؛ systemctl reload nginx
4) گواهی عملیاتی (دامنه واقعی): certbot --nginx -d fanavaryar.example.ir
5) هاردنینگ SSH: در /etc/ssh/sshd_config ست کنید
   PermitRootLogin no  و  PasswordAuthentication no
6) بکاپ دوره‌ای:
   cp deploy/backup/fanavaryar-backup.sh /usr/local/bin/ && chmod 750 /usr/local/bin/fanavaryar-backup.sh
   cp deploy/backup/fanavaryar-backup.{service,timer} /etc/systemd/system/
   systemctl daemon-reload && systemctl enable --now fanavaryar-backup.timer
=============================================================
EOF
log "server preparation finished"
