// ایجاد اولین مدیر سامانه در نخستین اجرا (قرارداد v1).
// رمز از متغیر FANAVARYAR_ADMIN_PASSWORD خوانده می‌شود و فقط یک‌بار لازم است؛
// بعد از اولین ورود، تغییر رمز از خود سامانه انجام می‌شود.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fanavaryar/internal/modules/users/domain"
	"fanavaryar/internal/platform/auth"
	"fanavaryar/internal/platform/database"
)

func ensureFirstAdmin(ctx context.Context, pool *pgxpool.Pool, getenv func(string) string) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("check users: %w", err)
	}
	if count > 0 {
		return nil
	}

	password := getenv("FANAVARYAR_ADMIN_PASSWORD")
	if password == "" {
		return errors.New("first run: set FANAVARYAR_ADMIN_PASSWORD to create the initial admin account")
	}
	if err := domain.ValidatePassword(password); err != nil {
		return fmt.Errorf("first run: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	// کاربر مدیر با نقش admin + ابرمدیر و همه مجوزهای کاتالوگ ایجاد می‌شود
	err = database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		var adminID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (username, full_name, position, job_title, password_hash,
			                   is_active, role, is_superadmin, scope_type)
			VALUES ('admin', 'مدیر سامانه', 'مدیر فناوری اطلاعات', 'مدیر فناوری اطلاعات', $1,
			        true, 'admin', true, 'all')
			RETURNING id`, hash).Scan(&adminID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_permissions (user_id, permission_id)
			SELECT $1, id FROM permissions`, adminID)
		return err
	})
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	slog.Info("initial admin account created (username: admin) — change the password after first login")
	return nil
}
