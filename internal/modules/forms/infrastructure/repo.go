// Package infrastructure — پیاده‌سازی PostgreSQL ماژول فرم‌ها (قرارداد v1).
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fanavaryar/internal/modules/forms/application"
	"fanavaryar/internal/modules/forms/domain"
)

type FormRepository struct {
	pool *pgxpool.Pool
}

func NewFormRepository(pool *pgxpool.Pool) *FormRepository {
	return &FormRepository{pool: pool}
}

func mapUnique(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrCodeTaken
	}
	return err
}

const tplCols = `t.id, t.code, t.name, t.description, t.is_active, t.current_version_id, t.created_at`

func scanTemplate(row pgx.Row) (application.FormTemplate, error) {
	var t application.FormTemplate
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.Description,
		&t.IsActive, &t.CurrentVersionID, &t.CreatedAt)
	return t, err
}

func scanVersion(row pgx.Row) (application.FormVersion, error) {
	var v application.FormVersion
	var raw []byte
	err := row.Scan(&v.ID, &v.TemplateID, &v.VersionNumber, &raw, &v.CreatedByName, &v.CreatedAt)
	if err != nil {
		return application.FormVersion{}, err
	}
	fields, err := domain.ParseFields(raw)
	if err != nil {
		return application.FormVersion{}, err
	}
	if fields == nil {
		fields = []domain.FieldDef{}
	}
	v.Fields = fields
	return v, nil
}

func (r *FormRepository) List(ctx context.Context) ([]application.FormTemplate, error) {
	rows, err := r.pool.Query(ctx, `SELECT ` + tplCols + ` FROM form_templates t ORDER BY t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []application.FormTemplate
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *FormRepository) GetByID(ctx context.Context, id int64) (application.FormTemplate, error) {
	t, err := scanTemplate(r.pool.QueryRow(ctx,
		`SELECT `+tplCols+` FROM form_templates t WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.FormTemplate{}, domain.ErrNotFound
	}
	return t, err
}

// Create قالب + نسخه ۱ در یک تراکنش؛ نسخه‌ها تغییرناپذیرند.
func (r *FormRepository) Create(ctx context.Context, in application.CreateInput,
	fields []domain.FieldDef, actorID *int64) (int64, error) {

	raw, err := json.Marshal(fields)
	if err != nil {
		return 0, err
	}

	var id int64
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO form_templates (code, name, description, created_by)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
			in.Code, in.Name, in.Description, actorID).Scan(&id); err != nil {
			return mapUnique(err)
		}
		var vid int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO form_versions (template_id, version_number, fields, created_by)
			VALUES ($1, 1, $2, $3)
			RETURNING id`,
			id, raw, actorID).Scan(&vid); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE form_templates SET current_version_id = $2 WHERE id = $1`, id, vid)
		return err
	})
	if err != nil {
		return 0, mapUnique(err)
	}
	return id, nil
}

func (r *FormRepository) SetActive(ctx context.Context, id int64, active bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE form_templates SET is_active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FormRepository) CreateVersion(ctx context.Context, templateID int64,
	fields []domain.FieldDef, actorID *int64) (int64, int, error) {

	raw, err := json.Marshal(fields)
	if err != nil {
		return 0, 0, err
	}

	var vid int64
	var ver int
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// قالب باید موجود باشد
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM form_templates WHERE id = $1)`, templateID).
			Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return domain.ErrNotFound
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO form_versions (template_id, version_number, fields, created_by)
			VALUES ($1, (SELECT COALESCE(MAX(version_number), 0) + 1 FROM form_versions WHERE template_id = $1), $2, $3)
			RETURNING id, version_number`,
			templateID, raw, actorID).Scan(&vid, &ver); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE form_templates SET current_version_id = $2 WHERE id = $1`, templateID, vid)
		return err
	})
	if err != nil {
		return 0, 0, err
	}
	return vid, ver, nil
}

func (r *FormRepository) VersionByID(ctx context.Context, id int64) (application.FormVersion, error) {
	v, err := scanVersion(r.pool.QueryRow(ctx, `
		SELECT v.id, v.template_id, v.version_number, v.fields, u.full_name, v.created_at
		FROM form_versions v
		LEFT JOIN users u ON u.id = v.created_by
		WHERE v.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.FormVersion{}, domain.ErrNotFound
	}
	return v, err
}

func (r *FormRepository) ListVersions(ctx context.Context, templateID int64) ([]application.FormVersion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.template_id, v.version_number, v.fields, u.full_name, v.created_at
		FROM form_versions v
		LEFT JOIN users u ON u.id = v.created_by
		WHERE v.template_id = $1
		ORDER BY v.version_number DESC`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []application.FormVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
