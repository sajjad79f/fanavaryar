// Package httpx — ابزارهای مشترک HTTP: پاسخ JSON، پاکت خطا و دیکود ورودی.
//
// قرارداد خطای API نسخه ۱ (هم‌سان با فرانت‌اند فناوریار):
//
//      {"error": "پیام خطا به فارسی"}
//
// کلید error همیشه یک رشته است؛ فرانت‌اند مستقیماً آن را به کاربر نشان می‌دهد.
package httpx

import (
        "encoding/json"
        "errors"
        "fmt"
        "log/slog"
        "net"
        "net/http"
        "strconv"
)

// AppError خطای کنترل‌شده دامنه‌ای است؛ هر خطای دیگر به‌عنوان خطای سرور فرض می‌شود.
type AppError struct {
        Status  int
        Code    string
        Message string
        Err     error
}

func (e *AppError) Error() string {
        if e.Err != nil {
                return fmt.Sprintf("%s: %v", e.Message, e.Err)
        }
        return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(status int, code, message string) *AppError {
        return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(msg string) *AppError   { return New(http.StatusBadRequest, "bad_request", msg) }
func NotFound(msg string) *AppError     { return New(http.StatusNotFound, "not_found", msg) }
func Forbidden(msg string) *AppError    { return New(http.StatusForbidden, "forbidden", msg) }
func Unauthorized(msg string) *AppError { return New(http.StatusUnauthorized, "unauthorized", msg) }
func Conflict(msg string) *AppError     { return New(http.StatusConflict, "conflict", msg) }

func Internal(msg string, err error) *AppError {
        return &AppError{Status: http.StatusInternalServerError, Code: "internal",
                Message: msg, Err: err}
}

// JSON یک پاسخ JSON با وضعیت مشخص می‌نویسد.
func JSON(w http.ResponseWriter, status int, data any) {
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.WriteHeader(status)
        if data != nil {
                _ = json.NewEncoder(w).Encode(data)
        }
}

// Error خطا را به پاکت استاندارد v1 تبدیل و رندر می‌کند.
// پیام خطاهای داخلی به کلاینت درز داده نمی‌شود.
func Error(w http.ResponseWriter, err error) {
        var appErr *AppError
        if !errors.As(err, &appErr) {
                appErr = Internal("internal error", err)
        }
        if appErr.Status >= 500 {
                // خطای سرور در پاسخ عمومی مبهم می‌ماند؛ ولی جزئیات واقعی با سطح Error
                // در لاگ سامانه ثبت می‌شود تا عیب‌یابی روی سرور ممکن باشد.
                slog.Error("request failed with internal error",
                        "status", appErr.Status,
                        "code", appErr.Code,
                        "detail", appErr.Error())
                JSON(w, appErr.Status, map[string]string{"error": "یک خطای داخلی رخ داد؛ بعداً دوباره تلاش کنید"})
                return
        }
        JSON(w, appErr.Status, map[string]string{"error": appErr.Message})
}

// Decode بدنه JSON درخواست را محدود و می‌خواند.
// فیلدهای ناشناخته نادیده گرفته می‌شوند تا تغییرهای کوچک فرانت‌اند، درخواست را نکنند.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
        r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // سقف ۱ مگابایت
        dec := json.NewDecoder(r.Body)
        if err := dec.Decode(dst); err != nil {
                return BadRequest("بدنه درخواست معتبر نیست")
        }
        return nil
}

// ClientIP آدرس واقعی کلاینت پشت Nginx را برمی‌گرداند.
func ClientIP(r *http.Request) string {
        host, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
                return r.RemoteAddr
        }
        return host
}

// Page پارامترهای صفحه‌بندی مشترک همه فهرست‌ها (items/total/page/per_page).
type Page struct {
        Number  int
        PerPage int
}

// ReadPage خواندن page و per_page از Query؛ سقف per_page صد رکورد است.
func ReadPage(r *http.Request) Page {
        p := Page{Number: 1, PerPage: 20}
        if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
                p.Number = v
        }
        if v, err := strconv.Atoi(r.URL.Query().Get("per_page")); err == nil && v > 0 && v <= 100 {
                p.PerPage = v
        }
        return p
}

// Offset برای SQL.
func (p Page) Offset() int { return (p.Number - 1) * p.PerPage }

// Result خروجی استاندارد فهرست‌ها؛ T آرایه آیتم‌هاست.
type Result[T any] struct {
        Items   []T `json:"items"`
        Total   int `json:"total"`
        Page    int `json:"page"`
        PerPage int `json:"per_page"`
}

// NewResult ساخت پاسخ صفحه‌بندی‌شده؛ آرایه خالی همیشه [] است نه null.
func NewResult[T any](items []T, total int, p Page) Result[T] {
        if items == nil {
                items = []T{}
        }
        return Result[T]{Items: items, Total: total, Page: p.Number, PerPage: p.PerPage}
}
