// Package infrastructure — پیاده‌سازی PostgreSQL ماژول کاربران (قرارداد v1).
package infrastructure

import (
        "context"
        "errors"
        "strconv"
        "strings"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgconn"
        "github.com/jackc/pgx/v5/pgxpool"

        "fanavaryar/internal/modules/users/application"
        "fanavaryar/internal/modules/users/domain"
)

type UserRepository struct {
        pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
        return &UserRepository{pool: pool}
}

func mapUnique(err error) error {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
                return domain.ErrUsernameTaken
        }
        return err
}

const userCols = `id, username, full_name, job_title, role, is_active, created_at`

func scanUser(row pgx.Row) (application.UserItem, error) {
        var u application.UserItem
        err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.JobTitle,
                &u.Role, &u.IsActive, &u.CreatedAt)
        return u, err
}

// AuthRow ردیف کامل برای ورود — شامل هش رمز.
type AuthRow struct {
        ID           int64
        Username     string
        FullName     string
        JobTitle     string
        Role         string
        IsSuperadmin bool
        IsActive     bool
        PasswordHash string
        ScopeType    string
        ScopeRefID   *int64
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (application.AuthRow, error) {
        var a application.AuthRow
        err := r.pool.QueryRow(ctx, `
                SELECT id, username, full_name, COALESCE(job_title, ''),
                       role, is_superadmin, is_active, password_hash, scope_type, scope_ref_id
                FROM users WHERE lower(username) = lower($1)`, username).
                Scan(&a.ID, &a.Username, &a.FullName, &a.JobTitle,
                        &a.Role, &a.IsSuperadmin, &a.IsActive, &a.PasswordHash, &a.ScopeType, &a.ScopeRefID)
        if errors.Is(err, pgx.ErrNoRows) {
                return application.AuthRow{}, domain.ErrNotFound
        }
        return a, err
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
        var exists bool
        err := r.pool.QueryRow(ctx,
                `SELECT EXISTS (SELECT 1 FROM users WHERE lower(username) = lower($1))`, username).
                Scan(&exists)
        return exists, err
}

func (r *UserRepository) RoleOf(ctx context.Context, userID int64) (string, bool, error) {
        var role string
        var super bool
        err := r.pool.QueryRow(ctx,
                `SELECT role, is_superadmin FROM users WHERE id = $1`, userID).
                Scan(&role, &super)
        if errors.Is(err, pgx.ErrNoRows) {
                return "", false, domain.ErrNotFound
        }
        return role, super, err
}

func (r *UserRepository) Create(ctx context.Context, in application.CreateInput, hash string) (application.UserItem, error) {
        u, err := scanUser(r.pool.QueryRow(ctx, `
                INSERT INTO users (username, full_name, position, job_title, password_hash, is_active, role)
                VALUES ($1, $2, $3, $3, $4, true, 'user')
                RETURNING `+userCols,
                strings.ToLower(strings.TrimSpace(in.Username)), strings.TrimSpace(in.FullName),
                trimOrNull(in.JobTitle), hash))
        if err != nil {
                return application.UserItem{}, mapUnique(err)
        }
        return u, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id int64,
        fullName string, jobTitle *string, isActive *bool) (application.UserItem, error) {
        tag, err := r.pool.Exec(ctx, `
                UPDATE users SET
                        full_name   = $2,
                        position    = $3,
                        job_title   = $3,
                        is_active   = COALESCE($4, is_active),
                        updated_at  = now()
                WHERE id = $1`,
                id, strings.TrimSpace(fullName), jobTitle, isActive)
        if err != nil {
                return application.UserItem{}, err
        }
        if tag.RowsAffected() == 0 {
                return application.UserItem{}, domain.ErrNotFound
        }
        return r.GetByID(ctx, id)
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (application.UserItem, error) {
        return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (r *UserRepository) SetPasswordHash(ctx context.Context, id int64, hash string) error {
        tag, err := r.pool.Exec(ctx,
                `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
        if err != nil {
                return err
        }
        if tag.RowsAffected() == 0 {
                return domain.ErrNotFound
        }
        return nil
}

func (r *UserRepository) ScopeOf(ctx context.Context, id int64) (string, *int64, error) {
        var st string
        var ref *int64
        err := r.pool.QueryRow(ctx,
                `SELECT scope_type, scope_ref_id FROM users WHERE id = $1`, id).Scan(&st, &ref)
        if errors.Is(err, pgx.ErrNoRows) {
                return "", nil, domain.ErrNotFound
        }
        return st, ref, err
}

func (r *UserRepository) SetScope(ctx context.Context, id int64, scopeType string, refID *int64) error {
        tag, err := r.pool.Exec(ctx,
                `UPDATE users SET scope_type = $2, scope_ref_id = $3, updated_at = now() WHERE id = $1`,
                id, scopeType, refID)
        if err != nil {
                return err
        }
        if tag.RowsAffected() == 0 {
                return domain.ErrNotFound
        }
        return nil
}

func (r *UserRepository) List(ctx context.Context, q string, p application.Page) ([]application.UserItem, int, error) {
        where := ` WHERE (username ILIKE $1 OR full_name ILIKE $1)`
        args := []any{"%" + q + "%"}

        var total int
        if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users`+where, args...).Scan(&total); err != nil {
                return nil, 0, err
        }

        args = append(args, p.PerPage, p.Offset())
        rows, err := r.pool.Query(ctx,
                `SELECT `+userCols+` FROM users`+where+
                        ` ORDER BY id LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
        if err != nil {
                return nil, 0, err
        }
        defer rows.Close()

        var out []application.UserItem
        for rows.Next() {
                u, err := scanUser(rows)
                if err != nil {
                        return nil, 0, err
                }
                out = append(out, u)
        }
        return out, total, rows.Err()
}

func (r *UserRepository) PasswordHashOf(ctx context.Context, id int64) (string, error) {
        var h string
        err := r.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&h)
        if errors.Is(err, pgx.ErrNoRows) {
                return "", domain.ErrNotFound
        }
        return h, err
}

// ---------- مجوزها ----------

func (r *UserRepository) PermissionsOf(ctx context.Context, userID int64) ([]string, error) {
        rows, err := r.pool.Query(ctx, `
                SELECT DISTINCT p.module || '.' || p.action
                FROM user_permissions up
                JOIN permissions p ON p.id = up.permission_id
                WHERE up.user_id = $1
                ORDER BY 1`, userID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        out := []string{}
        for rows.Next() {
                var code string
                if err := rows.Scan(&code); err != nil {
                        return nil, err
                }
                out = append(out, code)
        }
        return out, rows.Err()
}

// CountValidCodes چند تا از کدهای داده‌شده در کاتالوگ وجود دارد.
func (r *UserRepository) CountValidCodes(ctx context.Context, codes []string) (int, error) {
        var n int
        err := r.pool.QueryRow(ctx,
                `SELECT count(*) FROM permissions WHERE module || '.' || action = ANY($1)`, codes).Scan(&n)
        return n, err
}

func (r *UserRepository) ReplacePermissions(ctx context.Context, userID int64, grantedBy int64, codes []string) error {
        return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
                if _, err := tx.Exec(ctx, `DELETE FROM user_permissions WHERE user_id = $1`, userID); err != nil {
                        return err
                }
                if len(codes) == 0 {
                        return nil
                }
                _, err := tx.Exec(ctx, `
                        INSERT INTO user_permissions (user_id, permission_id, granted_by)
                        SELECT $1, p.id, $2
                        FROM permissions p
                        WHERE p.module || '.' || p.action = ANY($3)
                        ON CONFLICT DO NOTHING`,
                        userID, grantedBy, codes)
                return err
        })
}

type permDefRow struct {
        ID     int64
        Module string
        Action string
        Title  string
        Sort   int
}

// Catalog کاتالوگ مجوزها با ترتیب پایدار (sort = شماره درون ماژول).
func (r *UserRepository) Catalog(ctx context.Context) ([]application.PermDef, error) {
        rows, err := r.pool.Query(ctx, `
                SELECT id, module, action, title_fa,
                       ROW_NUMBER() OVER (PARTITION BY module ORDER BY id) AS sort
                FROM permissions
                ORDER BY module, id`)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var out []application.PermDef
        for rows.Next() {
                var p permDefRow
                if err := rows.Scan(&p.ID, &p.Module, &p.Action, &p.Title, &p.Sort); err != nil {
                        return nil, err
                }
                out = append(out, application.PermDef{
                        Code:   p.Module + "." + p.Action,
                        Module: p.Module,
                        Action: p.Action,
                        Title:  p.Title,
                        Sort:   p.Sort,
                })
        }
        return out, rows.Err()
}

// ---------- الگوهای دسترسی ----------

func (r *UserRepository) Templates(ctx context.Context) ([]application.PermTemplate, error) {
        rows, err := r.pool.Query(ctx, `
                SELECT t.id, t.name, t.created_at,
                       COALESCE((
                           SELECT array_agg(p.module || '.' || p.action ORDER BY p.module, p.action)
                           FROM permission_template_items ti
                           JOIN permissions p ON p.id = ti.permission_id
                           WHERE ti.template_id = t.id
                       ), '{}')
                FROM permission_templates t
                ORDER BY t.id`)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var out []application.PermTemplate
        for rows.Next() {
                var t application.PermTemplate
                if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.Permissions); err != nil {
                        return nil, err
                }
                if t.Permissions == nil {
                        t.Permissions = []string{}
                }
                out = append(out, t)
        }
        return out, rows.Err()
}

func (r *UserRepository) CreateTemplate(ctx context.Context, name string, codes []string) (int64, error) {
        var id int64
        err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
                if err := tx.QueryRow(ctx,
                        `INSERT INTO permission_templates (name) VALUES ($1) RETURNING id`, name).
                        Scan(&id); err != nil {
                        return err
                }
                if len(codes) == 0 {
                        return nil
                }
                _, err := tx.Exec(ctx, `
                        INSERT INTO permission_template_items (template_id, permission_id)
                        SELECT $1, p.id FROM permissions p
                        WHERE p.module || '.' || p.action = ANY($2)`,
                        id, codes)
                return err
        })
        if err != nil {
                return 0, mapUnique(err)
        }
        return id, nil
}

func (r *UserRepository) DeleteTemplate(ctx context.Context, id int64) error {
        tag, err := r.pool.Exec(ctx, `DELETE FROM permission_templates WHERE id = $1`, id)
        if err != nil {
                return err
        }
        if tag.RowsAffected() == 0 {
                return domain.ErrNotFound
        }
        return nil
}

// ---------- ابزار ----------

func trimOrNull(s string) *string {
        t := strings.TrimSpace(s)
        if t == "" {
                return nil
        }
        return &t
}
