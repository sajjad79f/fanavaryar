// Middleware احراز هویت و کنترل مجوز — قرارداد API نسخه ۱.
//
// اصل سند بخش ۵: همه مجوزها در Backend بررسی می‌شوند؛
// پنهان‌کردن دکمه در Frontend کنترل امنیتی نیست.
//
// قاعده مجوز v1:
//   - نقش admin و حساب‌های is_superadmin به همه چیز دسترسی دارند.
//   - بقیه کاربران فقط با کد مجوز صریح (مثل "assets.create") مجازند.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fanavaryar/internal/platform/httpx"
)

// AuthUser اطلاعات کاربر احرازشده + مجوزهای او برای این درخواست.
// کلیدهای JSON دقیقاً قرارداد UserInfo فرانت‌اند است (snake_case).
type AuthUser struct {
	ID           int64    `json:"id"`
	Username     string   `json:"username"`
	FullName     string   `json:"full_name"`
	JobTitle     string   `json:"job_title"`
	Role         string   `json:"role"` // admin | supervisor | user
	IsSuperadmin bool     `json:"is_superadmin"`
	ScopeType    string   `json:"-"`
	ScopeRefID   *int64   `json:"-"`
	Perms        []string `json:"permissions"` // کدهای نقطه‌ای مثل "assets.create"
}

// IsAdmin نقش مدیریتی کامل دارد (مدیر سامانه یا ابرمدیر).
func (u *AuthUser) IsAdmin() bool {
	return u != nil && (u.IsSuperadmin || u.Role == "admin")
}

// Has بررسی یک کد مجوز؛ نقش‌های مدیریتی همیشه مجازند.
func (u *AuthUser) Has(code string) bool {
	if u == nil {
		return false
	}
	if u.IsAdmin() {
		return true
	}
	for _, p := range u.Perms {
		if p == code {
			return true
		}
	}
	return false
}

// HasAny حداقل یکی از کدها را داشته باشد.
func (u *AuthUser) HasAny(codes ...string) bool {
	if u == nil {
		return false
	}
	if u.IsAdmin() {
		return true
	}
	for _, c := range codes {
		for _, p := range u.Perms {
			if p == c {
				return true
			}
		}
	}
	return false
}

// SortedPermissions خروجی پایدار برای پاسخ JSON.
func (u *AuthUser) SortedPermissions() []string {
	out := append([]string(nil), u.Perms...)
	sort.Strings(out)
	if out == nil {
		out = []string{}
	}
	return out
}

type ctxKey int

const authUserKey ctxKey = 1

// FromContext کاربر احرازشده را برمی‌گرداند؛ برای مسیرهای عمومی nil است.
func FromContext(ctx context.Context) *AuthUser {
	u, _ := ctx.Value(authUserKey).(*AuthUser)
	return u
}

// AuthMiddleware کوکی نشست را بررسی و کاربر + مجوزها را به Context اضافه می‌کند.
// مسیرهایی مثل /api/v1/auth/login قبل از این میدلور روی روتر سوار می‌شوند.
func AuthMiddleware(pool *pgxpool.Pool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				httpx.Error(w, httpx.Unauthorized("ابتدا وارد سامانه شوید"))
				return
			}

			user, err := loadUserBySession(r.Context(), pool, cookie.Value)
			if err != nil {
				if errors.Is(err, ErrSessionNotFound) {
					httpx.Error(w, httpx.Unauthorized("نشست معتبر نیست"))
					return
				}
				logger.Error("load session", "err", err)
				httpx.Error(w, httpx.Internal("internal error", err))
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission دسترسی به مسیر را به یک کد مجوز مشخص گره می‌زند.
// تلاش‌های ناموفق در لاگ سامانه ثبت می‌شوند (مبنای Audit).
func RequirePermission(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := FromContext(r.Context())
			if user == nil {
				httpx.Error(w, httpx.Unauthorized("ابتدا وارد سامانه شوید"))
				return
			}
			if !user.Has(code) {
				slog.Warn("permission denied",
					"user", user.Username, "code", code, "path", r.URL.Path)
				httpx.Error(w, httpx.Forbidden("شما مجوز لازم برای این عملیات را ندارید"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func loadUserBySession(ctx context.Context, pool *pgxpool.Pool, token string) (*AuthUser, error) {
	// نشست + کاربر در یک کوئری؛ کاربر غیرفعال در خود کوئری فیلتر می‌شود.
	row := pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.full_name,
		       COALESCE(u.job_title, u.position, ''),
		       u.role, u.is_superadmin, u.scope_type, u.scope_ref_id
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = encode(sha256(convert_to($1, 'UTF8')), 'hex')
		  AND s.expires_at > now()
		  AND u.is_active`, token)

	var u AuthUser
	var scopeRef *int64
	if err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.JobTitle,
		&u.Role, &u.IsSuperadmin, &u.ScopeType, &scopeRef); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	u.ScopeRefID = scopeRef

	perms, err := pool.Query(ctx, `
		SELECT DISTINCT p.module || '.' || p.action
		FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id
		WHERE up.user_id = $1`, u.ID)
	if err != nil {
		return nil, err
	}
	defer perms.Close()

	u.Perms = []string{}
	for perms.Next() {
		var perm string
		if err := perms.Scan(&perm); err != nil {
			return nil, err
		}
		u.Perms = append(u.Perms, perm)
	}
	return &u, perms.Err()
}

// ParsePermissionCode کد "module.action" را جدا می‌کند — برای ابزارها و تست‌ها.
func ParsePermissionCode(s string) (module, action string, ok bool) {
	i := strings.IndexByte(s, '.')
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}
