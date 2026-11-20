// Package application — سناریوهای ماژول مکان‌ها (قرارداد v1).
package application

import (
        "context"
        "log/slog"
        "strings"
        "time"

        "fanavaryar/internal/modules/locations/domain"
        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

// LocationRow یک ردیف تختِ مکان با شمارش تجهیزات — قرارداد فرانت‌اند.
type LocationRow struct {
        ID          int64     `json:"id"`
        ParentID    *int64    `json:"parent_id"`
        Kind        string    `json:"kind"`
        Name        string    `json:"name"`
        Code        *string   `json:"code"`
        AssetsCount int       `json:"assets_count"`
        CreatedAt   time.Time `json:"created_at"`
}

type Repository interface {
        List(ctx context.Context) ([]LocationRow, error)
        GetKind(ctx context.Context, id int64) (string, error)
        Create(ctx context.Context, in CreateInput) (LocationRow, error)
        Rename(ctx context.Context, id int64, name string, code *string) (LocationRow, error)
        Delete(ctx context.Context, id int64) error
        GetSnapshot(ctx context.Context, id int64) (string, string, error)
        CountChildren(ctx context.Context, id int64) (int, error)
        CountAssets(ctx context.Context, id int64) (int, error)
}

type AuditLogger interface {
        Log(ctx context.Context, e audit.Entry) error
}

type Service struct {
        repo  Repository
        audit AuditLogger
}

func NewService(repo Repository, auditLog AuditLogger) *Service {
        return &Service{repo: repo, audit: auditLog}
}

// List فهرست تخت مکان‌ها؛ ساخت درخت سمت کلاینت انجام می‌شود.
func (s *Service) List(ctx context.Context) ([]LocationRow, error) {
        rows, err := s.repo.List(ctx)
        if err != nil {
                return nil, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if rows == nil {
                rows = []LocationRow{}
        }
        return rows, nil
}

// ScopeLocationKind نوع مکان برای اعتبارسنجی محدوده دسترسی کاربران.
func (s *Service) ScopeLocationKind(ctx context.Context, id int64) (string, error) {
        kind, err := s.repo.GetKind(ctx, id)
        if err != nil {
                return "", err
        }
        return kind, nil
}

type CreateInput struct {
        Name     string
        Kind     string
        ParentID *int64
        Code     *string
}

func (s *Service) Create(ctx context.Context, actor *auth.AuthUser, in CreateInput) (LocationRow, error) {
        name := strings.TrimSpace(in.Name)
        if name == "" {
                return LocationRow{}, httpx.BadRequest("نام مکان الزامی است")
        }
        if len([]rune(name)) > 120 {
                return LocationRow{}, httpx.BadRequest("نام مکان حداکثر ۱۲۰ کاراکتر است")
        }
        kind := domain.Kind(in.Kind)
        switch kind {
        case domain.KindBuilding, domain.KindFloor, domain.KindRoom, domain.KindRack:
        default:
                return LocationRow{}, httpx.BadRequest("نوع مکان نامعتبر است")
        }
        if code := trimPtr(in.Code); code != nil && len(*code) > 30 {
                return LocationRow{}, httpx.BadRequest("کد مکان حداکثر ۳۰ کاراکتر است")
        }

        // قاعده سلسله‌مراتب: ساختمان ریشه است؛ هر فرزند فقط زیر والدِ مجاز خودش.
        if in.ParentID == nil {
                if kind != domain.KindBuilding {
                        return LocationRow{}, httpx.BadRequest("مکان ریشه فقط می‌تواند ساختمان باشد")
                }
        } else {
                parentKind, err := s.repo.GetKind(ctx, *in.ParentID)
                if err != nil {
                        return LocationRow{}, httpx.BadRequest("مکان والد یافت نشد")
                }
                if !domain.CanNest(domain.Kind(parentKind), kind) {
                        return LocationRow{}, httpx.BadRequest("این نوع مکان نمی‌تواند زیر این والد قرار بگیرد")
                }
        }

        row, err := s.repo.Create(ctx, CreateInput{Name: name, Kind: string(kind), ParentID: in.ParentID, Code: trimPtr(in.Code)})
        if err != nil {
                return LocationRow{}, httpx.Internal("ایجاد مکان ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "create", "location", row)
        return row, nil
}

func (s *Service) Rename(ctx context.Context, actor *auth.AuthUser, id int64, name string, code *string) (LocationRow, error) {
        name = strings.TrimSpace(name)
        if name == "" {
                return LocationRow{}, httpx.BadRequest("نام مکان الزامی است")
        }
        if len([]rune(name)) > 120 {
                return LocationRow{}, httpx.BadRequest("نام مکان حداکثر ۱۲۰ کاراکتر است")
        }
        if code := trimPtr(code); code != nil && len(*code) > 30 {
                return LocationRow{}, httpx.BadRequest("کد مکان حداکثر ۳۰ کاراکتر است")
        }
        row, err := s.repo.Rename(ctx, id, name, trimPtr(code))
        if err != nil {
                return LocationRow{}, httpx.Internal("ذخیره تغییرات ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "update", "location", row)
        return row, nil
}

func (s *Service) Delete(ctx context.Context, actor *auth.AuthUser, id int64) error {
        children, err := s.repo.CountChildren(ctx, id)
        if err != nil {
                return httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if children > 0 {
                return httpx.Conflict("این مکان زیرمجموعه دارد؛ ابتدا زیرمجموعه‌ها را حذف کنید")
        }
        assets, err := s.repo.CountAssets(ctx, id)
        if err != nil {
                return httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if assets > 0 {
                return httpx.Conflict("به این مکان تجهیز متصل است؛ ابتدا مکان تجهیزات را تغییر دهید")
        }
        if err := s.repo.Delete(ctx, id); err != nil {
                return httpx.Internal("حذف مکان ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "delete", "location", map[string]any{"id": id})
        return nil
}

func (s *Service) auditLog(ctx context.Context, actor *auth.AuthUser, action, entityType string, after any) {
        if s.audit == nil {
                return
        }
        e := audit.Entry{
                Module: audit.ModLocation, Action: action, EntityType: entityType, After: after,
        }
        if actor != nil {
                e.ActorID = &actor.ID
                e.ActorUsername = actor.Username
        }
        if err := s.audit.Log(ctx, e); err != nil {
                slog.Error("audit write failed", "module", "location", "action", action, "err", err)
        }
}

func trimPtr(s *string) *string {
        if s == nil {
                return nil
        }
        t := strings.TrimSpace(*s)
        if t == "" {
                return nil
        }
        return &t
}

// Snapshot نوع و نام مکان برای ماژول تجهیزات (به‌صورت قرارداد درون‌سامانه‌ای).
func (s *Service) Snapshot(ctx context.Context, id int64) (kind, name string, err error) {
        return s.repo.GetSnapshot(ctx, id)
}
