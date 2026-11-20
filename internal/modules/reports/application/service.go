// Package application — سناریوهای ماژول گزارش‌ها (قرارداد v1).
// هر گزارش به یک نسخه فرم متصل است و داده‌های آن با تعریف فیلدها اعتبارسنجی می‌شود.
package application

import (
        "context"
        "errors"
        "log/slog"
        "strings"
        "time"

        "fanavaryar/internal/modules/forms/application"
        "fanavaryar/internal/modules/forms/domain"
        reportsdomain "fanavaryar/internal/modules/reports/domain"
        "fanavaryar/internal/platform/appkit"
        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

// ReportListItem ردیف فهرست گزارش‌ها.
type ReportListItem struct {
        ID              int64     `json:"id"`
        Code            *string   `json:"code"`
        Title           string    `json:"title"`
        ReportDate      string    `json:"report_date"`
        Status          string    `json:"status"`
        CreatedByName   *string   `json:"created_by_name"`
        ApprovedByName  *string   `json:"approved_by_name"`
        CreatedAt       time.Time `json:"created_at"`
}

// ReportAsset خلاصه تجهیز متصل.
type ReportAsset struct {
        ID      int64  `json:"id"`
        AssetTag string `json:"asset_tag"`
        Name    string `json:"name"`
}

// ReportDetail پرونده کامل گزارش.
type ReportDetail struct {
        ID                    int64             `json:"id"`
        Code                  *string           `json:"code"`
        Title                 string            `json:"title"`
        ReportDate            string            `json:"report_date"`
        Status                string            `json:"status"`
        FormTemplateVersionID int64             `json:"form_template_version_id"`
        VersionNumber         int               `json:"version_number"`
        TemplateName          string            `json:"template_name"`
        Fields                []domain.FieldDef `json:"fields"`
        FormData              map[string]any    `json:"form_data"`
        CreatedByName         *string           `json:"created_by_name"`
        ApprovedByName        *string           `json:"approved_by_name"`
        ApprovedAt            *time.Time        `json:"approved_at"`
        RejectReason          *string           `json:"reject_reason"`
        CreatedAt             time.Time         `json:"created_at"`
        Assets                []ReportAsset     `json:"assets"`
}

type Repository interface {
        List(ctx context.Context, f ListFilter) ([]ReportListItem, int, error)
        GetDetail(ctx context.Context, id int64) (ReportDetail, error)
        Create(ctx context.Context, in CreateInput, actorID int64) (int64, error)
        Approve(ctx context.Context, id int64, approverID int64) error
        Reject(ctx context.Context, id int64, approverID int64, reason string) error
}

// VersionReader قرارداد خواندن نسخه فرم برای اعتبارسنجی داده گزارش.
type VersionReader interface {
        VersionByID(ctx context.Context, id int64) (application.FormVersion, error)
}

type AuditLogger interface {
        Log(ctx context.Context, e audit.Entry) error
}

// AssetChecker بررسی وجود تجهیزها (پیاده‌سازی در bootstrap).
type AssetChecker interface {
        CountExisting(ctx context.Context, ids []int64) (int, error)
}

type Service struct {
        repo     Repository
        versions VersionReader
        assets   AssetChecker
        audit    AuditLogger
}

func NewService(r Repository, versions VersionReader, assets AssetChecker, auditLog AuditLogger) *Service {
        return &Service{repo: r, versions: versions, assets: assets, audit: auditLog}
}

type ListFilter struct {
        Status string
        Query  string
        Page   httpx.Page
}

func (s *Service) List(ctx context.Context, f ListFilter) (httpx.Result[ReportListItem], error) {
        if f.Status != "" && !reportsdomain.ValidStatus(f.Status) {
                return httpx.Result[ReportListItem]{}, httpx.BadRequest("وضعیت گزارش نامعتبر است")
        }
        items, total, err := s.repo.List(ctx, f)
        if err != nil {
                return httpx.Result[ReportListItem]{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return httpx.NewResult(items, total, f.Page), nil
}

type CreateInput struct {
        Title                 string
        FormTemplateVersionID int64
        FormData              map[string]any
        AssetIDs              []int64
        ReportDate            string
}

func (s *Service) Create(ctx context.Context, actor *auth.AuthUser, in CreateInput) (int64, error) {
        if err := reportsdomain.ValidateTitle(in.Title); err != nil {
                return 0, httpx.BadRequest(err.Error())
        }
        if in.FormTemplateVersionID <= 0 {
                return 0, httpx.BadRequest("انتخاب فرم الزامی است")
        }
        if in.ReportDate == "" {
                return 0, httpx.BadRequest("تاریخ گزارش الزامی است")
        }
        if _, err := time.Parse("2006-01-02", in.ReportDate); err != nil {
                return 0, httpx.BadRequest("تاریخ گزارش معتبر نیست")
        }

        version, err := s.versions.VersionByID(ctx, in.FormTemplateVersionID)
        if err != nil {
                var appErr *httpx.AppError
                if errors.As(err, &appErr) {
                        return 0, httpx.BadRequest("نسخه فرم انتخابی معتبر نیست")
                }
                return 0, err
        }
        if in.FormData == nil {
                in.FormData = map[string]any{}
        }
        if err := domain.ValidateFormData(version.Fields, in.FormData); err != nil {
                return 0, httpx.BadRequest(err.Error())
        }
        if len(in.AssetIDs) > 0 {
                n, err := s.assets.CountExisting(ctx, in.AssetIDs)
                if err != nil {
                        return 0, httpx.Internal("بررسی تجهیزها ناموفق بود", err)
                }
                if int(n) != len(unique(in.AssetIDs)) {
                        return 0, httpx.BadRequest("برخی تجهیزهای انتخابی وجود ندارند")
                }
        }

        id, err := s.repo.Create(ctx, in, actor.ID)
        if err != nil {
                return 0, httpx.Internal("ثبت گزارش ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "create", id, nil,
                map[string]any{"title": in.Title, "version_id": in.FormTemplateVersionID})
        return id, nil
}

func (s *Service) Detail(ctx context.Context, id int64) (ReportDetail, error) {
        d, err := s.repo.GetDetail(ctx, id)
        if errors.Is(err, reportsdomain.ErrNotFound) {
                return ReportDetail{}, httpx.NotFound("گزارش یافت نشد")
        }
        if err != nil {
                return ReportDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return d, nil
}

// Approve تایید گزارش توسط کاربری غیر از ثبت‌کننده.
func (s *Service) Approve(ctx context.Context, actor *auth.AuthUser, id int64) error {
        if err := s.repo.Approve(ctx, id, actor.ID); err != nil {
                switch {
                case errors.Is(err, reportsdomain.ErrNotFound):
                        return httpx.NotFound("گزارش یافت نشد")
                case errors.Is(err, reportsdomain.ErrNotPending):
                        return httpx.Conflict("فقط گزارش در انتظار تایید قابل بررسی است")
                case errors.Is(err, reportsdomain.ErrSelfApproval):
                        return httpx.BadRequest("بررسی گزارش ثبت‌شده توسط خودتان مجاز نیست")
                }
                return httpx.Internal("تایید گزارش ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "approve", id, nil, map[string]any{"status": "approved"})
        return nil
}

// Reject رد گزارش با دلیل برای ثبت‌کننده.
func (s *Service) Reject(ctx context.Context, actor *auth.AuthUser, id int64, reason string) error {
        if len([]rune(strings.TrimSpace(reason))) < 3 {
                return httpx.BadRequest("دلیل رد گزارش الزامی است")
        }
        if err := s.repo.Reject(ctx, id, actor.ID, strings.TrimSpace(reason)); err != nil {
                switch {
                case errors.Is(err, reportsdomain.ErrNotFound):
                        return httpx.NotFound("گزارش یافت نشد")
                case errors.Is(err, reportsdomain.ErrNotPending):
                        return httpx.Conflict("فقط گزارش در انتظار تایید قابل بررسی است")
                case errors.Is(err, reportsdomain.ErrSelfApproval):
                        return httpx.BadRequest("بررسی گزارش ثبت‌شده توسط خودتان مجاز نیست")
                }
                return httpx.Internal("رد گزارش ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "reject", id, nil,
                map[string]any{"status": "rejected", "reason": reason})
        return nil
}

func (s *Service) auditLog(ctx context.Context, actor *auth.AuthUser, action string, id int64, before, after any) {
        if s.audit == nil {
                return
        }
        e := audit.Entry{
                Module: audit.ModReport, Action: action, EntityType: "report",
                EntityID: appkit.Int64ToStr(id), Before: before, After: after,
        }
        if actor != nil {
                e.ActorID = &actor.ID
                e.ActorUsername = actor.Username
        }
        if err := s.audit.Log(ctx, e); err != nil {
                slog.Error("audit write failed", "module", "report", "action", action, "err", err)
        }
}

func unique(ids []int64) []int64 {
        seen := map[int64]bool{}
        out := make([]int64, 0, len(ids))
        for _, id := range ids {
                if !seen[id] {
                        seen[id] = true
                        out = append(out, id)
                }
        }
        return out
}
