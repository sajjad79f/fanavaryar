// Package api — لایه Interface ماژول مکان‌ها (قرارداد v1).
package api

import (
        "net/http"
        "strconv"

        "github.com/go-chi/chi/v5"

        locationsApp "fanavaryar/internal/modules/locations/application"
        "fanavaryar/internal/platform/auth"
        "fanavaryar/internal/platform/httpx"
)

type Handler struct {
        svc *locationsApp.Service
}

func NewHandler(svc *locationsApp.Service) *Handler {
        return &Handler{svc: svc}
}

// Mount مسیرهای ماژول با مجوزهای لازم.
func (h *Handler) Mount(r chi.Router) {
        r.Route("/locations", func(r chi.Router) {
                // فهرست مکان‌ها برای انتخاب‌گر تجهیزات هم لازم است؛ یکی از دو مجوز کافی است
                r.Get("/", h.list)
                r.With(auth.RequirePermission("locations.create")).Post("/", h.create)
                r.With(auth.RequirePermission("locations.edit")).Patch("/{id}", h.update)
                r.With(auth.RequirePermission("locations.delete")).Delete("/{id}", h.delete)
        })
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
        u := auth.FromContext(r.Context())
        if u == nil || !u.HasAny("locations.view", "assets.view") {
                httpx.Error(w, httpx.Forbidden("شما مجوز لازم برای این عملیات را ندارید"))
                return
        }
        rows, err := h.svc.List(r.Context())
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, rows)
}

type locationInput struct {
        Name     string  `json:"name"`
        Kind     string  `json:"kind"`
        ParentID *int64  `json:"parent_id"`
        Code     *string `json:"code"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
        var in locationInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        row, err := h.svc.Create(r.Context(), auth.FromContext(r.Context()), locationsApp.CreateInput{
                Name:     in.Name,
                Kind:     in.Kind,
                ParentID: in.ParentID,
                Code:     in.Code,
        })
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusCreated, row)
}

type renameInput struct {
        Name string  `json:"name"`
        Code *string `json:"code"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        var in renameInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        row, err := h.svc.Rename(r.Context(), auth.FromContext(r.Context()), id, in.Name, in.Code)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, row)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        if err := h.svc.Delete(r.Context(), auth.FromContext(r.Context()), id); err != nil {
                httpx.Error(w, err)
                return
        }
        w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int64, error) {
        raw := chi.URLParam(r, "id")
        id, err := strconv.ParseInt(raw, 10, 64)
        if err != nil || id <= 0 {
                return 0, httpx.BadRequest("شناسه مکان نامعتبر است")
        }
        return id, nil
}
