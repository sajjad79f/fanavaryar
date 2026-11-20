// دفتر رویدادها — خواندن رویدادهای Audit با صفحه‌بندی و جست‌وجو (قرارداد v1).
package bootstrap

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"fanavaryar/internal/platform/audit"
	"fanavaryar/internal/platform/auth"
	"fanavaryar/internal/platform/httpx"
)

func mountAuditHistory(r chi.Router, svc *audit.Service) {
	r.With(auth.RequirePermission("audit.view")).Get("/audit", func(w http.ResponseWriter, req *http.Request) {
		p := httpx.ReadPage(req)
		res, err := svc.Query(req.Context(), req.URL.Query().Get("q"), p)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, res)
	})
}
