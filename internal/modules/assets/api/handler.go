// Package api — لایه Interface ماژول تجهیزات (قرارداد v1).
package api

import (
        "net/http"
        "strconv"

        "github.com/go-chi/chi/v5"

        "fanavaryar/internal/modules/assets/application"
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
        r.Route("/assets", func(r chi.Router) {
                r.With(auth.RequirePermission("assets.view")).Get("/", h.list)
                r.With(auth.RequirePermission("assets.create")).Post("/", h.create)
                r.With(auth.RequirePermission("assets.view")).Get("/{id}", h.detail)
                r.With(auth.RequirePermission("assets.edit")).Patch("/{id}", h.update)
                r.With(auth.RequirePermission("assets.edit")).Post("/{id}/move", h.move)
        })
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
        f := application.ListFilter{
                Status:   r.URL.Query().Get("status"),
                Category: r.URL.Query().Get("category"),
                Query:    r.URL.Query().Get("q"),
                Page:     httpx.ReadPage(r),
        }
        if raw := r.URL.Query().Get("location_id"); raw != "" {
                if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
                        f.LocationID = &v
                }
        }
        res, err := h.svc.List(r.Context(), auth.FromContext(r.Context()), f)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, res)
}

type assetInput struct {
        AssetTag          string          `json:"asset_tag"`
        Name              string          `json:"name"`
        Category          string          `json:"category"`
        Status            string          `json:"status"`
        SerialNumber      *string         `json:"serial_number"`
        Manufacturer      *string         `json:"manufacturer"`
        Model             *string         `json:"model"`
        ResponsiblePerson *string         `json:"responsible_person"`
        CurrentLocationID *int64          `json:"current_location_id"`
        Notes             *string         `json:"notes"`
        SpecFormVersionID *int64          `json:"spec_form_version_id"` // اختیاری: فرم مشخصات سخت‌افزاری
        SpecData          map[string]any  `json:"spec_data"`
        ClearSpecs        bool            `json:"clear_specs"` // حذف مشخصات فرمی
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
        var in assetInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        a, err := h.svc.Create(r.Context(), auth.FromContext(r.Context()), application.CreateInput{
                AssetTag:          in.AssetTag,
                Name:              in.Name,
                Category:          in.Category,
                Status:            in.Status,
                SerialNumber:      in.SerialNumber,
                Manufacturer:      in.Manufacturer,
                Model:             in.Model,
                ResponsiblePerson: in.ResponsiblePerson,
                CurrentLocationID: in.CurrentLocationID,
                Notes:             in.Notes,
                SpecFormVersionID: in.SpecFormVersionID,
                SpecData:          in.SpecData,
        })
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusCreated, a)
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r)
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

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        var in assetInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        a, err := h.svc.Update(r.Context(), auth.FromContext(r.Context()), id, application.UpdateInput{
                AssetTag:          nonEmptyPtr(in.AssetTag),
                Name:              nonEmptyPtr(in.Name),
                Category:          nonEmptyPtr(in.Category),
                Status:            nonEmptyPtr(in.Status),
                SerialNumber:      in.SerialNumber,
                Manufacturer:      in.Manufacturer,
                Model:             in.Model,
                ResponsiblePerson: in.ResponsiblePerson,
                CurrentLocationID: in.CurrentLocationID,
                Notes:             in.Notes,
                SpecFormVersionID: in.SpecFormVersionID,
                SpecData:          in.SpecData,
                ClearSpecs:        in.ClearSpecs,
        })
        if err != nil {
                httpx.Error(w, err)
                return
        }
        httpx.JSON(w, http.StatusOK, a)
}

type moveInput struct {
        ToLocationID int64   `json:"to_location_id"`
        Reason       *string `json:"reason"`
}

func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
        id, err := pathID(r)
        if err != nil {
                httpx.Error(w, err)
                return
        }
        var in moveInput
        if err := httpx.Decode(w, r, &in); err != nil {
                httpx.Error(w, err)
                return
        }
        if in.ToLocationID <= 0 {
                httpx.Error(w, httpx.BadRequest("مکان مقصد را انتخاب کنید"))
                return
        }
        if err := h.svc.Move(r.Context(), auth.FromContext(r.Context()), id, in.ToLocationID, in.Reason); err != nil {
                httpx.Error(w, err)
                return
        }
        w.WriteHeader(http.StatusNoContent)
}

// nonEmptyPtr فیلدهای اجباری payload؛ رشته خالی یعنی «ارسال نشده» و بدون تغییر می‌ماند.
func nonEmptyPtr(s string) *string {
        if s == "" {
                return nil
        }
        return &s
}

func pathID(r *http.Request) (int64, error) {
        raw := chi.URLParam(r, "id")
        id, err := strconv.ParseInt(raw, 10, 64)
        if err != nil || id <= 0 {
                return 0, httpx.BadRequest("شناسه تجهیز نامعتبر است")
        }
        return id, nil
}
