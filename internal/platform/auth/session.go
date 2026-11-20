// SessionService — نشست‌های سمت سرور در PostgreSQL.
//
// توکن نشست فقط در کوکی HttpOnly نگه‌داری می‌شود و در دیتابیس
// فقط SHA-256 آن ذخیره می‌گردد (نه خود توکن). غیرفعال‌سازی حساب،
// همه نشست‌های آن کاربر را باطل می‌کند (طبق سند بخش ۵).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SessionCookieName = "fanavaryar_session"

var ErrSessionNotFound = errors.New("auth: session not found")

type Session struct {
	UserID    int64
	Username  string
	ExpiresAt time.Time
}

type SessionService struct {
	pool    *pgxpool.Pool
	Timeout time.Duration
}

func NewSessionService(pool *pgxpool.Pool) *SessionService {
	return &SessionService{pool: pool}
}

// Create یک نشست جدید می‌سازد و توکن خام را برمی‌گرداند (فقط برای کوکی).
func (s *SessionService) Create(ctx context.Context, userID int64, ttl time.Duration) (token string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("auth: session token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	hash := tokenHash(token)

	_, err = s.pool.Exec(ctx, `
                INSERT INTO sessions (token_hash, user_id, expires_at)
                VALUES ($1, $2, $3)`,
		hash, userID, time.Now().Add(ttl),
	)
	if err != nil {
		return "", fmt.Errorf("auth: insert session: %w", err)
	}
	return token, nil
}

// Get نشست معتبر را برمی‌گرداند؛ نشست منقضی به‌صورت یافت‌نشده تلقی می‌شود.
func (s *SessionService) Get(ctx context.Context, token string) (Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx, `
                SELECT s.user_id, u.username, s.expires_at
                FROM sessions s
                JOIN users u ON u.id = s.user_id
                WHERE s.token_hash = $1 AND u.is_active`,
		tokenHash(token),
	).Scan(&sess.UserID, &sess.Username, &sess.ExpiresAt)
	if errors.Is(err, context.Canceled) {
		return Session{}, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		// خطای واقعی دیتابیس را نباید به «نشست نامعتبر» ترجمه کرد.
		return Session{}, fmt.Errorf("auth: load session: %w", err)
	}
	if time.Now().After(sess.ExpiresAt) {
		return Session{}, ErrSessionNotFound
	}
	return sess, nil
}

// Revoke یک نشست را حذف می‌کند (خروج کاربر).
func (s *SessionService) Revoke(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM sessions WHERE token_hash = $1`, tokenHash(token))
	return err
}

// RevokeAllForUser هنگام غیرفعال‌سازی حساب یا تغییر مجوزهای حساس استفاده می‌شود.
func (s *SessionService) RevokeAllForUser(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

// CleanupExpired نشست‌های منقضی را پاک می‌کند (فراخوانی دوره‌ای).
func (s *SessionService) CleanupExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM sessions WHERE expires_at < now()`)
	return err
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
