// Package api — لایه Interface ماژول گزارش‌ها (قرارداد v1).
package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"fanavaryar/internal/modules/reports/application"
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
	r.Route("/reports", func(r chi.Router) {
		r.With(auth.RequirePermission("reports.view")).Get("/", h.list)
		r.With(auth.RequirePermission("reports.create")).Post("/", h.create)
		r.With(auth.RequirePermission("reports.view")).Get("/{id}", h.detail)
		r.With(auth.RequirePermission("reports.approve")).Post("/{id}/approve", h.approve)
		r.With(auth.RequirePermission("reports.approve")).Post("/{id}/reject", h.reject)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.List(r.Context(), application.ListFilter{
		Status: r.URL.Query().Get("status"),
		Query:  r.URL.Query().Get("q"),
		Page:   httpx.ReadPage(r),
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

type createInput struct {
	Title                 string         `json:"title"`
	FormTemplateVersionID int64          `json:"form_template_version_id"`
	FormData              map[string]any `json:"form_data"`
	AssetIDs              []int64        `json:"asset_ids"`
	ReportDate            string         `json:"report_date"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in createInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	id, err := h.svc.Create(r.Context(), auth.FromContext(r.Context()), application.CreateInput{
		Title:                 in.Title,
		FormTemplateVersionID: in.FormTemplateVersionID,
		FormData:              in.FormData,
		AssetIDs:              in.AssetIDs,
		ReportDate:            in.ReportDate,
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]int64{"id": id})
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

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.Approve(r.Context(), auth.FromContext(r.Context()), id); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type rejectInput struct {
	Reason string `json:"reason"`
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var in rejectInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.Reject(r.Context(), auth.FromContext(r.Context()), id, in.Reason); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, httpx.BadRequest("شناسه گزارش نامعتبر است")
	}
	return id, nil
}
