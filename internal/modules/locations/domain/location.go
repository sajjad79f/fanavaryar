// Package domain — مدل دامنه مکان‌ها و قواعد اختصاصی آن.
//
// این لایه هیچ وابستگی به دیتابیس یا HTTP ندارد؛ قواعد سلسله‌مراتب
// ساختمان ← طبقه ← اتاق ← رک دقیقاً همین‌جا تعریف و تست می‌شوند.
package domain

import (
	"errors"
	"strings"
	"time"
)

type Kind string

const (
	KindBuilding Kind = "building"
	KindFloor    Kind = "floor"
	KindRoom     Kind = "room"
	KindRack     Kind = "rack"
)

var ErrNotFound = errors.New("locations: not found")
var ErrParentMissing = errors.New("locations: parent not found")
var ErrHasChildren = errors.New("locations: has child locations")
var ErrHasAssets = errors.New("locations: has assigned assets")

var kindTitle = map[Kind]string{
	KindBuilding: "ساختمان",
	KindFloor:    "طبقه",
	KindRoom:     "اتاق",
	KindRack:     "رک",
}

// TitleFa نام نمایشی فارسی نوع مکان.
func (k Kind) TitleFa() string {
	if t, ok := kindTitle[k]; ok {
		return t
	}
	return string(k)
}

// CanNest قاعده سلسله‌مراتب: هر فرزند فقط زیر والدِ مجاز خودش مجاز است.
//
//	ساختمان ← طبقه ← اتاق ← رک
func CanNest(parent, child Kind) bool {
	switch parent {
	case KindBuilding:
		return child == KindFloor
	case KindFloor:
		return child == KindRoom
	case KindRoom:
		return child == KindRack
	default:
		return false
	}
}

// IsRoot ریشه‌های درخت فقط ساختمان‌اند (همچنین در Migration با CHECK تثبیت شده).
func (l *Location) IsRoot() bool { return l.ParentID == nil }

type Location struct {
	ID        int64     `json:"id"`
	ParentID  *int64    `json:"parentId,omitempty"`
	Kind      Kind      `json:"kind"`
	Name      string    `json:"name"`
	Code      *string   `json:"code,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate قواعد پایه هر مکان را بررسی می‌کند.
func (l *Location) Validate() error {
	name := strings.TrimSpace(l.Name)
	if name == "" {
		return errors.New("نام مکان الزامی است")
	}
	if len(name) > 120 {
		return errors.New("نام مکان حداکثر ۱۲۰ کاراکتر است")
	}
	l.Name = name

	switch l.Kind {
	case KindBuilding, KindFloor, KindRoom, KindRack:
	default:
		return errors.New("نوع مکان نامعتبر است")
	}

	if l.IsRoot() && l.Kind != KindBuilding {
		return errors.New("مکان ریشه فقط می‌تواند ساختمان باشد")
	}

	if l.Code != nil && len(*l.Code) > 30 {
		return errors.New("کد مکان حداکثر ۳۰ کاراکتر است")
	}
	if l.Notes != nil && len(*l.Notes) > 1000 {
		return errors.New("توضیحات حداکثر ۱۰۰۰ کاراکتر است")
	}
	return nil
}

// Node گره درخت مکان‌ها برای نمایش یکپارچه در فرانت‌اند.
type Node struct {
	Location
	Children []*Node `json:"children"`
}

// BuildTree از فهرست تختِ مکان‌ها، درخت می‌سازد.
// ترتیب گره‌های خروجی همان ترتیب ورودی است؛ فراخوان‌دهنده برای نمایش پایدار،
// ورودی مرتب‌شده می‌فرستد.
func BuildTree(all []Location) []*Node {
	nodes := make(map[int64]*Node, len(all))
	for i := range all {
		nodes[all[i].ID] = &Node{Location: all[i]}
	}

	var roots []*Node
	for i := range all {
		loc := all[i]
		if loc.ParentID == nil {
			roots = append(roots, nodes[loc.ID])
			continue
		}
		if parent, ok := nodes[*loc.ParentID]; ok {
			parent.Children = append(parent.Children, nodes[loc.ID])
		} else {
			// داده خراب: والد یافت نشد — به‌جای سکوت، ریشه مستقل می‌شود
			roots = append(roots, nodes[loc.ID])
		}
	}
	return roots
}
