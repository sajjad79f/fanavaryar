// Package application — سناریوهای ماژول فرم‌ها: قالب، نسخه‌بندی تغییرناپذیر.
package application

import (
        "context"
        "encoding/json"
        "errors"
        "log/slog"
        "regexp"
        "strings"
        "time"

        "fanavaryar/internal/modules/forms/domain"
        "fanavaryar/internal/platform/appkit"
        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

// FormTemplate یک قالب فرم.
type FormTemplate struct {
        ID               int64     `json:"id"`
        Code             string    `json:"code"`
        Name             string    `json:"name"`
        Description      *string   `json:"description"`
        IsActive         bool      `json:"is_active"`
        CurrentVersionID *int64    `json:"current_version_id"`
        CreatedAt        time.Time `json:"created_at"`
}

// FormVersion یک نسخه تغییرناپذیر از فیلدهای فرم.
type FormVersion struct {
        ID            int64            `json:"id"`
        TemplateID    int64            `json:"template_id"`
        VersionNumber int              `json:"version_number"`
        Fields        []domain.FieldDef `json:"fields"`
        CreatedByName *string          `json:"created_by_name"`
        CreatedAt     time.Time        `json:"created_at"`
}

// FormDetail قالب + همه نسخه‌ها.
type FormDetail struct {
        FormTemplate
        Versions []FormVersion `json:"versions"`
}

type Repository interface {
        List(ctx context.Context) ([]FormTemplate, error)
        GetByID(ctx context.Context, id int64) (FormTemplate, error)
        Create(ctx context.Context, in CreateInput, fields []domain.FieldDef, actorID *int64) (int64, error)
        SetActive(ctx context.Context, id int64, active bool) error
        CreateVersion(ctx context.Context, templateID int64, fields []domain.FieldDef, actorID *int64) (int64, int, error)
        VersionByID(ctx context.Context, id int64) (FormVersion, error)
        ListVersions(ctx context.Context, templateID int64) ([]FormVersion, error)
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

type CreateInput struct {
        Code        string
        Name        string
        Description *string
        Fields      []domain.FieldDef
}

func (s *Service) List(ctx context.Context) ([]FormTemplate, error) {
        items, err := s.repo.List(ctx)
        if err != nil {
                return nil, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if items == nil {
                items = []FormTemplate{}
        }
        return items, nil
}

func (s *Service) Create(ctx context.Context, actor *auth.AuthUser, in CreateInput) (int64, error) {
        code := trimSpace(in.Code)
        name := trimSpace(in.Name)
        if code == "" || name == "" {
                return 0, httpx.BadRequest("کد و نام فرم الزامی است")
        }
        if len(code) > 60 || !isSlug(code) {
                return 0, httpx.BadRequest("کد فرم باید انگلیسی، کوتاه و بدون فاصله باشد")
        }
        if err := domain.ValidateFields(in.Fields); err != nil {
                return 0, httpx.BadRequest(err.Error())
        }
        id, err := s.repo.Create(ctx, CreateInput{Code: code, Name: name, Description: in.Description, Fields: in.Fields},
                in.Fields, actorID(actor))
        if err != nil {
                if errors.Is(err, domain.ErrCodeTaken) {
                        return 0, httpx.Conflict("این کد فرم قبلاً ثبت شده است")
                }
                return 0, httpx.Internal("ایجاد فرم ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "create", id, nil, map[string]any{"code": code, "name": name})
        return id, nil
}

func (s *Service) Detail(ctx context.Context, id int64) (FormDetail, error) {
        t, err := s.repo.GetByID(ctx, id)
        if errors.Is(err, domain.ErrNotFound) {
                return FormDetail{}, httpx.NotFound("فرم یافت نشد")
        }
        if err != nil {
                return FormDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        versions, err := s.repo.ListVersions(ctx, id)
        if err != nil {
                return FormDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if versions == nil {
                versions = []FormVersion{}
        }
        return FormDetail{FormTemplate: t, Versions: versions}, nil
}

// SetActive انتشار/غیرفعال‌سازی فرم؛ مجوز جهتِ تغییر جدا بررسی می‌شود.
func (s *Service) SetActive(ctx context.Context, actor *auth.AuthUser, id int64, active bool) error {
        code := "forms.disable"
        if active {
                code = "forms.publish"
        }
        if !actor.Has(code) {
                return httpx.Forbidden("شما مجوز لازم برای این عملیات را ندارید")
        }
        if err := s.repo.SetActive(ctx, id, active); err != nil {
                if errors.Is(err, domain.ErrNotFound) {
                        return httpx.NotFound("فرم یافت نشد")
                }
                return httpx.Internal("تغییر وضعیت فرم ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "update", id, nil, map[string]any{"is_active": active})
        return nil
}

func (s *Service) CreateVersion(ctx context.Context, actor *auth.AuthUser, templateID int64,
        fields []domain.FieldDef) (int64, int, error) {
        if err := domain.ValidateFields(fields); err != nil {
                return 0, 0, httpx.BadRequest(err.Error())
        }
        id, ver, err := s.repo.CreateVersion(ctx, templateID, fields, actorID(actor))
        if err != nil {
                if errors.Is(err, domain.ErrNotFound) {
                        return 0, 0, httpx.NotFound("فرم یافت نشد")
                }
                return 0, 0, httpx.Internal("ثبت نسخه ناموفق بود", err)
        }
        s.auditLog(ctx, actor, "new_version", templateID,
                nil, map[string]any{"version_id": id, "version_number": ver})
        return id, ver, nil
}

// VersionByID دریافت یک نسخه برای پر کردن فرم در ثبت گزارش.
func (s *Service) VersionByID(ctx context.Context, id int64) (FormVersion, error) {
        v, err := s.repo.VersionByID(ctx, id)
        if errors.Is(err, domain.ErrNotFound) {
                return FormVersion{}, httpx.NotFound("نسخه فرم یافت نشد")
        }
        if err != nil {
                return FormVersion{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return v, nil
}

func (s *Service) auditLog(ctx context.Context, actor *auth.AuthUser, action string,
        templateID int64, before, after any) {
        if s.audit == nil {
                return
        }
        e := audit.Entry{
                Module: audit.ModForm, Action: action, EntityType: "form_template",
                EntityID: appkit.Int64ToStr(templateID), Before: before, After: after,
        }
        if actor != nil {
                e.ActorID = &actor.ID
                e.ActorUsername = actor.Username
        }
        if err := s.audit.Log(ctx, e); err != nil {
                slog.Error("audit write failed", "module", "form", "action", action, "err", err)
        }
}

func (s *Service) marshalFields(fields []domain.FieldDef) (json.RawMessage, error) {
        b, err := json.Marshal(fields)
        if err != nil {
                return nil, err
        }
        return json.RawMessage(b), nil
}

func actorID(u *auth.AuthUser) *int64 {
        if u == nil {
                return nil
        }
        return &u.ID
}

func trimSpace(s string) string { return strings.TrimSpace(s) }

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,59}$`)

func isSlug(s string) bool { return slugRe.MatchString(s) }
