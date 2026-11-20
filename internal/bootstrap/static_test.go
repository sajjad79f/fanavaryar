// تست رفتار ارائه فایل‌های استاتیک:
//  1. فایل موجود با کش بلندمدت سرو می‌شود (هش نام دارد)
//  2. فایل گمشده با پسوند، ۴۰۴ صریح می‌گیرد — نه index.html با MIME اشتباه
//  3. مسیرهای SPA (حتی زیر /assets/ مثل شناسه تجهیز) به index.html می‌رسند
//  4. index.html بدون کش سرو می‌شود تا بعد از استقرار، نسخه تازه دیده شود
package bootstrap

import (
        "net/http"
        "net/http/httptest"
        "os"
        "path/filepath"
        "strings"
        "testing"

        "github.com/go-chi/chi/v5"
)

func staticTestHandler(t *testing.T) http.Handler {
        t.Helper()
        webDir := t.TempDir()
        if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0o755); err != nil {
                t.Fatal(err)
        }
        index := "<!doctype html><html><body>app</body></html>"
        if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte(index), 0o644); err != nil {
                t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(webDir, "assets", "index-abc123.js"), []byte("console.log(1)"), 0o644); err != nil {
                t.Fatal(err)
        }
        // فایل‌های تصویر/فونت/استایل برای تست Content-Type صریح (مستقل از MIME سیستم)
        if err := os.WriteFile(filepath.Join(webDir, "assets", "logo-abc123.png"), []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
                t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(webDir, "assets", "font-abc123.woff2"), []byte("wOF2"), 0o644); err != nil {
                t.Fatal(err)
        }
        if err := os.WriteFile(filepath.Join(webDir, "assets", "style-abc123.css"), []byte("body{}"), 0o644); err != nil {
                t.Fatal(err)
        }

        mux := chi.NewMux()
        mux.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
                w.WriteHeader(http.StatusOK)
        })
        mountStatic(mux, webDir)
        return mux
}

func do(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
        t.Helper()
        req := httptest.NewRequest(http.MethodGet, path, nil)
        rec := httptest.NewRecorder()
        h.ServeHTTP(rec, req)
        return rec
}

func TestStaticExistingAssetCached(t *testing.T) {
        h := staticTestHandler(t)
        rec := do(t, h, "/assets/index-abc123.js")
        if rec.Code != http.StatusOK {
                t.Fatalf("status = %d, want 200", rec.Code)
        }
        if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
                t.Fatalf("Cache-Control = %q, want immutable", cc)
        }
        if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
                t.Fatalf("Content-Type = %q, want javascript", ct)
        }
        if !strings.Contains(rec.Body.String(), "console.log(1)") {
                t.Fatal("asset body not served")
        }
}

func TestStaticMissingAssetIs404NotIndex(t *testing.T) {
        h := staticTestHandler(t)
        rec := do(t, h, "/assets/index-missing.js")
        if rec.Code != http.StatusNotFound {
                t.Fatalf("status = %d, want 404 (got index fallback?)", rec.Code)
        }
        if ct := rec.Header().Get("Content-Type"); strings.Contains(ct, "text/html") {
                t.Fatalf("Content-Type = %q, must not be text/html for missing build files", ct)
        }
}

func TestStaticSPARouteUnderAssets(t *testing.T) {
        h := staticTestHandler(t)
        // /assets/5 در React Router یعنی جزئیات تجهیز شماره ۵ — باید index برگردد
        rec := do(t, h, "/assets/5")
        if rec.Code != http.StatusOK {
                t.Fatalf("status = %d, want 200", rec.Code)
        }
        if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
                t.Fatalf("Content-Type = %q, want text/html", ct)
        }
        if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
                t.Fatalf("Cache-Control = %q, want no-cache", cc)
        }
        if !strings.Contains(rec.Body.String(), "app") {
                t.Fatal("index.html not served")
        }
}

func TestStaticIndexNoCache(t *testing.T) {
        h := staticTestHandler(t)
        rec := do(t, h, "/login")
        if rec.Code != http.StatusOK {
                t.Fatalf("status = %d, want 200", rec.Code)
        }
        if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
                t.Fatalf("Cache-Control = %q, want no-cache", cc)
        }
}

// روی سرورهای حداقلی بدون /etc/mime.types، کتابخانه استاندارد نوع فایل را
// نمی‌شناسد و با nosniff مرورگر JS را بلاک می‌کند؛ پس Content-Type باید
// از جدول داخلی و به‌صورت صریح ست شود — برای همه پسوندهای بیلد.
func TestStaticExplicitContentTypes(t *testing.T) {
        h := staticTestHandler(t)
        cases := []struct{ path, want string }{
                {"/assets/index-abc123.js", "text/javascript"},
                {"/assets/style-abc123.css", "text/css"},
                {"/assets/logo-abc123.png", "image/png"},
                {"/assets/font-abc123.woff2", "font/woff2"},
        }
        for _, tc := range cases {
                rec := do(t, h, tc.path)
                if rec.Code != http.StatusOK {
                        t.Fatalf("%s: status = %d, want 200", tc.path, rec.Code)
                }
                if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, tc.want) {
                        t.Fatalf("%s: Content-Type = %q, want %q", tc.path, ct, tc.want)
                }
        }
}

func TestStaticAPIPathGuard(t *testing.T) {
        h := staticTestHandler(t)
        rec := do(t, h, "/api/unknown-endpoint")
        if rec.Code != http.StatusNotFound {
                t.Fatalf("status = %d, want 404", rec.Code)
        }
        if strings.Contains(rec.Body.String(), "app") {
                t.Fatal("API paths must never get index.html")
        }
}
