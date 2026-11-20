// Middlewareهای عمومی HTTP: لاگ، ریکاوری، هدرهای امنیتی و HTTPS-Redirect.
package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// LogMiddleware هر درخواست را یک‌خطی و ساخت‌یافته لاگ می‌کند.
func LogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"ip", ClientIP(r),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// SecurityHeaders هدرهای امنیتی پایه را روی همه پاسخ‌ها ست می‌کند.
// طبق سند بخش ۷: ارتباط کاربران فقط از مسیر HTTPS است.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// RecoverMiddleware پنیک را به خطای ۵۰۰ تبدیل می‌کند تا کل سرویس نیندازد.
func RecoverMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					logger.Error("panic recovered", "path", r.URL.Path, "panic", p)
					Error(w, Internal("internal error", nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
