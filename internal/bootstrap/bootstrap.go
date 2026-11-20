// Package bootstrap — راه‌اندازی سامانه و سیم‌کشی ماژول‌ها (API نسخه ۱).
//
// این پکیج تنها جایی است که ماژول‌ها همدیگر را می‌بینند؛ خود ماژول‌ها
// فقط با Interfaceهای داخلی کار می‌کنند تا در آینده بتوان هر ماژول را
// جداگانه مستقر کرد (اصل Modular Monolith سند معماری).
//
// قرارداد مسیرها: همه APIها زیر /api/v1 هستند و خطاها با پاکت
// {"error": "..."} برمی‌گردند — هم‌سان با فرانت‌اند فناوریار.
package bootstrap

import (
        "context"
        "errors"
        "fmt"
        "log/slog"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

        "github.com/go-chi/chi/v5"
        "github.com/go-chi/chi/v5/middleware"

        assetsApi "fanavaryar/internal/modules/assets/api"
        assetsApp "fanavaryar/internal/modules/assets/application"
        assetsInfra "fanavaryar/internal/modules/assets/infrastructure"
        formsApi "fanavaryar/internal/modules/forms/api"
        formsApp "fanavaryar/internal/modules/forms/application"
        formsInfra "fanavaryar/internal/modules/forms/infrastructure"
        locationsApi "fanavaryar/internal/modules/locations/api"
        locationsApp "fanavaryar/internal/modules/locations/application"
        locationsInfra "fanavaryar/internal/modules/locations/infrastructure"
        reportsApi "fanavaryar/internal/modules/reports/api"
        reportsApp "fanavaryar/internal/modules/reports/application"
        reportsInfra "fanavaryar/internal/modules/reports/infrastructure"
        usersApi "fanavaryar/internal/modules/users/api"
        usersApp "fanavaryar/internal/modules/users/application"
        usersInfra "fanavaryar/internal/modules/users/infrastructure"
        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/config"
        "fanavaryar/internal/platform/database"
        "fanavaryar/internal/platform/httpx"
        "fanavaryar/migrations"
)

// Run کل چرخه عمر سامانه را مدیریت می‌کند.
func Run(ctx context.Context, getenv func(string) string) error {
        cfg, err := config.Load(getenv)
        if err != nil {
                fmt.Fprintln(os.Stderr, "config error:", err)
                return err
        }

        logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
        slog.SetDefault(logger)

        // پوشه فایل‌ها و PDFها خارج از دیتابیس (سند بخش ۶)
        if err := os.MkdirAll(cfg.FilesDir, 0o750); err != nil {
                return fmt.Errorf("create files dir: %w", err)
        }

        pool, err := database.Connect(ctx, cfg.DBDSN)
        if err != nil {
                return err
        }
        defer pool.Close()
        logger.Info("database connected", "dsn", database.TrimDSN(cfg.DBDSN))

        if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
                return fmt.Errorf("migrate: %w", err)
        }
        logger.Info("migrations applied")

        if err := ensureFirstAdmin(ctx, pool, getenv); err != nil {
                return err
        }

        // ---------- سیم‌کشی ماژول‌ها ----------
        auditSvc := audit.NewService(pool)
        sessionSvc := auth.NewSessionService(pool)

        locRepo := locationsInfra.NewLocationRepository(pool)
        locSvc := locationsApp.NewService(locRepo, auditSvc)
        locHandler := locationsApi.NewHandler(locSvc)

        userRepo := usersInfra.NewUserRepository(pool)
        usersSvc := usersApp.NewService(userRepo, passwordHasher{}, sessionSvc, auditSvc,
                scopeLocationChecker{svc: locSvc})
        usersHandler := usersApi.NewHandler(usersSvc, sessionSvc, cfg.CookieSecure, cfg.SessionTTL)

        assetRepo := assetsInfra.NewAssetRepository(pool)
        assetSvc := assetsApp.NewService(assetRepo, locationReader{svc: locSvc}, auditSvc)
        assetHandler := assetsApi.NewHandler(assetSvc)

        formsRepo := formsInfra.NewFormRepository(pool)
        formsSvc := formsApp.NewService(formsRepo, auditSvc)
        formsHandler := formsApi.NewHandler(formsSvc)

        reportRepo := reportsInfra.NewReportRepository(pool)
        reportSvc := reportsApp.NewService(reportRepo, formsSvc, reportRepo, auditSvc)
        reportHandler := reportsApi.NewHandler(reportSvc)

        // ---------- روتر ----------
        router := chi.NewRouter()
        router.Use(middleware.RequestID)
        router.Use(middleware.RealIP)
        router.Use(httpx.RecoverMiddleware(logger))
        router.Use(httpx.LogMiddleware(logger))
        router.Use(httpx.SecurityHeaders)

        // مسیر سلامت سرویس — برای مانیتورینگ
        router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
                if err := pool.Ping(r.Context()); err != nil {
                        httpx.Error(w, httpx.Internal("database unreachable", err))
                        return
                }
                httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
        })

        // ---------- API نسخه ۱ ----------
        router.Route("/api/v1", func(r chi.Router) {
                // مسیر عمومی ورود (قبل از میدلور احراز هویت)
                usersHandler.MountAuth(r)

                r.Group(func(r chi.Router) {
                        r.Use(auth.AuthMiddleware(pool, logger))
                        usersHandler.MountSession(r)
                        usersHandler.Mount(r)
                        locHandler.Mount(r)
                        assetHandler.Mount(r)
                        formsHandler.Mount(r)
                        reportHandler.Mount(r)
                        mountAuditHistory(r, auditSvc) // دفتر رویدادها
                        r.Get("/stats", hStats(pool, auditSvc)) // آمار داشبورد برای همه کاربران احرازشده
                })
        })

        // ارائه فایل‌های ساخت‌شده فرانت‌اند (اختیاری)
        mountStatic(router, cfg.WebDir)

        // ---------- پاک‌سازی دوره‌ای نشست‌های منقضی ----------
        go func() {
                ticker := time.NewTicker(time.Hour)
                defer ticker.Stop()
                for {
                        select {
                        case <-ctx.Done():
                                return
                        case <-ticker.C:
                                if err := sessionSvc.CleanupExpired(ctx); err != nil {
                                        logger.Warn("session cleanup failed", "err", err)
                                }
                        }
                }
        }()

        // ---------- سرور و خاموشی مهربان ----------
        srv := &http.Server{
                Addr:              cfg.Addr,
                Handler:           router,
                ReadHeaderTimeout: 10 * time.Second,
                ReadTimeout:       30 * time.Second,
                WriteTimeout:      30 * time.Second,
                IdleTimeout:       120 * time.Second,
        }

        runCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
        defer stop()

        go func() {
                <-runCtx.Done()
                logger.Info("shutting down")
                shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
                defer cancel()
                _ = srv.Shutdown(shutdownCtx)
        }()

        logger.Info("fanavaryar listening", "addr", cfg.Addr, "env", cfg.Env)
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
                return fmt.Errorf("server: %w", err)
        }
        logger.Info("bye")
        return nil
}
