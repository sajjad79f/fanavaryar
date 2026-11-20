// Package audit — Audit Log سامانه.
//
// طبق سند بخش ۵ و ۷: ثبت رویدادهای امنیتی و عملیاتی مهم الزامی است؛
// به‌ویژه تغییر مجوزها باید شامل عامل تغییر، زمان و جزئیات قبل/بعد باشد.
// رکوردهای این جدول هرگز از طریق API عادی ویرایش یا حذف نمی‌شوند.
//
// قرارداد v1: کد رویداد «module.action» با نام ماژول مفرد است
// (auth.login، user.create، asset.move، ...) و در خروجی Query ساخته می‌شود.
package audit

import (
        "context"
        "encoding/json"
        "fmt"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"

        "fanavaryar/internal/platform/httpx"
)

// ماژول‌های مفرد مورداستفاده در Entry — هم‌سان با برچسب‌های فرانت‌اند.
const (
        ModAuth     = "auth"
        ModUser     = "user"
        ModPerm     = "perm"
        ModLocation = "location"
        ModAsset    = "asset"
        ModForm     = "form"
        ModReport   = "report"
)

type Entry struct {
        ActorID       *int64 // nil برای رویدادهای سیستم (مثل لاگین ناموفق بدون کاربر)
        ActorUsername string
        Module        string // مفرد: auth, user, perm, location, asset, form, report
        Action        string // create, update, move, login, ...
        EntityType    string // user, location, asset, report, form_template, ...
        EntityID      string
        Before        any // وضعیت قبل از تغییر (برای update/delete)
        After         any // وضعیت بعد از تغییر
        IP            string
}

type Service struct {
        pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
        return &Service{pool: pool}
}

// Log یک رویداد ممیزی ثبت می‌کند.
// خطای ثبت ممیزی در عملیات‌های حساس را نباید نادیده گرفت؛
// فراخوان‌دهنده تصمیم می‌گیرد عملیات را رد کند یا فقط لاگ کند.
func (s *Service) Log(ctx context.Context, e Entry) error {
        before, err := toJSON(e.Before)
        if err != nil {
                return fmt.Errorf("audit: before: %w", err)
        }
        after, err := toJSON(e.After)
        if err != nil {
                return fmt.Errorf("audit: after: %w", err)
        }

        _, err = s.pool.Exec(ctx, `
                INSERT INTO audit_log
                        (actor_id, actor_username, module, action, entity_type, entity_id, before_data, after_data, ip)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
                e.ActorID, e.ActorUsername, e.Module, e.Action,
                e.EntityType, e.EntityID, before, after, e.IP,
        )
        if err != nil {
                return fmt.Errorf("audit: insert: %w", err)
        }
        return nil
}

// Record ساختار خروجی قرارداد v1 — AuditItem فرانت‌اند.
type Record struct {
        ID         int64           `json:"id"`
        Action     string          `json:"action"` // مثل "user.create"
        EntityType *string         `json:"entity_type"`
        EntityID   *int64          `json:"entity_id"`
        Details    json.RawMessage `json:"details"`
        IP         *string         `json:"ip"`
        UserID     *int64          `json:"user_id"`
        Username   *string         `json:"username"`
        FullName   *string         `json:"full_name"`
        CreatedAt  time.Time       `json:"created_at"`
}

// Query فهرست صفحه‌بندی‌شده رویدادها با جست‌وجوی متنی ساده.
func (s *Service) Query(ctx context.Context, q string, p httpx.Page) (httpx.Result[Record], error) {
        where := `WHERE (module || '.' || action) ILIKE $1
                  OR actor_username ILIKE $1
                  OR COALESCE(u.full_name, '') ILIKE $1
                  OR entity_type ILIKE $1
                  OR entity_id ILIKE $1
                  OR COALESCE(ip, '') ILIKE $1`
        args := []any{"%" + q + "%"}

        var total int
        if err := s.pool.QueryRow(ctx, `
                SELECT count(*) FROM audit_log a
                LEFT JOIN users u ON u.id = a.actor_id
                `+where, args...).Scan(&total); err != nil {
                return httpx.Result[Record]{}, fmt.Errorf("audit: count: %w", err)
        }

        args = append(args, p.PerPage, p.Offset())
        rows, err := s.pool.Query(ctx, `
                SELECT a.id,
                       a.module || '.' || a.action,
                       a.entity_type, a.entity_id,
                       COALESCE(a.after_data, a.before_data),
                       a.ip, a.actor_id, a.actor_username, u.full_name, a.created_at
                FROM audit_log a
                LEFT JOIN users u ON u.id = a.actor_id
                `+where+`
                ORDER BY a.id DESC
                LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
        if err != nil {
                return httpx.Result[Record]{}, fmt.Errorf("audit: query: %w", err)
        }
        defer rows.Close()

        var out []Record
        for rows.Next() {
                var r Record
                var entityType, entityID *string
                if err := rows.Scan(&r.ID, &r.Action, &entityType, &entityID,
                        &r.Details, &r.IP, &r.UserID, &r.Username, &r.FullName, &r.CreatedAt); err != nil {
                        return httpx.Result[Record]{}, err
                }
                if entityType != nil && *entityType != "" {
                        r.EntityType = entityType
                }
                if entityID != nil {
                        var id int64
                        if _, err := fmt.Sscanf(*entityID, "%d", &id); err == nil {
                                r.EntityID = &id
                        }
                }
                out = append(out, r)
        }
        if err := rows.Err(); err != nil {
                return httpx.Result[Record]{}, err
        }
        return httpx.NewResult(out, total, p), nil
}

// Recent آخرین رویدادها برای داشبورد.
type Recent struct {
        ID         int64     `json:"id"`
        Action     string    `json:"action"`
        EntityType *string   `json:"entity_type"`
        Username   *string   `json:"username"`
        CreatedAt  time.Time `json:"created_at"`
}

func (s *Service) Recent(ctx context.Context, limit int) ([]Recent, error) {
        if limit <= 0 || limit > 50 {
                limit = 8
        }
        rows, err := s.pool.Query(ctx, `
                SELECT id, module || '.' || action, entity_type, actor_username, created_at
                FROM audit_log ORDER BY id DESC LIMIT $1`, limit)
        if err != nil {
                return nil, fmt.Errorf("audit: recent: %w", err)
        }
        defer rows.Close()

        var out []Recent
        for rows.Next() {
                var r Recent
                if err := rows.Scan(&r.ID, &r.Action, &r.EntityType, &r.Username, &r.CreatedAt); err != nil {
                        return nil, err
                }
                out = append(out, r)
        }
        if out == nil {
                out = []Recent{}
        }
        return out, rows.Err()
}

func toJSON(v any) ([]byte, error) {
        if v == nil {
                return nil, nil
        }
        b, err := json.Marshal(v)
        if err != nil {
                return nil, err
        }
        return b, nil
}
