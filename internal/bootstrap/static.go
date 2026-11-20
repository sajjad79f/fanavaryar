// ارائه فایل‌های ساخت‌شده فرانت‌اند (React/TypeScript) در حالت عملیاتی.
// در توسعه، Vite خودش روی 5173 اجرا می‌شود و درخواست‌های API را proxy می‌کند.
//
// دو قاعده مهم این فایل:
//   1) فایل‌های بیلد (مثل assets/index-*.js) هش نام دارند و می‌توانند کش
//      بلندمدت بگیرند؛ اما index.html هرگز نباید کش شود تا بعد از استقرار
//      نسخه جدید، مرورگر نسخه قدیمی را نشان ندهد.
//   2) اگر فایلی با پسوند (مثل .js یا .css) روی دیسک پیدا نشود، به‌جای
//      برگرداندن index.html با Content-Type اشتباه، صریحاً ۴۰۴ می‌دهیم تا
//      خطای استقرار (کپی ناقص پوشه dist) پنهان نماند.
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
			// فایل‌های بیلد با هش نام‌گذاری شده‌اند؛ کش بلندمدت بی‌خطر است
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
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, req, index)
	})
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, req, index)
	})
}
