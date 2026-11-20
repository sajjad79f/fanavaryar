// ارائه فایل‌های ساخت‌شده فرانت‌اند (React/TypeScript) در حالت عملیاتی.
// در توسعه، Vite خودش روی 5173 اجرا می‌شود و درخواست‌های API را proxy می‌کند.
//
// سه قاعده مهم این فایل:
//   1) فایل‌های بیلد (مثل assets/index-*.js) هش نام دارند و می‌توانند کش
//      بلندمدت بگیرند؛ اما index.html هرگز نباید کش شود تا بعد از استقرار
//      نسخه جدید، مرورگر نسخه قدیمی را نشان ندهد.
//   2) اگر فایلی با پسوند (مثل .js یا .css) روی دیسک پیدا نشود، به‌جای
//      برگرداندن index.html با Content-Type اشتباه، صریحاً ۴۰۴ می‌دهیم تا
//      خطای استقرار (کپی ناقص پوشه dist) پنهان نماند.
//   3) Content-Type همیشه از جدول داخلی ست می‌شود؛ روی سرورهای حداقلی
//      (بدون /etc/mime.types) کتابخانه استاندارد نمی‌تواند نوع فایل را
//      تشخیص دهد و js/css را text/plain می‌فرستد؛ با هدر
//      X-Content-Type-Options: nosniff مرورگر اجرای JS را کامل بلاک می‌کند
//      و کاربر صفحه سفید می‌بیند. جدول پایین این مشکل را ریشه‌ای حل می‌کند.
package bootstrap

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// chiRouter حداقل امضاهای لازم از chi.Mux برای نصب فایل‌های استاتیک.
type chiRouter interface {
	Get(pattern string, handler http.HandlerFunc)
	NotFound(handler http.HandlerFunc)
}

// staticContentTypes جدول صریح نوع محتوا برای پسوندهای بیلد فرانت‌اند.
// هدف: رفتار یکسان روی هر سروری، مستقل از وجود /etc/mime.types.
var staticContentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".mjs":         "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".json":        "application/json; charset=utf-8",
	".map":         "application/json; charset=utf-8",
	".webmanifest": "application/manifest+json",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".webp":        "image/webp",
	".avif":        "image/avif",
	".ico":         "image/x-icon",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
	".txt":         "text/plain; charset=utf-8",
}

func contentTypeFor(path string) string {
	return staticContentTypes[strings.ToLower(filepath.Ext(path))]
}

func mountStatic(r chiRouter, webDir string) {
	if webDir == "" {
		return
	}
	index := filepath.Join(webDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		slog.Warn("web dist not found, static serving disabled", "dir", webDir)
		return
	}

	fs := http.FileServer(http.Dir(webDir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// مسیرهای /api به API می‌روند؛ اینجا فقط بقیه مسیرها
		if strings.HasPrefix(req.URL.Path, "/api/") {
			http.NotFound(w, req)
			return
		}

		candidate := filepath.Join(webDir, filepath.Clean(req.URL.Path))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			// فایل‌های بیلد با هش نام‌گذاری شده‌اند؛ کش بلندمدت بی‌خطر است.
			// Content-Type صریح قبل از سرو کردن — شاخص‌ترین رفع باگ صفحه سفید.
			if ct := contentTypeFor(candidate); ct != "" {
				w.Header().Set("Content-Type", ct)
			}
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fs.ServeHTTP(w, req)
			return
		}

		// درخواست «فایل» (دارای پسوند) که وجود ندارد → ۴۰۴ صریح؛
		// برگرداندن index.html اینجا باعث خطای MIME در مرورگر می‌شود.
		// مسیرهای SPA مثل /assets/5 (شناسه تجهیز) پسوند ندارند و سالم می‌مانند.
		if ext := filepath.Ext(req.URL.Path); ext != "" {
			slog.Warn("static file missing (check dist copy)", "path", req.URL.Path, "dir", webDir)
			http.NotFound(w, req)
			return
		}

		// SPA: هر مسیر ناشناخته به index.html می‌رسد — بدون کش
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, req, index)
	})
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, req, index)
	})
}
