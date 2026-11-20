// Package api — لایه Interface ماژول کاربران: ورود، نشست، حساب‌ها، مجوزها و الگوها.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"fanavaryar/internal/modules/users/application"
	"fanavaryar/internal/platform/auth"
	"fanavaryar/internal/platform/httpx"
)

type Handler struct {
	svc          *application.Service
	sessions     *auth.SessionService
	cookieSecure bool
	sessionTTL   time.Duration
}

func NewHandler(svc *application.Service, sessions *auth.SessionService,
	cookieSecure bool, sessionTTL time.Duration) *Handler {
	return &Handler{svc: svc, sessions: sessions,
		cookieSecure: cookieSecure, sessionTTL: sessionTTL}
}

// MountAuth مسیرهای عمومی ورود؛ قبل از میدلور احراز هویت سوار می‌شود.
func (h *Handler) MountAuth(r chi.Router) {
	r.Post("/auth/login", h.login)
}

// MountSession مسیرهای نشست که خودِ کاربر احرازشده استفاده می‌کند.
func (h *Handler) MountSession(r chi.Router) {
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/change-password", h.changePassword)
	r.Get("/auth/me", h.me)
}

// MountAdmin مسیرهای مدیریتی کاربران، کاتالوگ مجوز و الگوها.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.With(auth.RequirePermission("users.view")).Get("/", h.list)
		r.With(auth.RequirePermission("users.create")).Post("/", h.create)
		r.With(auth.RequirePermission("users.view")).Get("/{id}", h.get)
		r.With(auth.RequirePermission("users.edit")).Patch("/{id}", h.update)
		r.With(auth.RequirePermission("users.edit")).Post("/{id}/reset-password", h.resetPassword)
		r.With(auth.RequirePermission("users.edit_permissions")).Put("/{id}/permissions", h.setPermissions)
		r.With(auth.RequirePermission("users.edit_permissions")).Put("/{id}/scope", h.setScope)
	})

	r.Route("/permissions", func(r chi.Router) {
		r.With(auth.RequirePermission("users.view")).Get("/", h.catalog)
	})

	r.Route("/permission-templates", func(r chi.Router) {
		r.With(auth.RequirePermission("users.view")).Get("/", h.templates)
		r.With(auth.RequirePermission("users.edit_permissions")).Post("/", h.createTemplate)
		r.With(auth.RequirePermission("users.edit_permissions")).Delete("/{id}", h.deleteTemplate)
	})
}

// ---------- auth ----------

type loginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	u, token, err := h.svc.Login(r.Context(), in.Username, in.Password,
		httpx.ClientIP(r), h.sessionTTL)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	h.setSessionCookie(w, token, h.sessionTTL)
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token := ""
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil {
		token = cookie.Value
	}
	h.svc.Logout(r.Context(), auth.FromContext(r.Context()), token, httpx.ClientIP(r))
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u := auth.FromContext(r.Context())
	if u == nil {
		httpx.Error(w, httpx.Unauthorized("ابتدا وارد سامانه شوید"))
		return
	}
	resp := *u
	resp.Perms = u.SortedPermissions()
	httpx.JSON(w, http.StatusOK, &resp)
}

type changePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var in changePasswordInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	user := auth.FromContext(r.Context())
	token, err := h.svc.ChangePassword(r.Context(), user, in.OldPassword, in.NewPassword,
		httpx.ClientIP(r), h.sessionTTL)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	h.setSessionCookie(w, token, h.sessionTTL)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// ---------- users ----------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.List(r.Context(), r.URL.Query().Get("q"), httpx.ReadPage(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

type userInput struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	JobTitle string `json:"job_title"`
	Password string `json:"password"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in userInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	u, err := h.svc.Create(r.Context(), auth.FromContext(r.Context()), application.CreateInput{
		Username: in.Username,
		FullName: in.FullName,
		JobTitle: in.JobTitle,
		Password: in.Password,
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, u)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "کاربر")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	d, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

type profileInput struct {
	FullName string  `json:"full_name"`
	JobTitle string  `json:"job_title"`
	IsActive *bool   `json:"is_active"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "کاربر")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var in profileInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	u, err := h.svc.UpdateProfile(r.Context(), auth.FromContext(r.Context()), id, application.ProfileInput{
		FullName: in.FullName,
		JobTitle: in.JobTitle,
		IsActive: in.IsActive,
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

type resetPasswordInput struct {
	NewPassword string `json:"new_password"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "کاربر")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var in resetPasswordInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), auth.FromContext(r.Context()), id,
		in.NewPassword, httpx.ClientIP(r)); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type permissionsInput struct {
	PermissionCodes []string `json:"permission_codes"`
}

func (h *Handler) setPermissions(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "کاربر")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var in permissionsInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.SetPermissions(r.Context(), auth.FromContext(r.Context()), id, in.PermissionCodes); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type scopeInput struct {
	ScopeType  string `json:"scope_type"`
	LocationID *int64 `json:"location_id"`
}

func (h *Handler) setScope(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "کاربر")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var in scopeInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.SetScope(r.Context(), auth.FromContext(r.Context()), id, in.ScopeType, in.LocationID); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- کاتالوگ و الگوها ----------

func (h *Handler) catalog(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Catalog(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) templates(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Templates(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

type templateInput struct {
	Name            string   `json:"name"`
	PermissionCodes []string `json:"permission_codes"`
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var in templateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	id, err := h.svc.CreateTemplate(r.Context(), auth.FromContext(r.Context()), in.Name, in.PermissionCodes)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "الگو")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.DeleteTemplate(r.Context(), auth.FromContext(r.Context()), id); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request, entity string) (int64, error) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, httpx.BadRequest("شناسه " + entity + " نامعتبر است")
	}
	return id, nil
}
