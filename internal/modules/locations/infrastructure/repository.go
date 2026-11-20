// Package infrastructure — پیاده‌سازی PostgreSQL ماژول مکان‌ها (قرارداد v1).
package infrastructure

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fanavaryar/internal/modules/locations/application"
	"fanavaryar/internal/modules/locations/domain"
)

type LocationRepository struct {
	pool *pgxpool.Pool
}

func NewLocationRepository(pool *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{pool: pool}
}

const locCols = `
	l.id, l.parent_id, l.kind, l.name, l.code, l.created_at,
	(SELECT count(*) FROM assets a WHERE a.current_location_id = l.id) AS assets_count
`

func scanRow(row pgx.Row) (application.LocationRow, error) {
	var r application.LocationRow
	var kind string
	err := row.Scan(&r.ID, &r.ParentID, &kind, &r.Name, &r.Code, &r.CreatedAt, &r.AssetsCount)
	r.Kind = kind
	return r, err
}

func (r *LocationRepository) List(ctx context.Context) ([]application.LocationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ` + locCols + `
		FROM locations l
		ORDER BY l.parent_id NULLS FIRST, l.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []application.LocationRow
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *LocationRepository) GetKind(ctx context.Context, id int64) (string, error) {
	var kind string
	err := r.pool.QueryRow(ctx, `SELECT kind FROM locations WHERE id = $1`, id).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return kind, err
}

func (r *LocationRepository) Create(ctx context.Context, in application.CreateInput) (application.LocationRow, error) {
	return scanRow(r.pool.QueryRow(ctx, `
		INSERT INTO locations (parent_id, kind, name, code)
		VALUES ($1, $2, $3, $4)
		RETURNING id, parent_id, kind, name, code, created_at`+`,
		0`, in.ParentID, in.Kind, in.Name, in.Code))
}

func (r *LocationRepository) Rename(ctx context.Context, id int64, name string, code *string) (application.LocationRow, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE locations SET name = $2, code = $3, updated_at = now() WHERE id = $1`,
		id, name, code)
	if err != nil {
		return application.LocationRow{}, err
	}
	if tag.RowsAffected() == 0 {
		return application.LocationRow{}, domain.ErrNotFound
	}
	return scanRow(r.pool.QueryRow(ctx, `SELECT `+locCols+` FROM locations l WHERE l.id = $1`, id))
}

func (r *LocationRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM locations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *LocationRepository) CountChildren(ctx context.Context, id int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM locations WHERE parent_id = $1`, id).Scan(&n)
	return n, err
}

func (r *LocationRepository) CountAssets(ctx context.Context, id int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM assets WHERE current_location_id = $1`, id).Scan(&n)
	return n, err
}

// GetSnapshot نوع و نام مکان — برای adapter بین‌ماژولی.
func (r *LocationRepository) GetSnapshot(ctx context.Context, id int64) (string, string, error) {
	var kind, name string
	err := r.pool.QueryRow(ctx, `SELECT kind, name FROM locations WHERE id = $1`, id).
		Scan(&kind, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", domain.ErrNotFound
	}
	return kind, name, err
}
