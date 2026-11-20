// Package application — سناریوهای ماژول تجهیزات (قرارداد v1).
package application

import (
        "context"
        "errors"
        "log/slog"
        "strings"
        "time"

        "fanavaryar/internal/modules/assets/domain"
        "fanavaryar/internal/platform/appkit"
        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

// ErrLocationNotFound خطای قراردادی خواندن مکان.
var ErrLocationNotFound = errors.New("assets: location not found")

// LocationSnapshot حداقل اطلاعاتی که ماژول تجهیزات درباره مکان لازم دارد.
type LocationSnapshot struct {
        ID   int64
        Kind string
        Name string
}

// LocationReader قرارداد خواندن مکان؛ پیاده‌سازی آن در bootstrap سیم‌کشی می‌شود.
type LocationReader interface {
        GetSnapshot(ctx context.Context, id int64) (LocationSnapshot, error)
}

// Asset ردیف فهرست/پرونده — کلیدهای JSON قرارداد فرانت‌اند.
type Asset struct {
        ID                int64     `json:"id"`
        AssetTag          string    `json:"asset_tag"`
        Name              string    `json:"name"`
        Category          string    `json:"category"`
        Status            string    `json:"status"`
        SerialNumber      *string   `json:"serial_number"`
        Manufacturer      *string   `json:"manufacturer"`
        Model             *string   `json:"model"`
        ResponsiblePerson *string   `json:"responsible_person"`
        CurrentLocationID *int64    `json:"current_location_id"`
        LocationName      *string   `json:"location_name"`
        Notes             *string   `json:"notes"`
        CreatedAt         time.Time `json:"created_at"`
        UpdatedAt         time.Time `json:"updated_at"`
}

// Movement یک جابه‌جایی با نام‌های خوانا.
type Movement struct {
        ID               int64     `json:"id"`
        FromLocationID   *int64    `json:"from_location_id"`
        FromLocationName *string   `json:"from_location_name"`
        ToLocationID     *int64    `json:"to_location_id"`
        ToLocationName   *string   `json:"to_location_name"`
        Reason           *string   `json:"reason"`
        MovedByName      *string   `json:"moved_by_name"`
        CreatedAt        time.Time `json:"created_at"`
}

// RelatedReport گزارش مرتبط در پرونده تجهیز.
type RelatedReport struct {
        ID         int64  `json:"id"`
        Code       *string `json:"code"`
        Title      string `json:"title"`
        ReportDate string `json:"report_date"`
        Status     string `json:"status"`
}

// AssetDetail پرونده کامل تجهیز: مشخصات + جابه‌جایی‌ها + گزارش‌های مرتبط.
type AssetDetail struct {
        Asset
        Movements []Movement      `json:"movements"`
        Reports   []RelatedReport `json:"reports"`
}

type Repository interface {
        Create(ctx context.Context, in CreateInput, actorID *int64) (Asset, error)
        Update(ctx context.Context, id int64, in UpdateInput, actorID *int64) (Asset, error)
        GetByID(ctx context.Context, id int64) (Asset, error)
        List(ctx context.Context, f ListFilter) ([]Asset, int, error)
        InsertMove(ctx context.Context, m MoveInput, actorID *int64) error
        ListMoves(ctx context.Context, assetID int64) ([]Movement, error)
        ListReports(ctx context.Context, assetID int64) ([]RelatedReport, error)
}

type AuditLogger interface {
        Log(ctx context.Context, e audit.Entry) error
}

type Service struct {
        repo      Repository
        locations LocationReader
        audit     AuditLogger
}

func NewService(repo Repository, locations LocationReader, auditLog AuditLogger) *Service {
        return &Service{repo: repo, locations: locations, audit: auditLog}
}

// ---------- محدوده اطلاعاتی (Data Scope) ----------

// ScopeFilter محدوده دید کاربر را از مجوز عملیات جدا می‌کند.
type ScopeFilter struct {
        Type  string // all | building | floor | room
        RefID *int64 // برای building/floor/room الزامی است
}

func ScopeOf(u *auth.AuthUser) ScopeFilter {
        if u == nil {
                return ScopeFilter{Type: "all"}
        }
        return ScopeFilter{Type: u.ScopeType, RefID: u.ScopeRefID}
}

// Valid دامنه‌های مکان‌محور بدون مرجع، به کل سازمان گسترش نمی‌یابند.
func (f ScopeFilter) Valid() bool {
        switch f.Type {
        case "all":
                return true
        case "building", "floor", "room":
                return f.RefID != nil && *f.RefID > 0
        }
        return false
}

type ListFilter struct {
        Scope      ScopeFilter
        Status     string
        Category   string
        LocationID *int64
        Query      string
        Page       httpx.Page
}

// List فهرست صفحه‌بندی‌شده تجهیزات با محدوده دید کاربر.
func (s *Service) List(ctx context.Context, actor *auth.AuthUser, f ListFilter) (httpx.Result[Asset], error) {
        f.Scope = ScopeOf(actor)
        if !f.Scope.Valid() {
                return httpx.Result[Asset]{}, httpx.Forbidden("محدوده اطلاعاتی حساب شما معتبر نیست؛ با مدیر سامانه تماس بگیرید")
        }
        items, total, err := s.repo.List(ctx, f)
        if err != nil {
                return httpx.Result[Asset]{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return httpx.NewResult(items, total, f.Page), nil
}

// ---------- CRUD ----------

type CreateInput struct {
        AssetTag          string
        Name              string
        Category          string
        Status            string
        SerialNumber      *string
        Manufacturer      *string
        Model             *string
        ResponsiblePerson *string
        CurrentLocationID *int64
        Notes             *string
}

func (s *Service) Create(ctx context.Context, actor *auth.AuthUser, in CreateInput) (Asset, error) {
        status := in.Status
        if status == "" {
                status = string(domain.StatusActive)
        }
        a := domain.Asset{
                AssetTag: in.AssetTag, Name: in.Name, Category: in.Category,
                Status: domain.Status(status),
        }
        if err := a.Validate(); err != nil {
                return Asset{}, httpx.BadRequest(err.Error())
        }
        if in.CurrentLocationID != nil {
                if err := s.checkPhysicalLocation(ctx, *in.CurrentLocationID); err != nil {
                        return Asset{}, err
                }
        }

        out, err := s.repo.Create(ctx, in, actorID(actor))
        if err != nil {
                if errors.Is(err, domain.ErrTagTaken) {
                        return Asset{}, httpx.Conflict("این شناسه تجهیز قبلاً ثبت شده است")
                }
                return Asset{}, httpx.Internal("ایجاد تجهیز ناموفق بود", err)
        }
        s.emit(ctx, actor, "create", nil, out, out.ID)
        return out, nil
}

// UpdateInput ویرایش جزئی — فیلدهای ارسال‌شده فقط به‌روزرسانی می‌شوند؛
// رشته خالی یعنی پاک‌کردن مقدار (NULL)، nil یعنی بدون تغییر.
type UpdateInput struct {
        AssetTag          *string
        Name              *string
        Category          *string
        Status            *string
        SerialNumber      *string
        Manufacturer      *string
        Model             *string
        ResponsiblePerson *string
        CurrentLocationID *int64
        Notes             *string
}

// MoveInput درخواست جابه‌جایی تجهیز.
type MoveInput struct {
        AssetID      int64
        ToLocationID int64
        Reason       *string
}

func (s *Service) Update(ctx context.Context, actor *auth.AuthUser, id int64, in UpdateInput) (Asset, error) {
        before, err := s.Get(ctx, id)
        if err != nil {
                return Asset{}, err
        }

        // اعتبارسنجی مقادیر ارسال‌شده روی یک نسخه کاری
        work := domain.Asset{
                AssetTag: before.AssetTag, Name: before.Name, Category: before.Category,
                Status: domain.Status(before.Status),
        }
        if in.AssetTag != nil {
                work.AssetTag = *in.AssetTag
        }
        if in.Name != nil {
                work.Name = *in.Name
        }
        if in.Category != nil {
                work.Category = *in.Category
        }
        if in.Status != nil {
                work.Status = domain.Status(*in.Status)
        }
        if err := work.Validate(); err != nil {
                return Asset{}, httpx.BadRequest(err.Error())
        }
        if in.CurrentLocationID != nil {
                if err := s.checkPhysicalLocation(ctx, *in.CurrentLocationID); err != nil {
                        return Asset{}, err
                }
        }

        after, err := s.repo.Update(ctx, id, in, actorID(actor))
        if err != nil {
                if errors.Is(err, domain.ErrTagTaken) {
                        return Asset{}, httpx.Conflict("این شناسه تجهیز قبلاً ثبت شده است")
                }
                if errors.Is(err, domain.ErrNotFound) {
                        return Asset{}, httpx.NotFound("تجهیز یافت نشد")
                }
                return Asset{}, httpx.Internal("ذخیره تغییرات ناموفق بود", err)
        }
        s.emit(ctx, actor, "update", before, after, id)
        return after, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Asset, error) {
        a, err := s.repo.GetByID(ctx, id)
        if errors.Is(err, domain.ErrNotFound) {
                return Asset{}, httpx.NotFound("تجهیز یافت نشد")
        }
        if err != nil {
                return Asset{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return a, nil
}

// Detail پرونده کامل تجهیز برای صفحه جزئیات.
func (s *Service) Detail(ctx context.Context, id int64) (AssetDetail, error) {
        a, err := s.Get(ctx, id)
        if err != nil {
                return AssetDetail{}, err
        }
        moves, err := s.repo.ListMoves(ctx, id)
        if err != nil {
                return AssetDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        reports, err := s.repo.ListReports(ctx, id)
        if err != nil {
                return AssetDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if moves == nil {
                moves = []Movement{}
        }
        if reports == nil {
                reports = []RelatedReport{}
        }
        return AssetDetail{Asset: a, Movements: moves, Reports: reports}, nil
}

// Move جابه‌جایی تجهیز به اتاق/رک دیگر با ثبت تاریخچه کامل.
func (s *Service) Move(ctx context.Context, actor *auth.AuthUser, id int64, toLocationID int64, reason *string) error {
        a, err := s.Get(ctx, id)
        if err != nil {
                return err
        }
        if a.CurrentLocationID != nil && *a.CurrentLocationID == toLocationID {
                return httpx.BadRequest("مکان مقصد با مکان فعلی یکسان است")
        }
        if err := s.checkPhysicalLocation(ctx, toLocationID); err != nil {
                return err
        }
        if err := s.repo.InsertMove(ctx, MoveInput{AssetID: id, ToLocationID: toLocationID, Reason: reason}, actorID(actor)); err != nil {
                return httpx.Internal("ثبت جابه‌جایی ناموفق بود", err)
        }
        s.emit(ctx, actor, "move",
                map[string]any{"from_location_id": a.CurrentLocationID},
                map[string]any{"to_location_id": toLocationID, "reason": reason}, id)
        return nil
}

// checkPhysicalLocation مکان مقصد باید «اتاق» یا «رک» باشد و وجود داشته باشد.
func (s *Service) checkPhysicalLocation(ctx context.Context, locationID int64) error {
        if s.locations == nil {
                return nil // در برخی تست‌ها ریدر تزریق نمی‌شود
        }
        snap, err := s.locations.GetSnapshot(ctx, locationID)
        if err != nil {
                if errors.Is(err, ErrLocationNotFound) {
                        return httpx.BadRequest("مکان مقصد یافت نشد")
                }
                return httpx.Internal("بررسی مکان مقصد ناموفق بود", err)
        }
        for _, k := range domain.PhysicalKinds() {
                if snap.Kind == k {
                        return nil
                }
        }
        return httpx.BadRequest("محل فیزیکی تجهیز باید «اتاق» یا «رک» باشد")
}

func (s *Service) emit(ctx context.Context, actor *auth.AuthUser, action string,
        before, after any, id int64) {
        if s.audit == nil {
                return
        }
        e := audit.Entry{
                Module: audit.ModAsset, Action: action, EntityType: "asset",
                EntityID: appkit.Int64ToStr(id), Before: before, After: after,
        }
        if actor != nil {
                e.ActorID = actorID(actor)
                e.ActorUsername = actor.Username
        }
        if err := s.audit.Log(ctx, e); err != nil {
                slog.Error("audit write failed", "module", "asset", "action", action, "err", err)
        }
}

func actorID(u *auth.AuthUser) *int64 {
        if u == nil {
                return nil
        }
        return &u.ID
}

func trimPtr(s string) *string {
        t := strings.TrimSpace(s)
        if t == "" {
                return nil
        }
        return &t
}
