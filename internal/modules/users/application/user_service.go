// Package application — سناریوهای ماژول کاربران (قرارداد v1).
package application

import (
        "context"
        "errors"
        "log/slog"
        "strings"
        "time"

        "fanavaryar/internal/modules/users/domain"
        "fanavaryar/internal/platform/appkit"
        "fanavaryar/internal/platform/audit"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

// Page همان قرارداد صفحه‌بندی httpx.
type Page = httpx.Page

// UserItem ردیف فهرست کاربران — کلیدهای JSON دقیقاً قرارداد فرانت‌اند.
type UserItem struct {
        ID        int64     `json:"id"`
        Username  string    `json:"username"`
        FullName  string    `json:"full_name"`
        JobTitle  *string   `json:"job_title"`
        Role      string    `json:"role"`
        IsActive  bool      `json:"is_active"`
        CreatedAt time.Time `json:"created_at"`
}

// Scope محدوده اطلاعاتی کاربر.
type Scope struct {
        ScopeType  string `json:"scope_type"`
        LocationID *int64 `json:"location_id"`
}

// UserDetail جزئیات کاربر برای مودال دسترسی‌ها.
type UserDetail struct {
        UserItem
        Permissions []string `json:"permissions"`
        Scope       *Scope   `json:"scope"`
}

// AuthRow ردیف کامل کاربر برای ورود — شامل هش رمز و محدوده.
type AuthRow struct {
	ID           int64
	Username     string
	FullName     string
	JobTitle     string
	Role         string
	IsSuperadmin bool
	IsActive     bool
	PasswordHash string
	ScopeType    string
	ScopeRefID   *int64
}

// PermDef یک مجوز کاتالوگ.
type PermDef struct {
        Code   string `json:"code"`
        Module string `json:"module"`
        Action string `json:"action"`
        Title  string `json:"title"`
        Sort   int    `json:"sort"`
}

// PermTemplate الگوی دسترسی.
type PermTemplate struct {
        ID          int64     `json:"id"`
        Name        string    `json:"name"`
        CreatedAt   time.Time `json:"created_at"`
        Permissions []string  `json:"permissions"`
}

type Repository interface {
        GetByUsername(ctx context.Context, username string) (AuthRow, error)
        ExistsByUsername(ctx context.Context, username string) (bool, error)
        RoleOf(ctx context.Context, userID int64) (string, bool, error)
        Create(ctx context.Context, in CreateInput, hash string) (UserItem, error)
        UpdateProfile(ctx context.Context, id int64, fullName string, jobTitle *string, isActive *bool) (UserItem, error)
        GetByID(ctx context.Context, id int64) (UserItem, error)
        SetPasswordHash(ctx context.Context, id int64, hash string) error
        ScopeOf(ctx context.Context, id int64) (string, *int64, error)
        SetScope(ctx context.Context, id int64, scopeType string, refID *int64) error
        List(ctx context.Context, q string, p Page) ([]UserItem, int, error)
        PasswordHashOf(ctx context.Context, id int64) (string, error)
        PermissionsOf(ctx context.Context, userID int64) ([]string, error)
        CountValidCodes(ctx context.Context, codes []string) (int, error)
        ReplacePermissions(ctx context.Context, userID, grantedBy int64, codes []string) error
        Catalog(ctx context.Context) ([]PermDef, error)
        Templates(ctx context.Context) ([]PermTemplate, error)
        CreateTemplate(ctx context.Context, name string, codes []string) (int64, error)
        DeleteTemplate(ctx context.Context, id int64) error
}

// ScopeLocationChecker بررسی اینکه مکانِ انتخابی برای محدوده، از نوع درست است.
// پیاده‌سازی آن در bootstrap به ماژول Locations وصل می‌شود.
type ScopeLocationChecker interface {
        ScopeLocationKind(ctx context.Context, id int64) (string, error)
}

type PasswordHasher interface {
        Hash(password string) (string, error)
        Verify(password, encoded string) (bool, error)
}

type AuditLogger interface {
        Log(ctx context.Context, e audit.Entry) error
}

type Service struct {
        repo      Repository
        hasher    PasswordHasher
        sessions  *auth.SessionService
        audit     AuditLogger
        scopeLocs ScopeLocationChecker
}

func NewService(repo Repository, hasher PasswordHasher, sessions *auth.SessionService,
        auditLog AuditLogger, scopeLocs ScopeLocationChecker) *Service {
        return &Service{repo: repo, hasher: hasher, sessions: sessions,
                audit: auditLog, scopeLocs: scopeLocs}
}

// ---------- ورود و نشست ----------

// Login ورود با نام کاربری/رمز؛ توکن نشست در کوکی HttpOnly ست می‌شود (سمت handler).
func (s *Service) Login(ctx context.Context, username, password, ip string,
        ttl time.Duration) (*auth.AuthUser, string, error) {

        row, err := s.repo.GetByUsername(ctx, username)
        if errors.Is(err, domain.ErrNotFound) {
                s.auditLog(ctx, nil, username, audit.ModAuth, "login_failed", "", nil, nil, nil, ip)
                return nil, "", httpx.Unauthorized("نام کاربری یا رمز عبور نادرست است")
        }
        if err != nil {
                return nil, "", httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if !row.IsActive {
                return nil, "", httpx.Unauthorized("این حساب غیرفعال شده است؛ با مدیر سامانه تماس بگیرید")
        }
        ok, err := s.hasher.Verify(password, row.PasswordHash)
        if err != nil || !ok {
                s.auditLog(ctx, &row.ID, row.Username, audit.ModAuth, "login_failed", "", nil, nil, nil, ip)
                return nil, "", httpx.Unauthorized("نام کاربری یا رمز عبور نادرست است")
        }

        token, err := s.sessions.Create(ctx, row.ID, ttl)
        if err != nil {
                return nil, "", httpx.Internal("ایجاد نشست ناموفق بود", err)
        }

        u, err := s.buildAuthUser(ctx, row)
        if err != nil {
                return nil, "", err
        }
        s.auditLog(ctx, &row.ID, row.Username, audit.ModAuth, "login", "", nil, nil, nil, ip)
        return u, token, nil
}

func (s *Service) buildAuthUser(ctx context.Context, row AuthRow) (*auth.AuthUser, error) {
        perms, err := s.repo.PermissionsOf(ctx, row.ID)
        if err != nil {
                return nil, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return &auth.AuthUser{
                ID:           row.ID,
                Username:     row.Username,
                FullName:     row.FullName,
                JobTitle:     row.JobTitle,
                Role:         row.Role,
                IsSuperadmin: row.IsSuperadmin,
                ScopeType:    row.ScopeType,
                ScopeRefID:   row.ScopeRefID,
                Perms:        perms,
        }, nil
}

func (s *Service) Me(ctx context.Context, actor *auth.AuthUser) *auth.AuthUser { return actor }

// ChangePassword تغییر رمز توسط خود کاربر؛ همه نشست‌ها باطل و نشست تازه صادر می‌شود.
func (s *Service) ChangePassword(ctx context.Context, actor *auth.AuthUser,
        oldPassword, newPassword, ip string, ttl time.Duration) (string, error) {

        hash, err := s.repo.PasswordHashOf(ctx, actor.ID)
        if err != nil {
                return "", httpx.Internal("خطای ذخیره‌سازی", err)
        }
        ok, err := s.hasher.Verify(oldPassword, hash)
        if err != nil || !ok {
                return "", httpx.BadRequest("گذرواژه فعلی نادرست است")
        }
        if err := domain.ValidatePassword(newPassword); err != nil {
                return "", httpx.BadRequest(err.Error())
        }
        newHash, err := s.hasher.Hash(newPassword)
        if err != nil {
                return "", httpx.Internal("خطای پردازش رمز", err)
        }
        if err := s.repo.SetPasswordHash(ctx, actor.ID, newHash); err != nil {
                return "", httpx.Internal("ذخیره رمز ناموفق بود", err)
        }
        if err := s.sessions.RevokeAllForUser(ctx, actor.ID); err != nil {
                return "", httpx.Internal("ابطال نشست‌ها ناموفق بود", err)
        }
        token, err := s.sessions.Create(ctx, actor.ID, ttl)
        if err != nil {
                return "", httpx.Internal("ایجاد نشست ناموفق بود", err)
        }
        s.auditLog(ctx, &actor.ID, actor.Username, audit.ModAuth, "change_password", "user",
                appkit.Int64ToStr(actor.ID), nil, nil, ip)
        return token, nil
}

// Logout خروج؛ ابطال نشست جاری.
func (s *Service) Logout(ctx context.Context, actor *auth.AuthUser, token, ip string) {
        if token != "" {
                _ = s.sessions.Revoke(ctx, token)
        }
        if actor != nil {
                s.auditLog(ctx, &actor.ID, actor.Username, audit.ModAuth, "logout", "", nil, nil, nil, ip)
        }
}

// ---------- کاربران ----------

type CreateInput struct {
        Username string
        FullName string
        JobTitle string
        Password string
}

func (s *Service) List(ctx context.Context, q string, p Page) (httpx.Result[UserItem], error) {
        items, total, err := s.repo.List(ctx, strings.TrimSpace(q), p)
        if err != nil {
                return httpx.Result[UserItem]{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        return httpx.NewResult(items, total, p), nil
}

func (s *Service) Create(ctx context.Context, actor *auth.AuthUser, in CreateInput) (UserItem, error) {
        if err := domain.ValidateUsername(in.Username); err != nil {
                return UserItem{}, httpx.BadRequest(err.Error())
        }
        if err := domain.ValidateFullName(in.FullName); err != nil {
                return UserItem{}, httpx.BadRequest(err.Error())
        }
        if err := domain.ValidatePassword(in.Password); err != nil {
                return UserItem{}, httpx.BadRequest(err.Error())
        }
        taken, err := s.repo.ExistsByUsername(ctx, in.Username)
        if err != nil {
                return UserItem{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if taken {
                return UserItem{}, httpx.Conflict("این نام کاربری قبلاً ثبت شده است")
        }
        hash, err := s.hasher.Hash(in.Password)
        if err != nil {
                return UserItem{}, httpx.Internal("خطای پردازش رمز", err)
        }
        u, err := s.repo.Create(ctx, in, hash)
        if err != nil {
                if errors.Is(err, domain.ErrUsernameTaken) {
                        return UserItem{}, httpx.Conflict("این نام کاربری قبلاً ثبت شده است")
                }
                return UserItem{}, httpx.Internal("ایجاد کاربر ناموفق بود", err)
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModUser, "create", "user",
                appkit.Int64ToStr(u.ID), nil, map[string]any{"username": u.Username, "id": u.ID}, "")
        return u, nil
}

// guardTarget بررسی مجازبودن تغییر حساب هدف:
// حساب admin فقط توسط ابرمدیر تغییر می‌کند و تغییر وضعیت خودِ کاربر ممنوع است.
func (s *Service) guardTarget(ctx context.Context, actor *auth.AuthUser, targetID int64, activeChange *bool) error {
        role, _, err := s.repo.RoleOf(ctx, targetID)
        if errors.Is(err, domain.ErrNotFound) {
                return httpx.NotFound("کاربر یافت نشد")
        }
        if err != nil {
                return httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if role == "admin" && !actor.IsSuperadmin {
                return httpx.Forbidden("حساب مدیر سامانه فقط توسط ابرمدیر قابل تغییر است")
        }
        if activeChange != nil && targetID == actor.ID && !*activeChange {
                return httpx.BadRequest("تغییر وضعیت حساب خودتان مجاز نیست")
        }
        return nil
}

func (s *Service) Get(ctx context.Context, id int64) (UserDetail, error) {
        u, err := s.repo.GetByID(ctx, id)
        if errors.Is(err, domain.ErrNotFound) {
                return UserDetail{}, httpx.NotFound("کاربر یافت نشد")
        }
        if err != nil {
                return UserDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        perms, err := s.repo.PermissionsOf(ctx, id)
        if err != nil {
                return UserDetail{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        scope, err := s.scopeOf(ctx, id)
        if err != nil {
                return UserDetail{}, err
        }
        return UserDetail{UserItem: u, Permissions: perms, Scope: scope}, nil
}

func (s *Service) scopeOf(ctx context.Context, id int64) (*Scope, error) {
        st, ref, err := s.repo.ScopeOf(ctx, id)
        if err != nil {
                return nil, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if st == "" {
                return nil, nil
        }
        return &Scope{ScopeType: st, LocationID: ref}, nil
}

type ProfileInput struct {
        FullName string
        JobTitle string
        IsActive *bool
}

func (s *Service) UpdateProfile(ctx context.Context, actor *auth.AuthUser, id int64, in ProfileInput) (UserItem, error) {
        if err := s.guardTarget(ctx, actor, id, in.IsActive); err != nil {
                return UserItem{}, err
        }
        if err := domain.ValidateFullName(in.FullName); err != nil {
                return UserItem{}, httpx.BadRequest(err.Error())
        }
        var jobTitle *string
        if t := strings.TrimSpace(in.JobTitle); t != "" {
                jobTitle = &t
        }
        before, err := s.repo.GetByID(ctx, id)
        if errors.Is(err, domain.ErrNotFound) {
                return UserItem{}, httpx.NotFound("کاربر یافت نشد")
        }
        if err != nil {
                return UserItem{}, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        u, err := s.repo.UpdateProfile(ctx, id, in.FullName, jobTitle, in.IsActive)
        if err != nil {
                return UserItem{}, httpx.Internal("ذخیره تغییرات ناموفق بود", err)
        }
        // غیرفعال‌سازی حساب، همه نشست‌های آن کاربر را باطل می‌کند
        if in.IsActive != nil && !*in.IsActive && before.IsActive {
                _ = s.sessions.RevokeAllForUser(ctx, id)
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModUser, "update", "user",
                appkit.Int64ToStr(id), map[string]any{"full_name": before.FullName, "job_title": before.JobTitle},
                map[string]any{"full_name": u.FullName, "job_title": u.JobTitle, "is_active": u.IsActive}, "")
        return u, nil
}

func (s *Service) ResetPassword(ctx context.Context, actor *auth.AuthUser, id int64, newPassword, ip string) error {
        if err := s.guardTarget(ctx, actor, id, nil); err != nil {
                return err
        }
        if err := domain.ValidatePassword(newPassword); err != nil {
                return httpx.BadRequest(err.Error())
        }
        hash, err := s.hasher.Hash(newPassword)
        if err != nil {
                return httpx.Internal("خطای پردازش رمز", err)
        }
        if err := s.repo.SetPasswordHash(ctx, id, hash); err != nil {
                return httpx.Internal("بازنشانی رمز ناموفق بود", err)
        }
        // طبق قرارداد UI: همه نشست‌های فعال این کاربر باطل می‌شود
        if err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
                return httpx.Internal("ابطال نشست‌ها ناموفق بود", err)
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModUser, "reset_password", "user",
                appkit.Int64ToStr(id), nil, nil, ip)
        return nil
}

// SetPermissions جایگزینی کامل مجوزهای کاربر با کدهای داده‌شده.
func (s *Service) SetPermissions(ctx context.Context, actor *auth.AuthUser, id int64, codes []string) error {
        if err := s.guardTarget(ctx, actor, id, nil); err != nil {
                return err
        }
        role, _, err := s.repo.RoleOf(ctx, id)
        if err != nil {
                return httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if role == "admin" {
                return httpx.Forbidden("مجوزهای حساب مدیر سامانه قابل تغییر نیست")
        }
        clean := make([]string, 0, len(codes))
        for _, c := range codes {
                if c = strings.TrimSpace(c); c != "" {
                        clean = append(clean, c)
                }
        }
        if len(clean) > 0 {
                valid, err := s.repo.CountValidCodes(ctx, clean)
                if err != nil {
                        return httpx.Internal("خطای ذخیره‌سازی", err)
                }
                if valid != len(clean) {
                        return httpx.BadRequest("برخی کدهای مجوز معتبر نیستند")
                }
        }
        if err := s.repo.ReplacePermissions(ctx, id, actor.ID, clean); err != nil {
                return httpx.Internal("ذخیره مجوزها ناموفق بود", err)
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModUser, "set_permissions", "user",
                appkit.Int64ToStr(id), nil, map[string]any{"permission_codes": clean}, "")
        return nil
}

// SetScope تعیین محدوده اطلاعاتی کاربر؛ نوع محدوده باید با نوع مکان هم‌خوان باشد.
func (s *Service) SetScope(ctx context.Context, actor *auth.AuthUser, id int64, scopeType string, locID *int64) error {
        if err := s.guardTarget(ctx, actor, id, nil); err != nil {
                return err
        }
        role, _, err := s.repo.RoleOf(ctx, id)
        if err != nil {
                return httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if role == "admin" {
                return httpx.Forbidden("محدوده حساب مدیر سامانه قابل تغییر نیست")
        }
        if err := domain.ValidateScope(scopeType, locID); err != nil {
                return httpx.BadRequest(err.Error())
        }
        if locID != nil && s.scopeLocs != nil {
                kind, err := s.scopeLocs.ScopeLocationKind(ctx, *locID)
                if errors.Is(err, domain.ErrNotFound) || (err == nil && kind != scopeType) {
                        return httpx.BadRequest("مکان انتخابی با نوع محدوده هم‌خوان نیست")
                }
                if err != nil {
                        return httpx.Internal("بررسی مکان ناموفق بود", err)
                }
        }
        if err := s.repo.SetScope(ctx, id, scopeType, locID); err != nil {
                return httpx.Internal("ذخیره محدوده ناموفق بود", err)
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModUser, "set_scope", "user",
                appkit.Int64ToStr(id), nil,
                map[string]any{"scope_type": scopeType, "location_id": locID}, "")
        return nil
}

// ---------- کاتالوگ و الگوها ----------

func (s *Service) Catalog(ctx context.Context) ([]PermDef, error) {
        items, err := s.repo.Catalog(ctx)
        if err != nil {
                return nil, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if items == nil {
                items = []PermDef{}
        }
        return items, nil
}

func (s *Service) Templates(ctx context.Context) ([]PermTemplate, error) {
        t, err := s.repo.Templates(ctx)
        if err != nil {
                return nil, httpx.Internal("خطای ذخیره‌سازی", err)
        }
        if t == nil {
                t = []PermTemplate{}
        }
        return t, nil
}

func (s *Service) CreateTemplate(ctx context.Context, actor *auth.AuthUser, name string, codes []string) (int64, error) {
        name = strings.TrimSpace(name)
        if len([]rune(name)) < 2 || len([]rune(name)) > 80 {
                return 0, httpx.BadRequest("نام الگو باید ۲ تا ۸۰ کاراکتر باشد")
        }
        clean := make([]string, 0, len(codes))
        for _, c := range codes {
                if c = strings.TrimSpace(c); c != "" {
                        clean = append(clean, c)
                }
        }
        if len(clean) > 0 {
                valid, err := s.repo.CountValidCodes(ctx, clean)
                if err != nil {
                        return 0, httpx.Internal("خطای ذخیره‌سازی", err)
                }
                if valid != len(clean) {
                        return 0, httpx.BadRequest("برخی کدهای مجوز معتبر نیستند")
                }
        }
        id, err := s.repo.CreateTemplate(ctx, name, clean)
        if err != nil {
                return 0, httpx.Conflict("ثبت الگو ناموفق بود؛ شاید نام تکراری باشد")
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModPerm, "template_create", "permission_template",
                appkit.Int64ToStr(id), nil, map[string]any{"name": name, "permission_codes": clean}, "")
        return id, nil
}

func (s *Service) DeleteTemplate(ctx context.Context, actor *auth.AuthUser, id int64) error {
        if err := s.repo.DeleteTemplate(ctx, id); err != nil {
                if errors.Is(err, domain.ErrNotFound) {
                        return httpx.NotFound("الگو یافت نشد")
                }
                return httpx.Internal("حذف الگو ناموفق بود", err)
        }
        s.auditLog(ctx, actorID(actor), appkit.ActorUsername(actor), audit.ModPerm, "template_delete",
                "permission_template", appkit.Int64ToStr(id), nil, nil, "")
        return nil
}

// ---------- ابزار ----------

func (s *Service) auditLog(ctx context.Context, actorID *int64, username, module, action,
        entityType string, entityID any, before, after any, ip string) {
        if s.audit == nil {
                return
        }
        e := audit.Entry{
                ActorID: actorID, ActorUsername: username,
                Module: module, Action: action, EntityType: entityType,
                Before: before, After: after, IP: ip,
        }
        if entityID != nil {
                switch v := entityID.(type) {
                case int64:
                        e.EntityID = appkit.Int64ToStr(v)
                case string:
                        e.EntityID = v
                }
        }
        if err := s.audit.Log(ctx, e); err != nil {
                slog.Error("audit write failed", "module", module, "action", action, "err", err)
        }
}

func actorID(u *auth.AuthUser) *int64 {
        if u == nil {
                return nil
        }
        return &u.ID
}
