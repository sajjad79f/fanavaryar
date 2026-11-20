// Package infrastructure — پیاده‌سازی PostgreSQL ماژول تجهیزات (قرارداد v1).
// فیلتر محدوده اطلاعاتی با Recursive CTE روی زیرشاخه‌های مکان انجام می‌شود.
package infrastructure

import (
        "context"
        "errors"
        "strconv"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgconn"
        "github.com/jackc/pgx/v5/pgxpool"

        "fanavaryar/internal/modules/assets/application"
        "fanavaryar/internal/modules/assets/domain"
)

type AssetRepository struct {
        pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
        return &AssetRepository{pool: pool}
}

func mapUnique(err error) error {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
                return domain.ErrTagTaken
        }
        return err
}

const assetCols = `
        a.id, a.asset_tag, a.name, a.category, a.status,
        a.serial_number, a.manufacturer, a.model, a.responsible_person,
        a.current_location_id, loc.name, a.notes, a.created_at, a.updated_at
`

const assetFrom = ` FROM assets a LEFT JOIN locations loc ON loc.id = a.current_location_id `

func scanAsset(row pgx.Row) (application.Asset, error) {
        var a application.Asset
        var status string
        err := row.Scan(&a.ID, &a.AssetTag, &a.Name, &a.Category, &status,
                &a.SerialNumber, &a.Manufacturer, &a.Model, &a.ResponsiblePerson,
                &a.CurrentLocationID, &a.LocationName, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
        a.Status = status
        return a, err
}

func (r *AssetRepository) Create(ctx context.Context, in application.CreateInput, actorID *int64) (application.Asset, error) {
        status := in.Status
        if status == "" {
                status = string(domain.StatusActive)
        }
        var id int64
        if err := r.pool.QueryRow(ctx, `
                INSERT INTO assets (asset_tag, name, category, status, serial_number,
                                    manufacturer, model, responsible_person, current_location_id, notes)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                RETURNING id`,
                in.AssetTag, in.Name, in.Category, status,
                in.SerialNumber, in.Manufacturer, in.Model, in.ResponsiblePerson,
                in.CurrentLocationID, in.Notes).Scan(&id); err != nil {
                return application.Asset{}, mapUnique(err)
        }
        _, _ = r.pool.Exec(ctx, `
                INSERT INTO asset_events (asset_id, event_type, actor_id)
                VALUES ($1, 'created', $2)`, id, actorID)
        return r.GetByID(ctx, id)
}

// Update ویرایش جزئی؛ فیلدهای nil دست‌نخورده می‌مانند و رشته خالی NULL می‌شود.
func (r *AssetRepository) Update(ctx context.Context, id int64, in application.UpdateInput, actorID *int64) (application.Asset, error) {
        sets := []string{}
        args := []any{}
        add := func(col string, v any) {
                args = append(args, v)
                sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
        }
        addStr := func(col string, p *string) {
                if p == nil {
                        return
                }
                if *p == "" {
                        args = append(args, nil)
                } else {
                        args = append(args, *p)
                }
                sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
        }
        addStr("asset_tag", in.AssetTag)
        addStr("name", in.Name)
        addStr("category", in.Category)
        if in.Status != nil {
                add("status", *in.Status)
        }
        addStr("serial_number", in.SerialNumber)
        addStr("manufacturer", in.Manufacturer)
        addStr("model", in.Model)
        addStr("responsible_person", in.ResponsiblePerson)
        if in.CurrentLocationID != nil {
                add("current_location_id", *in.CurrentLocationID)
        }
        addStr("notes", in.Notes)
        sets = append(sets, "updated_at = now()")

        tag, err := r.pool.Exec(ctx,
                `UPDATE assets SET `+joinComma(sets)+` WHERE id = $`+strconv.Itoa(len(args)+1),
                append(args, id)...)
        if err != nil {
                return application.Asset{}, mapUnique(err)
        }
        if tag.RowsAffected() == 0 {
                return application.Asset{}, domain.ErrNotFound
        }
        _, _ = r.pool.Exec(ctx, `
                INSERT INTO asset_events (asset_id, event_type, actor_id)
                VALUES ($1, 'updated', $2)`, id, actorID)
        return r.GetByID(ctx, id)
}

func joinComma(parts []string) string {
        out := ""
        for i, p := range parts {
                if i > 0 {
                        out += ", "
                }
                out += p
        }
        return out
}

func (r *AssetRepository) GetByID(ctx context.Context, id int64) (application.Asset, error) {
        a, err := scanAsset(r.pool.QueryRow(ctx,
                `SELECT `+assetCols+assetFrom+` WHERE a.id = $1`, id))
        if errors.Is(err, pgx.ErrNoRows) {
                return application.Asset{}, domain.ErrNotFound
        }
        return a, err
}

// scopeClause برای building/floor/room: زیرشاخه‌های مکان مرجع + تجهیزاتِ بدون مکان.
func scopeClause(f application.ListFilter, args *[]any) string {
        if f.Scope.Type == "all" {
                return ""
        }
        *args = append(*args, *f.Scope.RefID)
        n := len(*args)
        return `
        AND (a.current_location_id IN (
                WITH RECURSIVE subtree AS (
                        SELECT id FROM locations WHERE id = $` + strconv.Itoa(n) + `
                        UNION ALL
                        SELECT l.id FROM locations l JOIN subtree s ON l.parent_id = s.id
                ) SELECT id FROM subtree
        ) OR a.current_location_id IS NULL)`
}

func (r *AssetRepository) List(ctx context.Context, f application.ListFilter) ([]application.Asset, int, error) {
        args := []any{}
        where := " WHERE 1 = 1" + scopeClause(f, &args)

        if f.Query != "" {
                args = append(args, "%"+f.Query+"%")
                where += ` AND (a.asset_tag ILIKE $` + strconv.Itoa(len(args)) +
                        ` OR a.name ILIKE $` + strconv.Itoa(len(args)) +
                        ` OR COALESCE(a.serial_number, '') ILIKE $` + strconv.Itoa(len(args)) +
                        ` OR COALESCE(a.responsible_person, '') ILIKE $` + strconv.Itoa(len(args)) + `)`
        }
        if f.Status != "" {
                args = append(args, f.Status)
                where += ` AND a.status = $` + strconv.Itoa(len(args))
        }
        if f.Category != "" {
                args = append(args, f.Category)
                where += ` AND a.category = $` + strconv.Itoa(len(args))
        }
        if f.LocationID != nil {
                args = append(args, *f.LocationID)
                where += ` AND a.current_location_id = $` + strconv.Itoa(len(args))
        }

        var total int
        if err := r.pool.QueryRow(ctx,
                `SELECT count(*)`+assetFrom+where, args...).Scan(&total); err != nil {
                return nil, 0, err
        }

        args = append(args, f.Page.PerPage, f.Page.Offset())
        rows, err := r.pool.Query(ctx,
                `SELECT `+assetCols+assetFrom+where+
                        ` ORDER BY a.asset_tag LIMIT $`+strconv.Itoa(len(args)-1)+
                        ` OFFSET $`+strconv.Itoa(len(args)), args...)
        if err != nil {
                return nil, 0, err
        }
        defer rows.Close()

        var out []application.Asset
        for rows.Next() {
                a, err := scanAsset(rows)
                if err != nil {
                        return nil, 0, err
                }
                out = append(out, a)
        }
        return out, total, rows.Err()
}

func (r *AssetRepository) InsertMove(ctx context.Context, m application.MoveInput, actorID *int64) error {
        // جابه‌جایی در یک تراکنش: ثبت تاریخچه + تغییر مکان تجهیز
        return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
                if _, err := tx.Exec(ctx, `
                        INSERT INTO asset_moves (asset_id, from_location_id, to_location_id, reason, moved_by)
                        VALUES ($1, (SELECT current_location_id FROM assets WHERE id = $1), $2, $3, $4)`,
                        m.AssetID, m.ToLocationID, m.Reason, actorID,
                ); err != nil {
                        return err
                }
                if _, err := tx.Exec(ctx, `
                        UPDATE assets SET current_location_id = $2, updated_at = now() WHERE id = $1`,
                        m.AssetID, m.ToLocationID,
                ); err != nil {
                        return err
                }
                _, err := tx.Exec(ctx, `
                        INSERT INTO asset_events (asset_id, event_type, actor_id)
                        VALUES ($1, 'moved', $2)`, m.AssetID, actorID)
                return err
        })
}

func (r *AssetRepository) ListMoves(ctx context.Context, assetID int64) ([]application.Movement, error) {
        rows, err := r.pool.Query(ctx, `
                SELECT m.id, m.from_location_id, lf.name, m.to_location_id, lt.name,
                       m.reason, u.full_name, m.created_at
                FROM asset_moves m
                LEFT JOIN locations lf ON lf.id = m.from_location_id
                LEFT JOIN locations lt ON lt.id = m.to_location_id
                LEFT JOIN users u ON u.id = m.moved_by
                WHERE m.asset_id = $1
                ORDER BY m.created_at DESC, m.id DESC`, assetID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var out []application.Movement
        for rows.Next() {
                var m application.Movement
                if err := rows.Scan(&m.ID, &m.FromLocationID, &m.FromLocationName,
                        &m.ToLocationID, &m.ToLocationName, &m.Reason, &m.MovedByName, &m.CreatedAt); err != nil {
                        return nil, err
                }
                out = append(out, m)
        }
        return out, rows.Err()
}

// ListReports گزارش‌های متصل به تجهیز از طریق report_assets.
func (r *AssetRepository) ListReports(ctx context.Context, assetID int64) ([]application.RelatedReport, error) {
        rows, err := r.pool.Query(ctx, `
                SELECT rp.id, rp.code, rp.title,
                       to_char(rp.report_date, 'YYYY-MM-DD'), rp.status
                FROM report_assets ra
                JOIN reports rp ON rp.id = ra.report_id
                WHERE ra.asset_id = $1
                ORDER BY rp.created_at DESC, rp.id DESC`, assetID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var out []application.RelatedReport
        for rows.Next() {
                var rep application.RelatedReport
                if err := rows.Scan(&rep.ID, &rep.Code, &rep.Title, &rep.ReportDate, &rep.Status); err != nil {
                        return nil, err
                }
                out = append(out, rep)
        }
        return out, rows.Err()
}

