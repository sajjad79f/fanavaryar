// Package domain — مدل تجهیز و قواعد چرخه عمر آن (قرارداد v1).
package domain

import (
	"errors"
	"strings"
)

type Status string

// وضعیت‌های هم‌سان با فرانت‌اند: فعال، در تعمیر، انبار، از رده خارج.
const (
	StatusActive  Status = "active"
	StatusRepair  Status = "repair"
	StatusStorage Status = "storage"
	StatusRetired Status = "retired"
)

func (s Status) TitleFa() string {
	switch s {
	case StatusActive:
		return "فعال"
	case StatusRepair:
		return "در تعمیر"
	case StatusStorage:
		return "انبار"
	case StatusRetired:
		return "از رده خارج"
	}
	return string(s)
}

var ErrNotFound = errors.New("assets: not found")
var ErrTagTaken = errors.New("assets: asset tag already taken")

var allStatuses = map[Status]bool{
	StatusActive: true, StatusRepair: true, StatusStorage: true, StatusRetired: true,
}

// PhysicalKinds انواع مکانی که به‌عنوان «محل فیزیکی تجهیز» معتبرند.
// جابه‌جایی به ساختمان یا طبقه بدون اتاق/رک مجاز نیست تا آدرس تجهیز همیشه دقیق باشد.
func PhysicalKinds() []string { return []string{"room", "rack"} }

type Asset struct {
	ID                int64
	AssetTag          string
	Name              string
	Category          string
	Status            Status
	SerialNumber      *string
	Manufacturer      *string
	Model             *string
	ResponsiblePerson *string
	CurrentLocationID *int64
	LocationName      *string
	Notes             *string
	CreatedAt         string
	UpdatedAt         string
}

// Validate قواعد پایه پرونده تجهیز.
func (a *Asset) Validate() error {
	tag := strings.TrimSpace(a.AssetTag)
	if len(tag) < 2 || len(tag) > 40 {
		return errors.New("شناسه تجهیز باید ۲ تا ۴۰ کاراکتر باشد")
	}
	a.AssetTag = tag

	name := strings.TrimSpace(a.Name)
	if len(name) < 2 || len(name) > 120 {
		return errors.New("نام تجهیز باید ۲ تا ۱۲۰ کاراکتر باشد")
	}
	a.Name = name

	cat := strings.TrimSpace(a.Category)
	if cat == "" || len(cat) > 40 {
		return errors.New("دسته تجهیز الزامی است (حداکثر ۴۰ کاراکتر)")
	}
	a.Category = cat

	if !allStatuses[a.Status] {
		return errors.New("وضعیت تجهیز نامعتبر است")
	}

	if a.Notes != nil && len(*a.Notes) > 2000 {
		return errors.New("توضیحات حداکثر ۲۰۰۰ کاراکتر است")
	}
	return nil
}
