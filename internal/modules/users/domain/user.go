// Package domain — مدل کاربر، مجوز و قواعد اعتبارسنجی (قرارداد v1).
//
// سه مفهوم از هم جدا هستند:
//   - مشخصات حساب: نام، نام کاربری، سمت سازمانی (job_title)، نقش
//   - سمت سازمانی و نقش: صرفاً توصیفی/مدیریتی؛ مجوز ریزدانه جدا نگه‌داری می‌شود
//   - مجوزها: کدهای نقطه‌ای مثل "assets.create" که Backend بررسی می‌کند
package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrNotFound      = errors.New("users: not found")
	ErrUsernameTaken = errors.New("users: username already taken")
)

var usernameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,31}$`)

// نقش‌های حساب — نقش admin همه مجوزها را دور می‌زند؛ supervisor و user مجوزمحورند.
var validRoles = map[string]bool{"admin": true, "supervisor": true, "user": true}

func ValidRole(r string) bool { return validRoles[r] }

// ValidateUsername نام کاربری: حروف کوچک لاتین، رقم، نقطه، خط تیره/زیرین.
// فارسی در نام کاربری مجاز نیست تا لاگ‌ها و URLها یکدست بمانند.
func ValidateUsername(s string) error {
	s = strings.ToLower(strings.TrimSpace(s))
	if !usernameRe.MatchString(s) {
		return errors.New("نام کاربری باید ۳ تا ۳۲ کاراکتر از حروف کوچک لاتین، رقم، نقطه یا خط تیره باشد")
	}
	return nil
}

func ValidateFullName(s string) error {
	s = strings.TrimSpace(s)
	if len([]rune(s)) < 3 || len([]rune(s)) > 120 {
		return errors.New("نام و نام خانوادگی باید ۳ تا ۱۲۰ کاراکتر باشد")
	}
	return nil
}

// ValidatePassword حداقل دستورالعمل امنیتی نسخه اول؛
// در فازهای بعد سیاست پیچیدگی قابل تنظیم می‌شود.
func ValidatePassword(s string) error {
	if len(s) < 8 {
		return errors.New("رمز عبور باید حداقل ۸ کاراکتر باشد")
	}
	if len(s) > 128 {
		return errors.New("رمز عبور حداکثر ۱۲۸ کاراکتر است")
	}
	if strings.TrimSpace(s) == "" {
		return errors.New("رمز عبور نمی‌تواند فقط فاصله باشد")
	}
	return nil
}

var validScopes = map[string]bool{
	"all": true, "building": true, "floor": true, "room": true,
}

// ValidateScope محدوده اطلاعاتی جدا از نوع عملیات کنترل می‌شود:
// داشتن مجوز مشاهده یا ویرایش به معنی دسترسی به همه تجهیزات نیست.
// قرارداد v1 فقط همین چهار مقدار را می‌شناسد (assigned در فازهای بعد).
func ValidateScope(scopeType string, refID *int64) error {
	if !validScopes[scopeType] {
		return errors.New("نوع محدوده اطلاعاتی نامعتبر است")
	}
	switch scopeType {
	case "all":
		return nil
	case "building", "floor", "room":
		if refID == nil || *refID <= 0 {
			return errors.New("برای این نوع محدوده، انتخاب مکان الزامی است")
		}
		return nil
	}
	return errors.New("نوع محدوده اطلاعاتی نامعتبر است")
}
