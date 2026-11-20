// Package database — اتصال PostgreSQL و اجرای Migrationهای کنترل‌شده.
//
// طبق سند بخش ۶: تغییرات ساختار دیتابیس فقط از مسیر Migration انجام می‌شود؛
// هر Migration یک‌بار اجرا می‌شود و در جدول schema_migrations ثبت می‌گردد.
package database

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect یک Connection Pool ایجاد می‌کند.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: parse dsn: %w", err)
	}
	// برای شروع با یک Monolith داخلی، همین سقف‌ها کافی و منطقی‌اند؛
	// بعداً بر اساس بار واقعی تنظیم می‌شوند.
	cfg.MaxConns = 20
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	return pool, nil
}

// Migrate همه فایل‌های *.up.sql داخل fsys را به ترتیب نام اعمال می‌کند.
// هر نسخه فقط یک‌بار اجرا می‌شود (Idempotent).
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	if _, err := pool.Exec(ctx, `
                CREATE TABLE IF NOT EXISTS schema_migrations (
                        version    TEXT PRIMARY KEY,
                        applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
                )`); err != nil {
		return fmt.Errorf("database: ensure schema_migrations: %w", err)
	}

	entries, err := fs.Glob(fsys, "*.up.sql")
	if err != nil {
		return fmt.Errorf("database: list migrations: %w", err)
	}
	sort.Strings(entries)

	for _, name := range entries {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
		).Scan(&exists); err != nil {
			return fmt.Errorf("database: check %s: %w", name, err)
		}
		if exists {
			continue
		}

		body, err := readFile(fsys, name)
		if err != nil {
			return fmt.Errorf("database: read %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("database: begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, body); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("database: apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("database: record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("database: commit %s: %w", name, err)
		}
	}
	return nil
}

func readFile(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name) //nolint:gocritic
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WithTx یک تراکنش باز می‌کند، fn را اجرا می‌کند و خطاها را Rollback می‌کند.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// TrimDSN برای لاگ امن: رمز عبور در لاگ‌ها هرگز نمایش داده نمی‌شود.
func TrimDSN(dsn string) string {
	if i := strings.Index(dsn, "://"); i >= 0 {
		rest := dsn[i+3:]
		if j := strings.Index(rest, "@"); j >= 0 {
			user := rest[:j]
			if k := strings.Index(user, ":"); k >= 0 {
				return dsn[:i+3] + user[:k] + ":***" + rest[j:]
			}
		}
	}
	return dsn
}
