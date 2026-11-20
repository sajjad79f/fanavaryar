// Package appkit — helperهای کوچک مشترک ماژول‌ها.
// عمداً کوچک نگه‌داری می‌شود تا وابستگی ماژول‌ها به هم از این مسیر رد نشود.
package appkit

import (
	"strconv"

	"fanavaryar/internal/platform/auth"
)

func Int64ToStr(i int64) string { return strconv.FormatInt(i, 10) }

// ActorUsername نام کاربری عامل عملیات؛ برای رویدادهای سیستم "system" برمی‌گردد.
func ActorUsername(u *auth.AuthUser) string {
	if u == nil {
		return "system"
	}
	return u.Username
}
