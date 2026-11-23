// Package infrastructure — پیاده‌سازی PostgreSQL ماژول گزارش‌ها (قرارداد v1).
package infrastructure

import (
        "context"
        "encoding/json"
        "errors"
        "strconv"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgxpool"

        "fanavaryar/internal/modules/forms/domain"
        "fanavaryar/internal/modules/reports/application"
        reportsdomain "fanavaryar/internal/modules/reports/domain"
)

type ReportRepository struct {
        pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
        return &ReportRepository{pool: pool}
}

const listCols = `
        rp.id, rp.code, rp.title, to_char(rp.report_date, 'YYYY-MM-DD'), rp.status,
        cu.full_name, au.full_name, rp.created_at
`

func scanListItem(row pgx.Row) (application.ReportListItem, error) {
        var i application.ReportListItem
        var status string
        err := row.Scan(&i.ID, &i.Code, &i.Title, &i.ReportDate, &status,
                &i.CreatedByName, &i.ApprovedByName, &i.CreatedAt)
        i.Status = status
        return i, err
}

func (r *ReportRepository) List(ctx context.Context, f application.ListFilter) ([]application.ReportListItem, int, error) {
        args := []any{}
        where := `
        WHERE ($1 = '' OR rp.code ILIKE '%' || $1 || '%' OR rp.title ILIKE '%' || $1 || '%')
          AND ($2 = '' OR rp.status = $2)`
        args = append(args, f.Query, f.Status)

        var total int
        if err := r.pool.QueryRow(ctx, `
                SELECT count(*) FROM reports rp
                JOIN users cu ON cu.id = rp.author_id
                LEFT JOIN users au ON au.id = rp.approved_by
                `+where, args...).Scan(&total); err != nil {
                return nil, 0, err
        }

        args = append(args, f.Page.PerPage, f.Page.Offset())
        rows, err := r.pool.Query(ctx, `
                SELECT `+listCols+`
                FROM reports rp
                JOIN users cu ON cu.id = rp.author_id
                LEFT JOIN users au ON au.id = rp.approved_by
                `+where+`
                ORDER BY rp.created_at DESC, rp.id DESC
                LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
        if err != nil {
                return nil, 0, err
        }
        defer rows.Close()

        var out []application.ReportListItem
        for rows.Next() {
                i, err := scanListItem(rows)
                if err != nil {
                        return nil, 0, err
                }
                out = append(out, i)
        }
        return out, total, rows.Err()
}

// Create ثبت گزارش: کد یکتا + اتصال به نسخه فرم + اتصال تجهیزها در یک تراکنش.
// فرم‌های بدون نیاز به تایید ($6=true) مستقیم با وضعیت approved و زمان تایید ثبت می‌شوند.
func (r *ReportRepository) Create(ctx context.Context, in application.CreateInput, authorID int64) (int64, error) {
        autoApprove := !in.RequiresApproval
        var id int64
        err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
                if err := tx.QueryRow(ctx, `
                        INSERT INTO reports (title, status, author_id, form_template_version_id, form_data, report_date, approved_at)
                        VALUES ($1,
                                CASE WHEN $6 THEN 'approved' ELSE 'pending' END,
                                $2, $3, $4, $5,
                                CASE WHEN $6 THEN now() ELSE NULL END)
                        RETURNING id, created_at`,
                        in.Title, authorID, in.FormTemplateVersionID, in.FormData, in.ReportDate, autoApprove,
                ).Scan(&id, new(any)); err != nil {
                        return err
                }
                // کد گزارش یکتا بر اساس تاریخ ثبت و شناسه
                if _, err := tx.Exec(ctx, `
                        UPDATE reports
                        SET code = 'R-' || to_char(created_at, 'YYYYMMDD') || '-' || lpad(id::text, 4, '0')
                        WHERE id = $1`, id); err != nil {
                        return err
                }
                for _, aid := range uniqueIDs(in.AssetIDs) {
                        if _, err := tx.Exec(ctx, `
                                INSERT INTO report_assets (report_id, asset_id) VALUES ($1, $2)
                                ON CONFLICT DO NOTHING`, id, aid); err != nil {
                                return err
                        }
                }
                return nil
        })
        if err != nil {
                return 0, err
        }
        return id, nil
}

func uniqueIDs(ids []int64) []int64 {
        seen := map[int64]bool{}
        out := []int64{}
        for _, id := range ids {
                if !seen[id] {
                        seen[id] = true
                        out = append(out, id)
                }
        }
        return out
}

// CountExisting قرارداد AssetChecker برای اعتبارسنجی تجهیزهای انتخابی.
func (r *ReportRepository) CountExisting(ctx context.Context, ids []int64) (int, error) {
        var n int
        err := r.pool.QueryRow(ctx,
                `SELECT count(*) FROM assets WHERE id = ANY($1)`, ids).Scan(&n)
        return n, err
}

func (r *ReportRepository) GetDetail(ctx context.Context, id int64) (application.ReportDetail, error) {
        var d application.ReportDetail
        var status, templateName string
        var fieldsRaw []byte
        var formData []byte
        var versionNumber int

        err := r.pool.QueryRow(ctx, `
                SELECT rp.id, rp.code, rp.title, to_char(rp.report_date, 'YYYY-MM-DD'), rp.status,
                       rp.form_template_version_id, fv.version_number, ft.name,
                       fv.fields, COALESCE(rp.form_data, '{}'::jsonb),
                       cu.full_name, au.full_name, rp.approved_at, rp.reject_reason, rp.created_at
                FROM reports rp
                JOIN users cu ON cu.id = rp.author_id
                LEFT JOIN users au ON au.id = rp.approved_by
                JOIN form_versions fv ON fv.id = rp.form_template_version_id
                JOIN form_templates ft ON ft.id = fv.template_id
                WHERE rp.id = $1`, id).
                Scan(&d.ID, &d.Code, &d.Title, &d.ReportDate, &status,
                        &d.FormTemplateVersionID, &versionNumber, &templateName,
                        &fieldsRaw, &formData,
                        &d.CreatedByName, &d.ApprovedByName, &d.ApprovedAt, &d.RejectReason, &d.CreatedAt)
        if errors.Is(err, pgx.ErrNoRows) {
                return application.ReportDetail{}, domain.ErrNotFound
        }
        if err != nil {
                return application.ReportDetail{}, err
        }
        d.Status = status
        d.VersionNumber = versionNumber
        d.TemplateName = templateName

        fields, err := domain.ParseFields(fieldsRaw)
        if err != nil {
                return application.ReportDetail{}, err
        }
        if fields == nil {
                fields = []domain.FieldDef{}
        }
        d.Fields = fields

        if err := jsonUnmap(formData, &d.FormData); err != nil {
                return application.ReportDetail{}, err
        }

        rows, err := r.pool.Query(ctx, `
                SELECT a.id, a.asset_tag, a.name
                FROM report_assets ra
                JOIN assets a ON a.id = ra.asset_id
                WHERE ra.report_id = $1
                ORDER BY a.asset_tag`, id)
        if err != nil {
                return application.ReportDetail{}, err
        }
        defer rows.Close()

        d.Assets = []application.ReportAsset{}
        for rows.Next() {
                var a application.ReportAsset
                if err := rows.Scan(&a.ID, &a.AssetTag, &a.Name); err != nil {
                        return application.ReportDetail{}, err
                }
                d.Assets = append(d.Assets, a)
        }
        return d, rows.Err()
}

// Approve تایید گزارش؛ وضعیت و عدم تاییدِ گزارشِ خود، در خود کوئری/تراکنش کنترل می‌شود.
func (r *ReportRepository) Approve(ctx context.Context, id int64, approverID int64) error {
        return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
                var authorID int64
                var status string
                err := tx.QueryRow(ctx, `
                        SELECT author_id, status FROM reports WHERE id = $1 FOR UPDATE`, id).
                        Scan(&authorID, &status)
                if errors.Is(err, pgx.ErrNoRows) {
                        return reportsdomain.ErrNotFound
                }
                if err != nil {
                        return err
                }
                if status != string(reportsdomain.StatusPending) {
                        return reportsdomain.ErrNotPending
                }
                if authorID == approverID {
                        return reportsdomain.ErrSelfApproval
                }
                _, err = tx.Exec(ctx, `
                        UPDATE reports SET status = 'approved', approved_by = $2, approved_at = now(), updated_at = now()
                        WHERE id = $1`, id, approverID)
                return err
        })
}

func (r *ReportRepository) Reject(ctx context.Context, id int64, approverID int64, reason string) error {
        return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
                var authorID int64
                var status string
                err := tx.QueryRow(ctx, `
                        SELECT author_id, status FROM reports WHERE id = $1 FOR UPDATE`, id).
                        Scan(&authorID, &status)
                if errors.Is(err, pgx.ErrNoRows) {
                        return reportsdomain.ErrNotFound
                }
                if err != nil {
                        return err
                }
                if status != string(reportsdomain.StatusPending) {
                        return reportsdomain.ErrNotPending
                }
                if authorID == approverID {
                        return reportsdomain.ErrSelfApproval
                }
                _, err = tx.Exec(ctx, `
                        UPDATE reports
                        SET status = 'rejected', approved_by = $2, approved_at = now(), reject_reason = $3, updated_at = now()
                        WHERE id = $1`, id, approverID, reason)
                return err
        })
}

// jsonUnmap رمزگشایی JSONB به map با مقدار پیش‌فرض.
func jsonUnmap(raw []byte, dst *map[string]any) error {
        m := map[string]any{}
        if len(raw) > 0 {
                if err := json.Unmarshal(raw, &m); err != nil {
                        return err
                }
        }
        *dst = m
        return nil
}
