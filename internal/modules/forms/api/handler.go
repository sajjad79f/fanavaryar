// Package api — لایه Interface ماژول فرم‌ها (قرارداد v1).
package api

import (
        "net/http"
        "strconv"

        "github.com/go-chi/chi/v5"

        "fanavaryar/internal/modules/forms/application"
        "fanavaryar/internal/modules/forms/domain"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

type Handler struct {
        svc *application.Service
}

func NewHandler(svc *application.Service) *Handler {
        return &Handler{svc: svc}
}

func (h *Handler) Mount(r chi.Router) {
        r.Route("/forms", func(r chi.Router) {
                // خواندن فرم‌ها برای ثبت گزارش هم لازم است؛ پس یکی از دو مجوز کافی است
                r.Get("/", h.readGuard(h.list))
                r.With(auth.RequirePermission("forms.template_create")).Post("/", h.create)
                r.Get("/versions/{vid}", h.readGuard(h.version))
                r.Get("/{id}", h.readGuard(h.detail))
                r.Patch("/{id}", h.update)
                r.With(auth.RequirePermission("forms.template_edit")).Post("/{id}/versions", h.newVersion)
        })
}

// readGuard خواندن فرم را به داشتن forms.view یا reports.create گره می‌زند.
func (h *Handler) readGuard(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                u := auth.FromContext(r.Context())
                if u == nil {
                        httpx.Error(w, httpx.Unauthorized("ابتدا وارد سامانه شوید"))
                        return
                }
                if !u.HasAny("forms.view", "reports.create") {
                        httpx.Error(w, httpx.Forbidden("شما مجوز لازم برای این عملیات را ندارید"))
                        return
                }
                next(w, r)
        }
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
        items, err := h.svc.List(r.Context(), r.URL.Query().Get("kind"))
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, items)
}

type createInput struct {
        Code             string            `json:"code"`
        Name             string            `json:"name"`
        Description      *string           `json:"description"`
        Kind             string            `json:"kind"` // خالی = report
        Fields           []domain.FieldDef `json:"fields"`
        RequiresApproval *bool             `json:"requires_approval"` // nil = true
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
        var in createInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        id, err := h.svc.Create(r.Context(), auth.FromContext(r.Context()), application.CreateInput{
                Code:             in.Code,
                Name:             in.Name,
                Description:      in.Description,
                Kind:             in.Kind,
                Fields:           in.Fields,
                RequiresApproval: in.RequiresApproval,
        })
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r, "id", "فرم")
        if err != nil {
                httpx.Error(w, err)
                return
        }
        d, err := h.svc.Detail(r.Context(), id)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, d)
}

type updateInput struct {
        IsActive         *bool `json:"is_active"`
        RequiresApproval *bool `json:"requires_approval"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
        u := auth.FromContext(r.Context())
        if u == nil {
                httpx.Error(w, httpx.Unauthorized("ابتدا وارد سامانه شوید"))
                return
        }
        id, err := pathID(r, "id", "فرم")
        if err != nil {
                httpx.Error(w, err)
                return
        }
        var in updateInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        if in.IsActive == nil && in.RequiresApproval == nil {
                httpx.Error(w, httpx.BadRequest("مقدار is_active یا requires_approval الزامی است"))
                return
        }
        if in.IsActive != nil {
                if !u.HasAny("forms.publish", "forms.disable") {
                        httpx.Error(w, httpx.Forbidden("شما مجوز لازم برای این عملیات را ندارید"))
                        return
                }
                if err := h.svc.SetActive(r.Context(), u, id, *in.IsActive); err != nil {
                        httpx.Error(w, err)
                        return
                }
        }
        if in.RequiresApproval != nil {
                if err := h.svc.SetRequiresApproval(r.Context(), u, id, *in.RequiresApproval); err != nil {
                        httpx.Error(w, err)
                        return
                }
        }
        w.WriteHeader(http.StatusNoContent)
}

type versionInput struct {
        Fields []domain.FieldDef `json:"fields"`
}

func (h *Handler) newVersion(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r, "id", "فرم")
        if err != nil {
                httpx.Error(w, err)
                return
        }
        var in versionInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        vid, ver, err := h.svc.CreateVersion(r.Context(), auth.FromContext(r.Context()), id, in.Fields)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusCreated, map[string]any{"id": vid, "version_number": ver})
}

func (h *Handler) version(w http.ResponseWriter, r *http.Request) {
        raw := chi.URLParam(r, "vid")
        vid, err := strconv.ParseInt(raw, 10, 64)
        if err != nil || vid <= 0 {
                httpx.Error(w, httpx.BadRequest("شناسه نسخه نامعتبر است"))
                return
        }
        v, err := h.svc.VersionByID(r.Context(), vid)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, v)
}

func pathID(r *http.Request, param, entity string) (int64, error) {
        raw := chi.URLParam(r, param)
        id, err := strconv.ParseInt(raw, 10, 64)
        if err != nil || id <= 0 {
                return 0, httpx.BadRequest("شناسه " + entity + " نامعتبر است")
        }
        return id, nil
}
