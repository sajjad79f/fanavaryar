// Package domain — مدل گزارش فعالیت (قرارداد v1).
// چرخه: pending → approved | rejected؛ گزارش بررسی‌شده تغییر نمی‌کند.
package domain

import (
        "errors"
        "strings"
)

type Status string

const (
        StatusPending  Status = "pending"
        StatusApproved Status = "approved"
        StatusRejected Status = "rejected"
)

func (s Status) TitleFa() string {
        switch s {
        case StatusPending:
                return "در انتظار تایید"
        case StatusApproved:
                return "تایید شده"
        case StatusRejected:
                return "رد شده"
        }
        return string(s)
}

var ErrNotFound = errors.New("reports: not found")
var ErrNotPending = errors.New("reports: not pending")
var ErrSelfApproval = errors.New("reports: self approval")

var allStatuses = map[Status]bool{StatusPending: true, StatusApproved: true, StatusRejected: true}

func ValidStatus(s string) bool { return allStatuses[Status(s)] }

// ValidateTitle قواعد عنوان گزارش.
func ValidateTitle(title string) error {
        t := strings.TrimSpace(title)
        if len([]rune(t)) < 3 || len([]rune(t)) > 150 {
                return errors.New("عنوان گزارش باید ۳ تا ۱۵۰ کاراکتر باشد")
        }
        return nil
}
