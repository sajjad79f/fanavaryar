// داشبورد — آمار کلی سامانه برای GET /api/v1/stats.
package bootstrap

import (
        "net/http"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"

        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/httpx"
)

type statsBody struct {
        Stats          statsNumbers  `json:"stats"`
        PendingReports []pendingRow  `json:"pending_reports"`
        RecentAudit    []audit.Recent `json:"recent_audit"`
}

type statsNumbers struct {
        AssetsTotal    int `json:"assets_total"`
        AssetsActive   int `json:"assets_active"`
        ReportsPending int `json:"reports_pending"`
        ReportsMonth   int `json:"reports_month"`
        FormsActive    int `json:"forms_active"`
        UsersActive    int `json:"users_active"`
}

type pendingRow struct {
        ID            int64   `json:"id"`
        Code          *string `json:"code"`
        Title         string  `json:"title"`
        ReportDate    string  `json:"report_date"`
        CreatedByName *string   `json:"created_by_name"`
        CreatedAt     time.Time `json:"created_at"`
}

func hStats(pool *pgxpool.Pool, auditSvc *audit.Service) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                var s statsNumbers
                err := pool.QueryRow(r.Context(), `
                        SELECT
                                (SELECT count(*) FROM assets),
                                (SELECT count(*) FROM assets WHERE status = 'active'),
                                (SELECT count(*) FROM reports WHERE status = 'pending'),
                                (SELECT count(*) FROM reports WHERE created_at > now() - interval '30 days'),
                                (SELECT count(*) FROM form_templates WHERE is_active),
                                (SELECT count(*) FROM users WHERE is_active)
                `).Scan(&s.AssetsTotal, &s.AssetsActive, &s.ReportsPending, &s.ReportsMonth,
                        &s.FormsActive, &s.UsersActive)
                if err != nil {
                        httpx.Error(w, httpx.Internal("خطای ذخیره‌سازی", err))
                        return
                }

                rows, err := pool.Query(r.Context(), `
                        SELECT rp.id, rp.code, rp.title, to_char(rp.report_date, 'YYYY-MM-DD'),
                               cu.full_name, rp.created_at
                        FROM reports rp
                        JOIN users cu ON cu.id = rp.author_id
                        WHERE rp.status = 'pending'
                        ORDER BY rp.created_at DESC, rp.id DESC
                        LIMIT 8`)
                if err != nil {
                        httpx.Error(w, httpx.Internal("خطای ذخیره‌سازی", err))
                        return
                }
                defer rows.Close()
                pending := []pendingRow{}
                for rows.Next() {
                        var p pendingRow
                        if err := rows.Scan(&p.ID, &p.Code, &p.Title, &p.ReportDate, &p.CreatedByName, &p.CreatedAt); err != nil {
                                httpx.Error(w, httpx.Internal("خطای ذخیره‌سازی", err))
                                return
                        }
                        pending = append(pending, p)
                }

                recent, err := auditSvc.Recent(r.Context(), 8)
                if err != nil {
                        httpx.Error(w, httpx.Internal("خطای ذخیره‌سازی", err))
                        return
                }

                httpx.JSON(w, http.StatusOK, statsBody{
                        Stats: s, PendingReports: pending, RecentAudit: recent,
                })
        }
}
