// Package config — خواندن پیکربندی از متغیرهای محیطی.
//
// اصل سند معماری: اطلاعات محرمانه (DSN دیتابیس، کلیدها) فقط در سمت سرور
// نگه‌داری می‌شوند و هرگز در کد فرانت‌اند یا گیت قرار نمی‌گیرند.
// فایل نمونه: deploy/env/fanavaryar.env.example
package config

import (
	"fmt"
	"strconv"
	"time"
)

type Config struct {
	// آدرس HTTP که برنامه روی آن گوش می‌دهد (روی localhost؛ Nginx جلوی آن است)
	Addr string
	// رشته اتصال PostgreSQL
	DBDSN string
	// پوشه نگهداری فایل‌ها و PDFها (خارج از دیتابیس، طبق سند بخش ۶)
	FilesDir string
	// کوکی امن فقط روی HTTPS (در محیط آزمایشی false، در عملیاتی true)
	CookieSecure bool
	// طول عمر نشست کاربر
	SessionTTL time.Duration
	// محیط اجرا: "dev" | "test" | "prod"
	Env string
	// مسیر فایل‌های ساخت‌شده فرانت‌اند (اختیاری؛ خالی = ارائه استاتیک خاموش)
	WebDir string
}

func Load(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		Addr:     getenv("FANAVARYAR_ADDR"),
		DBDSN:    getenv("FANAVARYAR_DB_DSN"),
		FilesDir: getenv("FANAVARYAR_FILES_DIR"),
		Env:      getenv("FANAVARYAR_ENV"),
		WebDir:   getenv("FANAVARYAR_WEB_DIR"),
	}

	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8080"
	}
	if cfg.FilesDir == "" {
		cfg.FilesDir = "/var/lib/fanavaryar/files"
	}
	if cfg.Env == "" {
		cfg.Env = "dev"
	}

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("config: FANAVARYAR_DB_DSN is required")
	}

	switch getenv("FANAVARYAR_COOKIE_SECURE") {
	case "true", "1", "yes":
		cfg.CookieSecure = true
	case "", "false", "0", "no":
		cfg.CookieSecure = false
	default:
		return nil, fmt.Errorf("config: invalid FANAVARYAR_COOKIE_SECURE value")
	}

	ttlHours := getenv("FANAVARYAR_SESSION_TTL_HOURS")
	if ttlHours == "" {
		ttlHours = "24"
	}
	hours, err := strconv.Atoi(ttlHours)
	if err != nil || hours <= 0 || hours > 24*30 {
		return nil, fmt.Errorf("config: invalid FANAVARYAR_SESSION_TTL_HOURS")
	}
	cfg.SessionTTL = time.Duration(hours) * time.Hour

	return cfg, nil
}

// IsProd تشخیص محیط عملیاتی برای تصمیم‌های امنیتی (کوکی، هدرها و ...)
func (c *Config) IsProd() bool { return c.Env == "prod" }
